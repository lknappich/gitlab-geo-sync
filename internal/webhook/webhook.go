// Package webhook implements an HTTP receiver for GitLab push/create/delete
// webhooks. When a webhook arrives, it triggers an immediate per-project
// git sync for the affected project, bypassing the normal sweep interval.
// This reduces lag to near-real-time for hot paths while periodic sweeps
// remain the safety net.
package webhook

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/anomalyco/gitlab-geo-sync/internal/metrics"
)

// Server receives GitLab webhooks and triggers immediate sync.
type Server struct {
	addr        string
	secretToken string
	trigger     TriggerFunc
	mux         *http.ServeMux
	srv         *http.Server
}

// TriggerFunc is called when a webhook is received. It receives the
// project path (e.g. "group/subgroup/project") and the event type.
// Implementations should run git fetch for the specific project.
type TriggerFunc func(ctx context.Context, projectPath, eventType string) error

// NewServer creates a webhook receiver.
func NewServer(addr, secretToken string, trigger TriggerFunc) *Server {
	s := &Server{
		addr:        addr,
		secretToken: secretToken,
		trigger:     trigger,
		mux:         http.NewServeMux(),
	}
	s.mux.HandleFunc("/webhook", s.handleWebhook)
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	return s
}

// Start blocks until ctx is cancelled or the server errors.
func (s *Server) Start(ctx context.Context) error {
	s.srv = &http.Server{Addr: s.addr, Handler: s.mux}
	errCh := make(chan error, 1)
	go func() { errCh <- s.srv.ListenAndServe() }()
	log.Info().Str("addr", s.addr).Msg("webhook server listening")
	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.srv.Shutdown(shutCtx)
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

// handleWebhook validates the GitLab webhook token and parses the payload.
func (s *Server) handleWebhook(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// GitLab sends the secret token in the X-Gitlab-Token header.
	token := req.Header.Get("X-Gitlab-Token")
	if !hmac.Equal([]byte(token), []byte(s.secretToken)) {
		metrics.DriftTotal.WithLabelValues("webhook", "critical").Inc()
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	eventType := req.Header.Get("X-Gitlab-Event")
	body, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	projectPath, err := extractProjectPath(body)
	if err != nil {
		log.Warn().Err(err).Str("event", eventType).Msg("webhook: extract project path")
		// Still 200 so GitLab doesn't retry indefinitely.
		w.WriteHeader(http.StatusOK)
		return
	}

	// Trigger sync asynchronously so we respond to GitLab quickly.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		start := time.Now()
		if err := s.trigger(ctx, projectPath, eventType); err != nil {
			log.Error().Err(err).Str("project", projectPath).Str("event", eventType).
				Msg("webhook-triggered sync failed")
			metrics.SyncDurationSeconds.WithLabelValues("webhook_trigger", "error").
				Observe(time.Since(start).Seconds())
			return
		}
		metrics.SyncDurationSeconds.WithLabelValues("webhook_trigger", "ok").
			Observe(time.Since(start).Seconds())
		metrics.LastSyncTimestamp.WithLabelValues("webhook_trigger").
			Set(float64(time.Now().Unix()))
		log.Info().Str("project", projectPath).Str("event", eventType).
			Msg("webhook-triggered sync complete")
	}()

	w.WriteHeader(http.StatusOK)
}

// extractProjectPath pulls the project path from the webhook payload.
// GitLab webhooks include a "project" object with "path_with_namespace"
// in push, tag, and most system webhooks.
func extractProjectPath(body []byte) (string, error) {
	var payload struct {
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("parse webhook payload: %w", err)
	}
	if payload.Project.PathWithNamespace == "" {
		return "", fmt.Errorf("no project.path_with_namespace in payload")
	}
	return payload.Project.PathWithNamespace, nil
}

// TriggerManager debounces rapid-fire webhooks for the same project so
// we don't run multiple concurrent fetches for the same repo during a
// push burst.
type TriggerManager struct {
	mu       sync.Mutex
	pending  map[string]context.CancelFunc
	trigger  TriggerFunc
}

// NewTriggerManager wraps a TriggerFunc with per-project debouncing.
func NewTriggerManager(trigger TriggerFunc) *TriggerManager {
	return &TriggerManager{
		pending: map[string]context.CancelFunc{},
		trigger: trigger,
	}
}

// Trigger debounces: if a sync for this project is already pending,
// cancels it and starts a new one after a short delay.
func (m *TriggerManager) Trigger(ctx context.Context, projectPath, eventType string) error {
	m.mu.Lock()
	if cancel, ok := m.pending[projectPath]; ok {
		cancel()
	}
	ctx2, cancel := context.WithCancel(ctx)
	m.pending[projectPath] = cancel
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.pending, projectPath)
		m.mu.Unlock()
	}()

	// Debounce: wait 2 seconds for burst coalescing, then fetch.
	select {
	case <-ctx2.Done():
		return ctx2.Err()
	case <-time.After(2 * time.Second):
	}

	return m.trigger(ctx2, projectPath, eventType)
}

// hex is used for potential HMAC-SHA256 verification (GitLab also supports
// a token-based approach which we use above; this is kept for reference).
var _ = strings.TrimSpace