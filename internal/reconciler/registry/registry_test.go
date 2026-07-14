package registry

import (
	"testing"
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