package autorepair

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitRepairDryRun(t *testing.T) {
	r := NewGitRepair("primary:22", "/repos", true)
	err := r.RepairRepo(context.Background(), "group/proj.git")
	if err != nil {
		t.Fatalf("dry-run should not error: %v", err)
	}
}

func TestGitRepairMissingRepo(t *testing.T) {
	dir := t.TempDir()
	r := NewGitRepair("primary:22", dir, false)
	err := r.RepairRepo(context.Background(), "nonexistent/repo.git")
	if err == nil {
		t.Fatal("expected error for missing repo")
	}
}

func TestS3RepairDryRun(t *testing.T) {
	r := NewS3Repair(true)
	err := r.RepairObject(context.Background(), "bucket", "key/to/object")
	if err != nil {
		t.Fatalf("dry-run should not error: %v", err)
	}
}

func TestS3RepairNonDryRunLogsOnly(t *testing.T) {
	r := NewS3Repair(false)
	err := r.RepairObject(context.Background(), "bucket", "key/to/object")
	if err != nil {
		t.Fatalf("non-dry-run should not error (logs only): %v", err)
	}
}

func TestGitRepairWithFakeRepo(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "group", "proj.git")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatal(err)
	}
	// We're testing that it attempts to run git fetch; it will fail
	// because there's no remote, but it should attempt it (not dry-run).
	r := NewGitRepair("nonexistent:22", dir, false)
	err := r.RepairRepo(context.Background(), "group/proj.git")
	if err == nil {
		t.Log("git fetch succeeded unexpectedly (maybe git not installed)")
	}
	// Error is expected; we just verify it doesn't panic.
}

type mockAutoRunner struct {
	out   []byte
	err   error
	calls int
}

func (m *mockAutoRunner) Run(ctx context.Context, name string, args, env []string) ([]byte, error) {
	m.calls++
	return m.out, m.err
}

func TestGitRepairSuccess(t *testing.T) {
	runner := &mockAutoRunner{out: []byte("")}
	g := NewGitRepair("p:22", "/repos", false).WithRunner(runner)
	err := g.RepairRepo(context.Background(), "group/proj.git")
	if err != nil {
		t.Fatalf("RepairRepo: %v", err)
	}
	if runner.calls != 1 {
		t.Errorf("calls = %d, want 1", runner.calls)
	}
}

func TestGitRepairFailure(t *testing.T) {
	runner := &mockAutoRunner{err: errors.New("fetch failed"), out: []byte("error output")}
	g := NewGitRepair("p:22", "/repos", false).WithRunner(runner)
	err := g.RepairRepo(context.Background(), "group/proj.git")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "fetch failed") {
		t.Errorf("err = %v", err)
	}
}
