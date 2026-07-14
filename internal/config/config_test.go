package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempConfig writes a YAML file in a temp dir and returns its path.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return p
}

func TestLoadMinimal(t *testing.T) {
	// All required env-secret placeholders must be set before Load.
	t.Setenv("PG_REPL_PASSWORD", "replpass")
	t.Setenv("PG_CTRL_PASSWORD", "ctrlpass")
	t.Setenv("S3_AK", "AKIAEXAMPLE")
	t.Setenv("S3_SK", "secretexample")
	t.Setenv("SEC_REPL_PASSWORD", "secpass")

	yaml := `
primary:
  name: primary-eu
  external_url: https://gitlab.primary.example.com
  postgres:
    host: 10.0.0.10
    port: 5432
    db: gitlabhq_production
    user: gitlab
    password: ${PG_CTRL_PASSWORD}
    replication_user: gitlab_repl
    replication_password: ${PG_REPL_PASSWORD}
  git:
    mode: rsync
    repos_path: /var/opt/gitlab/git-data/repositories
  object_storage:
    backend: s3
    s3:
      region: eu-west-1
      primary_bucket: gitlab-primary
      replica_bucket: gitlab-replica
      access_key: ${S3_AK}
      secret_key: ${S3_SK}
secondaries:
  - name: secondary-us
    external_url: https://gitlab.secondary.example.com
    postgres:
      host: 10.1.0.10
      port: 5432
      db: gitlabhq_production
      user: gitlab
      password: ${PG_CTRL_PASSWORD}
      replication_user: gitlab_repl
      replication_password: ${SEC_REPL_PASSWORD}
    git:
      mode: rsync
      repos_path: /var/opt/gitlab/git-data/repositories
    object_storage:
      backend: s3
      s3:
        region: us-east-1
        primary_bucket: gitlab-primary
        replica_bucket: gitlab-replica
        access_key: ${S3_AK}
        secret_key: ${S3_SK}
sync:
  sweep_interval: 1m
  failover_enabled: true
metrics:
  addr: ":9101"
log:
  level: debug
  format: text
control_db: sqlite://data/geoctl.db
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Primary.Name != "primary-eu" {
		t.Errorf("primary name = %q", cfg.Primary.Name)
	}
	if cfg.Primary.Postgres.ReplicationPassword != "replpass" {
		t.Errorf("repl password not expanded: %q", cfg.Primary.Postgres.ReplicationPassword)
	}
	if cfg.Sync.SweepInterval.String() != "1m0s" {
		t.Errorf("sweep interval = %v", cfg.Sync.SweepInterval)
	}
	if len(cfg.Secondaries) != 1 || cfg.Secondaries[0].Name != "secondary-us" {
		t.Errorf("secondaries = %+v", cfg.Secondaries)
	}
}

func TestLoadRejectsMissingEnv(t *testing.T) {
	// Intentionally do NOT set PG_REPL_PASSWORD.
	t.Setenv("PG_CTRL_PASSWORD", "x")

	yaml := `
primary:
  name: p
  external_url: https://p.example.com
  postgres:
    host: h
    port: 5432
    db: d
    user: u
    password: ${PG_CTRL_PASSWORD}
    replication_user: r
    replication_password: ${PG_REPL_PASSWORD}
  git:
    mode: rsync
    repos_path: /r
  object_storage:
    backend: fs
secondaries:
  - name: s
    postgres:
      host: h
      port: 5432
      db: d
      user: u
      replication_user: r
      replication_password: ${SEC_REPL_PASSWORD}
    git:
      mode: rsync
      repos_path: /r
    object_storage:
      backend: fs
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing env vars, got nil")
	}
}

func TestValidateRejectsUnknownGitMode(t *testing.T) {
	t.Setenv("PG_REPL_PASSWORD", "x")
	t.Setenv("PG_CTRL_PASSWORD", "x")
	t.Setenv("SEC_REPL_PASSWORD", "x")

	yaml := `
primary:
  name: p
  external_url: https://p.example.com
  postgres:
    host: h
    port: 5432
    db: d
    user: u
    password: ${PG_CTRL_PASSWORD}
    replication_user: r
    replication_password: ${PG_REPL_PASSWORD}
  git:
    mode: teleport
  object_storage:
    backend: fs
secondaries:
  - name: s
    postgres:
      host: h
      port: 5432
      db: d
      user: u
      replication_user: r
      replication_password: ${SEC_REPL_PASSWORD}
    git:
      mode: rsync
      repos_path: /r
    object_storage:
      backend: fs
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid git mode, got nil")
	}
}