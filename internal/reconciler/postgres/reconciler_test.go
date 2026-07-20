package postgresreconciler

import (
	"testing"
)

func TestName(t *testing.T) {
	r := &Reconciler{}
	if r.Name() != "postgres" {
		t.Errorf("Name() = %q, want postgres", r.Name())
	}
}

func TestSecondaryPoolNotFound(t *testing.T) {
	r := &Reconciler{
		secondaries: []secondaryConn{
			{name: "s1", pool: nil},
			{name: "s2", pool: nil},
		},
	}
	if got := r.SecondaryPool("nonexistent"); got != nil {
		t.Errorf("SecondaryPool(unknown) = %v, want nil", got)
	}
}

func TestSecondaryPoolFound(t *testing.T) {
	r := &Reconciler{
		secondaries: []secondaryConn{
			{name: "s1", pool: nil},
		},
	}
	if got := r.SecondaryPool("s1"); got != nil {
		t.Errorf("SecondaryPool(s1) = %v, want nil (pool is nil in test)", got)
	}
}

func TestCloseNilSafe(t *testing.T) {
	r := &Reconciler{}
	r.Close()
}

func TestPrimaryPoolNil(t *testing.T) {
	r := &Reconciler{}
	if r.PrimaryPool() != nil {
		t.Error("PrimaryPool() should be nil on zero-value Reconciler")
	}
}
