package metrics

import (
	"context"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	s := NewServer(":0")
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.addr != ":0" {
		t.Errorf("got addr %q", s.addr)
	}
}

func TestServerStartCancel(t *testing.T) {
	s := NewServer(":0")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	err := s.Start(ctx)
	if err != nil {
		t.Errorf("unexpected error on cancel: %v", err)
	}
}

func TestRegisterDoesNotPanic(t *testing.T) {
	Register(Registry)
}
