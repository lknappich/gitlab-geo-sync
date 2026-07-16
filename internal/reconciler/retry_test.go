package reconciler

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrySucceedsFirstAttempt(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), "test", 3, 1*time.Millisecond, 10*time.Millisecond, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetrySucceedsAfterFailures(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), "test", 3, 1*time.Millisecond, 10*time.Millisecond, func() error {
		calls++
		if calls < 2 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestRetryExhaustedReturnsLastError(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), "test", 2, 1*time.Millisecond, 5*time.Millisecond, func() error {
		calls++
		return errors.New("persistent")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestRetryRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Retry(ctx, "test", 3, 1*time.Millisecond, 5*time.Millisecond, func() error {
		return errors.New("transient")
	})
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
