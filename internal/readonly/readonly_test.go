package readonly

import (
	"context"
	"testing"
)

func TestEnableDryRun(t *testing.T) {
	err := Enable(context.Background(), "example.com:22", true)
	if err != nil {
		t.Fatalf("dry-run Enable should not error: %v", err)
	}
}

func TestDisableDryRun(t *testing.T) {
	err := Disable(context.Background(), "example.com:22", true)
	if err != nil {
		t.Fatalf("dry-run Disable should not error: %v", err)
	}
}

func TestEnableEmptySSHHost(t *testing.T) {
	err := Enable(context.Background(), "", true)
	if err == nil {
		t.Fatal("expected error for empty ssh_host")
	}
}

func TestDisableEmptySSHHost(t *testing.T) {
	err := Disable(context.Background(), "", true)
	if err == nil {
		t.Fatal("expected error for empty ssh_host")
	}
}
