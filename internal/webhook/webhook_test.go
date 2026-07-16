package webhook

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestExtractProjectPath(t *testing.T) {
	body := []byte(`{"project":{"path_with_namespace":"group/subgroup/project"}}`)
	path, err := extractProjectPath(body)
	if err != nil {
		t.Fatal(err)
	}
	if path != "group/subgroup/project" {
		t.Errorf("got %q", path)
	}
}

func TestExtractProjectPathMissing(t *testing.T) {
	body := []byte(`{"project":{}}`)
	_, err := extractProjectPath(body)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestHandleWebhookRejectsInvalidToken(t *testing.T) {
	s := NewServer(":0", "secret", func(context.Context, string, string) error { return nil })
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-Gitlab-Token", "wrong")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", w.Code)
	}
}

func TestHandleWebhookRejectsGet(t *testing.T) {
	s := NewServer(":0", "secret", func(context.Context, string, string) error { return nil })
	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", w.Code)
	}
}

func TestHandleWebhookAcceptsValid(t *testing.T) {
	called := make(chan string, 1)
	s := NewServer(":0", "secret", func(ctx context.Context, p, e string) error {
		called <- p
		return nil
	})
	body := []byte(`{"project":{"path_with_namespace":"group/proj"}}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Gitlab-Token", "secret")
	req.Header.Set("X-Gitlab-Event", "Push Hook")
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200", w.Code)
	}
	select {
	case p := <-called:
		if p != "group/proj" {
			t.Errorf("trigger got %q", p)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("trigger not called within 10s (debounce delay)")
	}
}

func TestTriggerManagerDebounces(t *testing.T) {
	var callCount int32
	mgr := NewTriggerManager(func(ctx context.Context, p, e string) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	go mgr.Trigger(context.Background(), "proj", "push")
	go mgr.Trigger(context.Background(), "proj", "push")

	time.Sleep(4 * time.Second)
	count := atomic.LoadInt32(&callCount)
	if count != 1 {
		t.Errorf("expected 1 call after debounce, got %d", count)
	}
}
