package gitfetch

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestRepoDiskPathLegacy(t *testing.T) {
	got := repoDiskPath("default", false, "group/subgroup/project", 42)
	want := "group/subgroup/project.git"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRepoDiskPathLegacyEmptyRoute(t *testing.T) {
	got := repoDiskPath("default", false, "", 42)
	if got != "" {
		t.Errorf("expected empty path for empty route, got %q", got)
	}
}

func TestRepoDiskPathHashed(t *testing.T) {
	h := sha1.New()
	fmt.Fprintf(h, "%d", 42)
	full := hex.EncodeToString(h.Sum(nil))
	got := repoDiskPath("default", true, "ignored", 42)
	want := fmt.Sprintf("@hashed/%s/%s.git", full[:2], full)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSha1HexDeterministic(t *testing.T) {
	h1 := sha1Hex(1)
	h2 := sha1Hex(1)
	if h1 != h2 {
		t.Errorf("sha1Hex not deterministic: %q vs %q", h1, h2)
	}
	if len(h1) != 40 {
		t.Errorf("expected 40-char hex, got %d", len(h1))
	}
}

func TestFetchProjectEmptyPath(t *testing.T) {
	r := &Reconciler{}
	err := r.FetchProject(nil, "")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestFetchProjectRejectsTraversal(t *testing.T) {
	r := &Reconciler{}
	err := r.FetchProject(nil, "../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for traversal path")
	}
}
