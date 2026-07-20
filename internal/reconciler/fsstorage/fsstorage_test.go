package fsstorage

import (
	"testing"

	"github.com/lknappich/gitlab-geo-sync/internal/config"
	"github.com/lknappich/gitlab-geo-sync/internal/sshexec"
)

func TestNewCollectsFSPaths(t *testing.T) {
	primary := &config.SiteConfig{
		SSHHost: "primary:22",
		ObjectStore: config.ObjectStoreConfig{
			Backend: "fs",
			FSPaths: []string{"/var/opt/gitlab/uploads", "/var/opt/gitlab/artifacts"},
		},
		Registry: &config.RegistryConfig{
			Mode:   "fs",
			FSPath: "/var/opt/gitlab/registry",
		},
	}
	secondary := &config.SiteConfig{}

	r := New(primary, secondary, true, sshexec.Default)
	if len(r.pathPairs) != 3 {
		t.Errorf("expected 3 path pairs, got %d", len(r.pathPairs))
	}
}

func TestNewNoRegistry(t *testing.T) {
	primary := &config.SiteConfig{
		SSHHost: "primary:22",
		ObjectStore: config.ObjectStoreConfig{
			Backend: "fs",
			FSPaths: []string{"/uploads"},
		},
	}
	secondary := &config.SiteConfig{}

	r := New(primary, secondary, true, sshexec.Default)
	if len(r.pathPairs) != 1 {
		t.Errorf("expected 1 path pair, got %d", len(r.pathPairs))
	}
}

func TestNewNoPaths(t *testing.T) {
	primary := &config.SiteConfig{
		SSHHost: "primary:22",
		ObjectStore: config.ObjectStoreConfig{
			Backend: "fs",
		},
	}
	secondary := &config.SiteConfig{}

	r := New(primary, secondary, true, sshexec.Default)
	if len(r.pathPairs) != 0 {
		t.Errorf("expected 0 path pairs, got %d", len(r.pathPairs))
	}
}
