package registry

import (
	"testing"

	"github.com/lknappich/gitlab-geo-sync/internal/config"
)

func TestToSet(t *testing.T) {
	s := toSet([]string{"a", "b", "c"})
	if len(s) != 3 || !s["a"] || !s["b"] || !s["c"] {
		t.Errorf("toSet got %v", s)
	}
}

func TestSetDiff(t *testing.T) {
	a := toSet([]string{"a", "b", "c"})
	b := toSet([]string{"b", "c"})
	diff := setDiff(a, b)
	if len(diff) != 1 || diff[0] != "a" {
		t.Errorf("setDiff got %v", diff)
	}
}

func TestSetDiffEmpty(t *testing.T) {
	a := toSet([]string{"a", "b"})
	b := toSet([]string{"a", "b"})
	diff := setDiff(a, b)
	if len(diff) != 0 {
		t.Errorf("setDiff got %v, want empty", diff)
	}
}

func TestEqualSet(t *testing.T) {
	a := toSet([]string{"a", "b"})
	b := toSet([]string{"a", "b"})
	if !equalSet(a, b) {
		t.Error("equalSet should be true")
	}
	c := toSet([]string{"a"})
	if equalSet(a, c) {
		t.Error("equalSet should be false for different sizes")
	}
	d := toSet([]string{"a", "c"})
	if equalSet(a, d) {
		t.Error("equalSet should be false for different keys")
	}
}

func TestIsAuthError(t *testing.T) {
	if isAuthError(nil) {
		t.Error("nil should not be auth error")
	}
	if !isAuthError(errAuthRequired) {
		t.Error("errAuthRequired should be auth error")
	}
	if isAuthError(errAuthRequiredAlternate) {
		t.Error("different error should not be auth error")
	}
}

var errAuthRequiredAlternate = errStringOther("some other error")

type errStringOther string

func (e errStringOther) Error() string { return string(e) }

func TestNewURLConstruction(t *testing.T) {
	primary := &config.SiteConfig{ExternalURL: "https://gitlab.primary.example.com/"}
	secondary := &config.SiteConfig{ExternalURL: "https://gitlab.secondary.example.com"}
	r := New(primary, secondary, true)
	if r.primaryURL != "https://gitlab.primary.example.com/v2" {
		t.Errorf("primaryURL = %q", r.primaryURL)
	}
	if r.secondaryURL != "https://gitlab.secondary.example.com/v2" {
		t.Errorf("secondaryURL = %q", r.secondaryURL)
	}
	if !r.dryRun {
		t.Error("dryRun should be true")
	}
}

func TestName(t *testing.T) {
	r := &Reconciler{}
	if r.Name() != "registry" {
		t.Errorf("Name() = %q, want registry", r.Name())
	}
}
