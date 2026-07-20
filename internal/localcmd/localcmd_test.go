package localcmd

import (
	"context"
	"errors"
	"testing"
)

type mockRunner struct {
	out []byte
	err error
}

func (m *mockRunner) Run(ctx context.Context, name string, args, env []string) ([]byte, error) {
	return m.out, m.err
}

func TestRunWithDefault(t *testing.T) {
	// RunWith with nil runner falls back to Default (real exec).
	// Use a simple command that always succeeds.
	out, err := RunWith(context.Background(), nil, "true", nil, nil)
	if err != nil {
		t.Fatalf("true should succeed: %v (out: %s)", err, out)
	}
}

func TestRunWithMock(t *testing.T) {
	mock := &mockRunner{out: []byte("hello"), err: nil}
	out, err := RunWith(context.Background(), mock, "echo", []string{"hello"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "hello" {
		t.Errorf("out = %q, want hello", out)
	}
}

func TestRunWithMockError(t *testing.T) {
	mock := &mockRunner{err: errors.New("failed")}
	_, err := RunWith(context.Background(), mock, "false", nil, nil)
	if err == nil {
		t.Fatal("expected error from mock")
	}
}

func TestDefaultRunnerIsRealRunner(t *testing.T) {
	if _, ok := Default.(realRunner); !ok {
		t.Error("Default should be a realRunner")
	}
}
