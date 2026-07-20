package objectstorage

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lknappich/gitlab-geo-sync/internal/config"
)

func TestNewRejectsNilPrimary(t *testing.T) {
	_, err := New(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for nil primary")
	}
	if !strings.Contains(err.Error(), "primary.object_storage.s3 is required") {
		t.Errorf("err = %v", err)
	}
}

func TestName(t *testing.T) {
	r := &Reconciler{}
	if r.Name() != "object_storage" {
		t.Errorf("Name() = %q, want object_storage", r.Name())
	}
}

func TestCredentialsFromConfig(t *testing.T) {
	s3cfg := &config.S3Config{
		AccessKey: "AKIAEXAMPLE",
		SecretKey: "secretexample",
	}
	provider := credentialsFromConfig(s3cfg)
	if provider == nil {
		t.Fatal("credentialsFromConfig returned nil provider")
	}
	creds, err := provider.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if creds.AccessKeyID != "AKIAEXAMPLE" {
		t.Errorf("AccessKeyID = %q", creds.AccessKeyID)
	}
	if creds.SecretAccessKey != "secretexample" {
		t.Errorf("SecretAccessKey = %q", creds.SecretAccessKey)
	}
	if creds.SessionToken != "" {
		t.Errorf("SessionToken = %q, want empty", creds.SessionToken)
	}
}

func TestPtrDereferences(t *testing.T) {
	v := ptr("hello")
	if *v != "hello" {
		t.Errorf("*ptr = %q", *v)
	}
	n := ptr(42)
	if *n != 42 {
		t.Errorf("*ptr = %d", *n)
	}
}

// --- Mock bucketLister ---

type mockLister struct {
	count int64
	size  int64
	err   error
}

func (m *mockLister) stats(ctx context.Context) (int64, int64, error) {
	return m.count, m.size, m.err
}

func TestReconcileMatch(t *testing.T) {
	r := newReconcilerWithListers(
		&mockLister{count: 10, size: 1024},
		&mockLister{count: 10, size: 1024},
		0,
	)
	res := r.Reconcile(context.Background())
	if !res.OK {
		t.Errorf("expected OK, got: %s", res.Detail)
	}
}

func TestReconcileDrift(t *testing.T) {
	r := newReconcilerWithListers(
		&mockLister{count: 100, size: 1024},
		&mockLister{count: 99, size: 1024},
		0,
	)
	res := r.Reconcile(context.Background())
	if res.OK {
		t.Error("expected not-OK on drift")
	}
	if res.Remaining != 1 {
		t.Errorf("Remaining = %d, want 1", res.Remaining)
	}
}

func TestReconcileDriftNegativeDelta(t *testing.T) {
	r := newReconcilerWithListers(
		&mockLister{count: 50, size: 100},
		&mockLister{count: 55, size: 100},
		0,
	)
	res := r.Reconcile(context.Background())
	if res.OK {
		t.Error("expected not-OK on drift")
	}
	if res.Remaining != 5 {
		t.Errorf("Remaining = %d, want 5", res.Remaining)
	}
}

func TestReconcilePrimaryError(t *testing.T) {
	r := newReconcilerWithListers(
		&mockLister{err: errors.New("access denied")},
		&mockLister{count: 10, size: 100},
		0,
	)
	res := r.Reconcile(context.Background())
	if res.OK {
		t.Error("expected not-OK on primary error")
	}
	if !strings.Contains(res.Detail, "primary list") {
		t.Errorf("Detail = %q", res.Detail)
	}
}

func TestReconcileReplicaError(t *testing.T) {
	r := newReconcilerWithListers(
		&mockLister{count: 10, size: 100},
		&mockLister{err: errors.New("access denied")},
		0,
	)
	res := r.Reconcile(context.Background())
	if res.OK {
		t.Error("expected not-OK on replica error")
	}
	if !strings.Contains(res.Detail, "replica list") {
		t.Errorf("Detail = %q", res.Detail)
	}
}

func TestReconcileSizeDrift(t *testing.T) {
	r := newReconcilerWithListers(
		&mockLister{count: 10, size: 100},
		&mockLister{count: 10, size: 90},
		0,
	)
	res := r.Reconcile(context.Background())
	if res.OK {
		t.Error("expected not-OK on size drift")
	}
}
