package consistency

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSampleGitFsckEmptyPath(t *testing.T) {
	r := &Reconciler{reposPath: "", samplePct: 0.1}
	if n := r.sampleGitFsck(context.Background()); n != 0 {
		t.Errorf("expected 0 failures on empty path, got %d", n)
	}
}

func TestSampleGitFsckNoRepos(t *testing.T) {
	dir := t.TempDir()
	r := &Reconciler{reposPath: dir, samplePct: 1.0}
	if n := r.sampleGitFsck(context.Background()); n != 0 {
		t.Errorf("expected 0 failures on dir with no repos, got %d", n)
	}
}

func TestSampleGitFsckFindsGitDirs(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "group", "project.git")
	if err := os.MkdirAll(filepath.Join(repoDir, "objects"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Override exec to always succeed (we're testing discovery, not real git).
	oldExec := execCmd
	execCmd = func(ctx context.Context, name string, args ...string) cmdRunner {
		return &fakeCmd{out: []byte("ok"), err: nil}
	}
	t.Cleanup(func() { execCmd = oldExec })

	r := &Reconciler{reposPath: dir, samplePct: 1.0}
	n := r.sampleGitFsck(context.Background())
	if n != 0 {
		t.Errorf("expected 0 fsck failures with stub, got %d", n)
	}
}

type fakeCmd struct {
	out []byte
	err error
}

func (f *fakeCmd) CombinedOutput() ([]byte, error) { return f.out, f.err }
