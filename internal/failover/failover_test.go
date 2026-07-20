package failover

import (
	"context"
	"testing"

	"github.com/lknappich/gitlab-geo-sync/internal/config"
)

func TestNewControllerDefaults(t *testing.T) {
	cfg := &config.Config{
		Primary: config.SiteConfig{
			Name:        "p",
			ExternalURL: "https://p.example.com",
		},
		Secondaries: []config.SiteConfig{
			{Name: "s", ExternalURL: "https://s.example.com"},
		},
	}
	fc := New(cfg, true)
	if fc.quorum != 1 {
		t.Errorf("quorum = %d, want 1", fc.quorum)
	}
	if fc.autoFailover {
		t.Error("autoFailover should be false by default")
	}
}

func TestNewControllerWithFailoverConfig(t *testing.T) {
	cfg := &config.Config{
		Primary: config.SiteConfig{
			Name:        "p",
			ExternalURL: "https://p.example.com",
		},
		Secondaries: []config.SiteConfig{
			{Name: "s", ExternalURL: "https://s.example.com"},
		},
		Failover: &config.FailoverConfig{
			AutoFailover:   true,
			QuorumRequired: 3,
			DNSPlugin:      "route53",
		},
	}
	fc := New(cfg, false)
	if fc.quorum != 3 {
		t.Errorf("quorum = %d, want 3", fc.quorum)
	}
	if !fc.autoFailover {
		t.Error("autoFailover should be true")
	}
}

func TestPromoteRejectsWhenFailoverDisabled(t *testing.T) {
	cfg := &config.Config{
		Primary: config.SiteConfig{
			Name:        "p",
			ExternalURL: "https://p.example.com",
		},
		Secondaries: []config.SiteConfig{
			{Name: "s", ExternalURL: "https://s.example.com"},
		},
		Sync: config.SyncConfig{FailoverEnabled: false},
	}
	fc := New(cfg, false)
	err := fc.Promote(context.Background(), "s")
	if err == nil {
		t.Fatal("expected error when failover disabled")
	}
}

func TestPromoteDryRunSkipsChecks(t *testing.T) {
	cfg := &config.Config{
		Primary: config.SiteConfig{
			Name:        "p",
			ExternalURL: "https://p.example.com",
		},
		Secondaries: []config.SiteConfig{
			{Name: "s", ExternalURL: "https://s.example.com"},
		},
		Sync: config.SyncConfig{FailoverEnabled: false},
	}
	fc := New(cfg, true) // dryRun=true
	err := fc.Promote(context.Background(), "s")
	if err != nil {
		t.Fatalf("dry-run promote should succeed, got: %v", err)
	}
}

func TestPromoteRejectsUnknownSecondary(t *testing.T) {
	cfg := &config.Config{
		Primary: config.SiteConfig{
			Name:        "p",
			ExternalURL: "https://p.example.com",
		},
		Secondaries: []config.SiteConfig{
			{Name: "s", ExternalURL: "https://s.example.com"},
		},
	}
	fc := New(cfg, true)
	err := fc.Promote(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown secondary")
	}
}

func TestIsPrimaryDownInitiallyFalse(t *testing.T) {
	cfg := &config.Config{
		Primary: config.SiteConfig{
			Name:        "p",
			ExternalURL: "https://p.example.com",
		},
		Secondaries: []config.SiteConfig{
			{Name: "s", ExternalURL: "https://s.example.com"},
		},
	}
	fc := New(cfg, false)
	if fc.IsPrimaryDown() {
		t.Error("primary should be up initially")
	}
}
