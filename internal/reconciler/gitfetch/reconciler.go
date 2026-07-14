// Package gitfetch reconciles git repository data using per-project
// `git fetch --prune +refs/*:refs/*` instead of filesystem rsync. This
// mode is used when the primary's filesystem is not directly accessible
// (e.g. different cloud provider). It pulls the project list from the
// local replicated PostgreSQL DB (which is already 1:1 via WAL streaming)
// so no GitLab API calls are needed.
package gitfetch

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/anomalyco/gitlab-geo-sync/internal/metrics"
	"github.com/anomalyco/gitlab-geo-sync/internal/reconciler"
)

const name = "git_fetch"

// Reconciler fetches all project repos from the primary's Gitaly via SSH.
type Reconciler struct {
	primarySSHHost string
	reposPath      string
	secondaryName  string
	primaryPool    *pgxpool.Pool
	dryRun         bool
	maxParallel    int
}

// New creates a git fetch reconciler.
func New(primarySSHHost, reposPath, secondaryName string, primaryPool *pgxpool.Pool, dryRun bool) *Reconciler {
	return &Reconciler{
		primarySSHHost: primarySSHHost,
		reposPath:      reposPath,
		secondaryName:  secondaryName,
		primaryPool:    primaryPool,
		dryRun:         dryRun,
		maxParallel:    8,
	}
}

func (r *Reconciler) Name() string { return name }

// Reconcile queries the DB for all project repository paths, then runs
// `git fetch --prune` on each local repo against the primary's SSH URL.
func (r *Reconciler) Reconcile(ctx context.Context) reconciler.Result {
	start := time.Now()

	projects, err := r.listProjects(ctx)
	if err != nil {
		metrics.DriftTotal.WithLabelValues(name, "critical").Inc()
		return reconciler.Result{OK: false, Detail: fmt.Sprintf("list projects: %v", err), Remaining: 1}
	}

	if len(projects) == 0 {
		return reconciler.Result{OK: true, Detail: "no projects to sync"}
	}

	failed := 0
	repaired := 0
	for _, p := range projects {
		select {
		case <-ctx.Done():
			return reconciler.Result{OK: false, Detail: "cancelled", Remaining: len(projects)}
		default:
		}
		if r.fetchOne(ctx, p) {
			repaired++
		} else {
			failed++
		}
	}

	elapsed := time.Since(start)
	resultStr := "ok"
	if failed > 0 {
		resultStr = "error"
	}
	metrics.SyncDurationSeconds.WithLabelValues(name, resultStr).Observe(elapsed.Seconds())

	if failed > 0 {
		metrics.DriftTotal.WithLabelValues(name, "warning").Inc()
		return reconciler.Result{
			OK:        false,
			Detail:    fmt.Sprintf("fetched %d/%d projects in %s (%d failed)", repaired, len(projects), elapsed, failed),
			Repaired:  repaired,
			Remaining: failed,
		}
	}
	metrics.LastSyncTimestamp.WithLabelValues(name).Set(float64(time.Now().Unix()))
	return reconciler.Result{
		OK:       true,
		Detail:   fmt.Sprintf("fetched %d projects in %s", len(projects), elapsed),
		Repaired: repaired,
	}
}

// projectRow holds the minimal fields needed to locate and fetch a repo.
type projectRow struct {
	ID            int32
	RepoPath      string // e.g. "group/subgroup/project.git"
	HashedPath    string // hashed storage path (if enabled)
}

// listProjects queries the replicated DB for all project repository paths.
// These are public GitLab CE schema columns.
func (r *Reconciler) listProjects(ctx context.Context) ([]projectRow, error) {
	rows, err := r.primaryPool.Query(ctx, `
		SELECT id,
		       repository_storage,
		       CASE
		         WHEN repository_storage IS NOT NULL
		           THEN COALESCE(
		             NULLIF(
		               regexp_replace(
		                 CASE WHEN disk_path IS NOT NULL AND disk_path != ''
		                      THEN disk_path
		                      ELSE CONCAT(namespace::text, '/', path::text, '.git')
		                 END, '^/', ''), ''),
		             repository_storage || '/' ||
		             CASE WHEN disk_path IS NOT NULL AND disk_path != ''
		               THEN disk_path
		               ELSE CONCAT(namespace::text, '/', path::text, '.git')
		             END)
		       END
		FROM projects
		WHERE repository_storage IS NOT NULL
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []projectRow
	for rows.Next() {
		var p projectRow
		var id int32
		var repoPath *string
		if err := rows.Scan(&id, &repoPath); err != nil {
			return nil, err
		}
		p.ID = id
		if repoPath != nil {
			p.RepoPath = *repoPath
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// fetchOne runs `git fetch --prune +refs/*:refs/* --no-tags +refs/tags/*:refs/tags/*`
// on a single local repo against the primary SSH URL. Returns true on
// success, false on failure.
func (r *Reconciler) fetchOne(ctx context.Context, p projectRow) bool {
	if p.RepoPath == "" {
		return false
	}

	localPath := filepath.Join(r.reposPath, p.RepoPath)
	remoteURL := fmt.Sprintf("ssh://%s/var/opt/gitlab/git-data/repositories/%s",
		r.primarySSHHost, p.RepoPath)

	args := []string{
		"-C", localPath,
		"fetch", "--prune", "--no-tags",
		remoteURL,
		"+refs/*:refs/*",
		"+refs/tags/*:refs/tags/*",
	}

	if r.dryRun {
		return true
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = out
		return false
	}
	return true
}

// maxParallel is exposed for future concurrent fetching; currently sequential.
func (r *Reconciler) SetMaxParallel(n int) { r.maxParallel = n }

// _ keeps strings imported if we trim usage later.
var _ = strings.TrimSpace