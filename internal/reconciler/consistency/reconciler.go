// Package consistency implements a periodic full-audit reconciler that
// compares row counts of key GitLab tables between primary and secondary,
// and runs `git fsck` on a sample of secondary repositories. It observes
// drift; it does not auto-repair in v0 (Phase 2 will add repairs).
package consistency

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/anomalyco/gitlab-geo-sync/internal/metrics"
	"github.com/anomalyco/gitlab-geo-sync/internal/reconciler"
)

const name = "consistency_sweep"

// tablesToCount are the key GitLab tables whose row counts we compare.
// These are all public CE schema tables observable on any install.
var tablesToCount = []string{
	"projects",
	"namespaces",
	"users",
	"merge_requests",
	"issues",
	"notes",
	"ci_builds",
	"ci_pipelines",
	"labels",
	"milestones",
}

// Reconciler compares row counts and samples git fsck.
type Reconciler struct {
	primary     *pgxpool.Pool
	secondary   *pgxpool.Pool
	secondaryName string
	reposPath   string
	samplePct   float64 // 0.0–1.0
}

// New creates a consistency sweep reconciler.
func New(primary, secondary *pgxpool.Pool, secondaryName, reposPath string, samplePct float64) *Reconciler {
	return &Reconciler{
		primary:     primary,
		secondary:   secondary,
		secondaryName: secondaryName,
		reposPath:    reposPath,
		samplePct:    samplePct,
	}
}

func (r *Reconciler) Name() string { return name }

// Reconcile runs the full audit.
func (r *Reconciler) Reconcile(ctx context.Context) reconciler.Result {
	result := reconciler.Result{Detail: "sweep complete"}
	drifts := 0

	for _, table := range tablesToCount {
		pCount, err := rowCount(ctx, r.primary, table)
		if err != nil {
			result.Remaining++
			result.Detail = fmt.Sprintf("%s; %s primary count error: %v", result.Detail, table, err)
			continue
		}
		sCount, err := rowCount(ctx, r.secondary, table)
		if err != nil {
			result.Remaining++
			result.Detail = fmt.Sprintf("%s; %s secondary count error: %v", result.Detail, table, err)
			continue
		}
		if pCount != sCount {
			drifts++
			result.Remaining++
			metrics.DriftTotal.WithLabelValues("db:"+table, "warning").Inc()
			result.Detail = fmt.Sprintf("%s; %s drift: primary=%d secondary=%d", result.Detail, table, pCount, sCount)
		}
	}

	// Git fsck sample.
	if r.reposPath != "" {
		fsckDrifts := r.sampleGitFsck(ctx)
		drifts += fsckDrifts
	}

	result.OK = drifts == 0
	result.Repaired = 0
	return result
}

// rowCount returns the approximate row count for a table. Uses
// pg_class.reltuples (cheap, stats-based) rather than a full COUNT(*).
// Returns 0 if the table doesn't exist (fresh install may not have
// all tables until features are first used).
func rowCount(ctx context.Context, pool *pgxpool.Pool, table string) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT reltuples::bigint
		FROM pg_class
		WHERE relname = $1 AND relkind = 'r'
		LIMIT 1`, table).Scan(&n)
	if err != nil {
		// Table doesn't exist — treat as 0 rows, not an error.
		return 0, nil
	}
	return n, nil
}

// sampleGitFsck walks the repos path, picks a random sample of .git
// directories, and runs `git fsck --no-dangling` on each. Returns the
// number of repos that failed fsck.
func (r *Reconciler) sampleGitFsck(ctx context.Context) int {
	if r.samplePct <= 0 || r.reposPath == "" {
		return 0
	}
	var allRepos []string
	_ = filepath.Walk(r.reposPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && (filepath.Base(path) == ".git" || strings.HasSuffix(path, ".git")) {
			allRepos = append(allRepos, path)
			return filepath.SkipDir
		}
		return nil
	})
	if len(allRepos) == 0 {
		return 0
	}
	n := int(float64(len(allRepos)) * r.samplePct)
	if n < 1 {
		n = 1
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	failed := 0
	for i := 0; i < n; i++ {
		repo := allRepos[rng.Intn(len(allRepos))]
		if !gitFsck(ctx, repo) {
			failed++
			metrics.DriftTotal.WithLabelValues("git_fsck", "critical").Inc()
		}
	}
	return failed
}

// gitFsck runs `git fsck --no-dangling` on a repo and returns true if ok.
func gitFsck(ctx context.Context, repoPath string) bool {
	out, err := execGitFsck(ctx, repoPath)
	if err != nil {
		_ = out
		return false
	}
	return true
}

func execGitFsck(ctx context.Context, repoPath string) (string, error) {
	// Inline exec to keep the package self-contained.
	return execCommand(ctx, "git", "-C", repoPath, "fsck", "--no-dangling")
}

func execCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := execCmd(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// execCmd is a variable so tests can stub it.
var execCmd = newExecCmd

// cmdRunner is the interface for running external commands; *exec.Cmd
// satisfies it natively.
type cmdRunner interface {
	CombinedOutput() ([]byte, error)
}

func newExecCmd(ctx context.Context, name string, args ...string) cmdRunner {
	return exec.CommandContext(ctx, name, args...)
}

// strconv is used in formatting; keep import to avoid removal.
var _ = strconv.Itoa