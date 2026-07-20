package gitrsync

import (
	"testing"

	"github.com/lknappich/gitlab-geo-sync/internal/config"
	"github.com/lknappich/gitlab-geo-sync/internal/sshexec"
)

func TestNewConstructorFields(t *testing.T) {
	primary := &config.SiteConfig{
		SSHHost: "primary.example.com:22",
		Git:     config.GitStorage{ReposPath: "/var/opt/gitlab/git-data/repositories"},
	}
	secondary := &config.SiteConfig{
		Git: config.GitStorage{ReposPath: "/var/opt/gitlab/git-data/repositories"},
	}
	sshCfg := sshexec.Config{KnownHostsFile: "/etc/known_hosts", StrictHostKeyChecking: "yes"}
	r := New(primary, secondary, true, sshCfg)
	if r.sshHost != "primary.example.com:22" {
		t.Errorf("sshHost = %q", r.sshHost)
	}
	if r.srcPath != "/var/opt/gitlab/git-data/repositories" {
		t.Errorf("srcPath = %q", r.srcPath)
	}
	if r.dstPath != "/var/opt/gitlab/git-data/repositories" {
		t.Errorf("dstPath = %q", r.dstPath)
	}
	if !r.dryRun {
		t.Error("dryRun should be true")
	}
	if r.sshCfg.KnownHostsFile != "/etc/known_hosts" {
		t.Errorf("sshCfg.KnownHostsFile = %q", r.sshCfg.KnownHostsFile)
	}
}

func TestNewEmptyPaths(t *testing.T) {
	primary := &config.SiteConfig{SSHHost: "host:22"}
	secondary := &config.SiteConfig{}
	r := New(primary, secondary, false, sshexec.Config{})
	if r.sshHost != "host:22" {
		t.Errorf("sshHost = %q", r.sshHost)
	}
	if r.srcPath != "" {
		t.Errorf("srcPath = %q, want empty", r.srcPath)
	}
	if r.dstPath != "" {
		t.Errorf("dstPath = %q, want empty", r.dstPath)
	}
	if r.dryRun {
		t.Error("dryRun should be false")
	}
}

func TestName(t *testing.T) {
	r := &Reconciler{}
	if r.Name() != "git_rsync" {
		t.Errorf("Name() = %q, want git_rsync", r.Name())
	}
}

func TestErrResultOK(t *testing.T) {
	got := errResult(nil)
	if got != "ok" {
		t.Errorf("errResult(nil) = %q, want ok", got)
	}
}

func TestErrResultError(t *testing.T) {
	got := errResult(errOops)
	if got != "error" {
		t.Errorf("errResult(err) = %q, want error", got)
	}
}

var errOops = errString("oops")

type errString string

func (e errString) Error() string { return string(e) }
