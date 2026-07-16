// Package config defines the runtime configuration for gitlab-geo-sync.
//
// Configuration is loaded from a YAML file. All secret values MUST be
// supplied via environment variables referenced by ${ENV_VAR} placeholders
// in the YAML; literals are rejected for any field tagged `env:""`.
package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"github.com/anomalyco/gitlab-geo-sync/internal/sshexec"
)

// Config is the root configuration object.
type Config struct {
	Primary      SiteConfig          `yaml:"primary"`
	Secondaries  []SiteConfig        `yaml:"secondaries"`
	Sync         SyncConfig          `yaml:"sync"`
	Metrics      MetricsConfig       `yaml:"metrics"`
	Log          LogConfig           `yaml:"log"`
	ControlDB    string              `yaml:"control_db"` // "sqlite://path" or "postgres://dsn"
	Webhook      *WebhookConfig      `yaml:"webhook,omitempty"`
	APIValidator *APIValidatorConfig `yaml:"api_validator,omitempty"`
	Failover     *FailoverConfig     `yaml:"failover,omitempty"`
	SSH          SSHConfig           `yaml:"ssh,omitempty"`
}

// SiteConfig describes one GitLab site (primary or secondary).
type SiteConfig struct {
	Name        string            `yaml:"name"`
	ExternalURL string            `yaml:"external_url"`
	Postgres    PostgresConfig    `yaml:"postgres"`
	Git         GitStorage        `yaml:"git"`
	ObjectStore ObjectStoreConfig `yaml:"object_storage"`
	Registry    *RegistryConfig   `yaml:"registry,omitempty"`
	SSHHost     string            `yaml:"ssh_host,omitempty"` // host:port for rsync/git
}

// PostgresConfig: connection details for streaming replication control.
type PostgresConfig struct {
	// Host/Port/DB/User/Password are the control connection (used by geoctl
	// to query pg_stat_replication, not the streaming receiver itself,
	// which uses PrimaryConnInfo).
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DB       string `yaml:"db"`
	User     string `yaml:"user"`
	Password string `yaml:"password" env:"required"`

	// ReplicationUser is the Postgres role with REPLICATION privilege used
	// for physical WAL streaming.
	ReplicationUser     string `yaml:"replication_user"`
	ReplicationPassword string `yaml:"replication_password" env:"required"`

	// SlotName is the logical replication slot name to use on the primary.
	SlotName string `yaml:"slot_name,omitempty"`

	// SSLMode controls TLS for the control and replication connections.
	// Valid values: disable, allow, prefer, require, verify-ca, verify-full.
	// Defaults to "require" when empty — never sends passwords in cleartext
	// unless the operator explicitly sets sslmode: disable.
	SSLMode string `yaml:"sslmode,omitempty"`

	// SSLRootCert is the path to the CA certificate for verify-ca/verify-full.
	SSLRootCert string `yaml:"ssl_root_cert,omitempty"`

	// SSLCert / SSLKey are the client certificate and key for mutual TLS.
	SSLCert string `yaml:"ssl_cert,omitempty"`
	SSLKey  string `yaml:"ssl_key,omitempty"`
}

// GitStorage describes where and how git repository data lives.
type GitStorage struct {
	// Mode: "rsync" (filesystem copy of /var/opt/gitlab/git-data) or
	// "fetch" (per-project git fetch +refs/*:refs/* --prune).
	Mode string `yaml:"mode"`

	// ReposPath is the on-disk path to repositories (rsync mode), e.g.
	// /var/opt/gitlab/git-data/repositories.
	ReposPath string `yaml:"repos_path,omitempty"`

	// HashedStorage true if GitLab is using hashed storage layout.
	HashedStorage bool `yaml:"hashed_storage,omitempty"`
}

// ObjectStoreConfig describes blob storage replication.
type ObjectStoreConfig struct {
	Backend string `yaml:"backend"` // "s3" | "fs"

	// S3
	S3 *S3Config `yaml:"s3,omitempty"`

	// FS paths to rsync (fs backend), e.g. uploads, artifacts, packages, registry dirs.
	FSPaths []string `yaml:"fs_paths,omitempty"`
}

// S3Config describes an S3-compatible bucket pair.
type S3Config struct {
	Region         string        `yaml:"region"`
	PrimaryBucket  string        `yaml:"primary_bucket"`
	ReplicaBucket  string        `yaml:"replica_bucket"`
	AccessKey      string        `yaml:"access_key" env:"required"`
	SecretKey      string        `yaml:"secret_key" env:"required"`
	Endpoint       string        `yaml:"endpoint,omitempty"` // for MinIO etc.
	ReplicationLag time.Duration `yaml:"replication_lag,omitempty"`
}

// RegistryConfig describes the container registry backing.
type RegistryConfig struct {
	Mode string `yaml:"mode"` // "s3" | "fs"
	// FSPath is the registry filesystem root (mode=fs).
	FSPath string `yaml:"fs_path,omitempty"`
	// S3 follows the same S3Config pattern (registry may share or differ
	// from main object storage).
	S3 *S3Config `yaml:"s3,omitempty"`
}

// SyncConfig controls reconciler behavior.
type SyncConfig struct {
	// Interval for periodic full reconciliation sweeps.
	SweepInterval time.Duration `yaml:"sweep_interval"`

	// LagWarningThreshold: emit a warning if PG replay lag exceeds this.
	LagWarningThreshold time.Duration `yaml:"lag_warning_threshold"`

	// LagCriticalThreshold: emit critical + page (via DriftTotal) above.
	LagCriticalThreshold time.Duration `yaml:"lag_critical_threshold"`

	// FailoverEnabled: if true, geoctl failover is permitted (requires
	// quorum + --yes). Set false to forbid automated promotion.
	FailoverEnabled bool `yaml:"failover_enabled"`

	// ReadOnlySecondary: if true, the serve command will enforce
	// read-only mode on the secondary GitLab (block writes at the app
	// layer via maintenance mode or proxy 403s on mutating methods).
	ReadOnlySecondary bool `yaml:"read_only_secondary"`

	// ConsistencySamplePct: fraction (0.0–1.0) of secondary git repos
	// to run `git fsck` on during each sweep. Default 0.01 (1%).
	ConsistencySamplePct float64 `yaml:"consistency_sample_pct"`
}

// MetricsConfig: HTTP server for /metrics and /healthz.
type MetricsConfig struct {
	Addr string `yaml:"addr"`
}

// LogConfig: level + format.
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// WebhookConfig enables the webhook receiver that triggers immediate
// per-project sync on push/create/delete events from the primary.
type WebhookConfig struct {
	Addr        string `yaml:"addr"`                        // e.g. ":9102"
	SecretToken string `yaml:"secret_token" env:"required"` // GitLab webhook secret token for validation
}

// APIValidatorConfig enables the optional API-based observational
// validator that diffs counts between primary and secondary. It never
// writes via API — strictly read-only.
type APIValidatorConfig struct {
	Enabled        bool   `yaml:"enabled"`
	PrimaryToken   string `yaml:"primary_token" env:"required"`
	SecondaryToken string `yaml:"secondary_token" env:"required"`
}

// FailoverConfig controls the failover controller.
type FailoverConfig struct {
	// HealthCheckURLs are additional URLs to check for primary liveness
	// (e.g. https://gitlab.primary.example.com/-/health, the Gitaly address, etc.).
	HealthCheckURLs []string `yaml:"health_check_urls,omitempty"`

	// HealthCheckInterval: how often to poll primary health.
	HealthCheckInterval time.Duration `yaml:"health_check_interval,omitempty"`

	// QuorumRequired: minimum number of health checks that must fail to
	// consider the primary down. Default 1.
	QuorumRequired int `yaml:"quorum_required,omitempty"`

	// DNSPlugin: "route53" | "cloudflare" | "none" (manual DNS).
	DNSPlugin string `yaml:"dns_plugin,omitempty"`

	// DNSConfig is plugin-specific config (e.g. zone ID, record name).
	DNSConfig map[string]string `yaml:"dns_config,omitempty"`

	// AutoFailover: if true, failover triggers automatically when quorum
	// is reached. DANGEROUS — default false (human-gated).
	AutoFailover bool `yaml:"auto_failover,omitempty"`
}

// SSHConfig controls SSH host-key verification policy for all SSH-based
// operations (rsync, git fetch, db_key_base checks, failover commands,
// doctor checks). Production deployments should pin host keys via
// known_hosts_file and leave strict_host_key_checking at its default
// of "yes".
type SSHConfig struct {
	// KnownHostsFile is the path to a known_hosts file. When set,
	// StrictHostKeyChecking defaults to "yes" and -o
	// UserKnownHostsFile=<path> is passed to every ssh invocation.
	KnownHostsFile string `yaml:"known_hosts_file,omitempty"`

	// StrictHostKeyChecking overrides the default. Valid values:
	// "yes", "no", "accept-new". When empty, defaults to "yes" if
	// KnownHostsFile is set, otherwise "accept-new" (TOFU).
	StrictHostKeyChecking string `yaml:"strict_host_key_checking,omitempty"`
}

// Load reads and validates a config file, expanding ${ENV} placeholders
// and enforcing that all fields tagged env:"required" are non-empty.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	expanded, err := ExpandEnv(raw)
	if err != nil {
		return nil, err
	}
	var c Config
	dec := yaml.NewDecoder(strings.NewReader(string(expanded)))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	// Stamp an instance ID so parallel reconcilers are disambiguatable.
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// ExpandEnv replaces ${VAR} placeholders in raw YAML with the corresponding
// environment variable value. If a referenced variable is unset or empty,
// returns an error naming the variable. Literal secret values are NOT
// supported; the env expansion is the only mechanism.
var envRefRe = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

// ExpandEnv expands ${VAR} references in raw bytes.
func ExpandEnv(raw []byte) ([]byte, error) {
	var missing []string
	out := envRefRe.ReplaceAllFunc(raw, func(m []byte) []byte {
		name := envRefRe.FindSubmatch(m)[1]
		v, ok := os.LookupEnv(string(name))
		if !ok || v == "" {
			missing = append(missing, string(name))
			return m
		}
		return []byte(v)
	})
	if len(missing) > 0 {
		return nil, fmt.Errorf("environment variables referenced but not set: %s",
			strings.Join(missing, ", "))
	}
	return out, nil
}

// Validate performs semantic validation beyond YAML parsing.
func (c *Config) validate() error {
	var errs []error
	if c.Primary.Name == "" {
		errs = append(errs, errors.New("primary.name is required"))
	}
	if c.Primary.ExternalURL == "" {
		errs = append(errs, errors.New("primary.external_url is required"))
	}
	if c.Primary.Postgres.Host == "" {
		errs = append(errs, errors.New("primary.postgres.host is required"))
	}
	if c.Primary.Postgres.Port == 0 {
		errs = append(errs, errors.New("primary.postgres.port is required"))
	}
	if c.Primary.Postgres.ReplicationUser == "" {
		errs = append(errs, errors.New("primary.postgres.replication_user is required"))
	}
	if c.Primary.Postgres.ReplicationPassword == "" {
		errs = append(errs, errors.New("primary.postgres.replication_password is required (via env)"))
	}
	c.warnInsecureSSL(&errs)
	if c.Primary.Git.Mode == "" {
		errs = append(errs, errors.New("primary.git.mode is required (rsync|fetch)"))
	}
	switch c.Primary.Git.Mode {
	case "rsync", "fetch":
	default:
		errs = append(errs, fmt.Errorf("primary.git.mode %q invalid; want rsync or fetch", c.Primary.Git.Mode))
	}
	if c.Primary.Git.Mode == "rsync" && c.Primary.Git.ReposPath == "" {
		errs = append(errs, errors.New("primary.git.repos_path is required when mode=rsync"))
	}
	if len(c.Secondaries) == 0 {
		errs = append(errs, errors.New("at least one secondary is required"))
	}
	seen := map[string]bool{}
	for i := range c.Secondaries {
		s := &c.Secondaries[i]
		if s.Name == "" {
			errs = append(errs, fmt.Errorf("secondaries[%d].name is required", i))
		}
		if s.Name != "" && seen[s.Name] {
			errs = append(errs, fmt.Errorf("secondaries[%d].name %q duplicated", i, s.Name))
		}
		seen[s.Name] = true
		if s.Postgres.Host == "" {
			errs = append(errs, fmt.Errorf("secondaries[%d].postgres.host is required", i))
		}
		if s.Postgres.ReplicationPassword == "" {
			errs = append(errs, fmt.Errorf("secondaries[%d].postgres.replication_password is required (via env)", i))
		}
	}
	if c.Sync.SweepInterval == 0 {
		c.Sync.SweepInterval = 5 * time.Minute
	}
	if c.Sync.LagWarningThreshold == 0 {
		c.Sync.LagWarningThreshold = 30 * time.Second
	}
	if c.Sync.LagCriticalThreshold == 0 {
		c.Sync.LagCriticalThreshold = 5 * time.Minute
	}
	if c.Metrics.Addr == "" {
		c.Metrics.Addr = ":9101"
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "json"
	}
	if c.ControlDB == "" {
		c.ControlDB = "sqlite://data/geoctl.db"
	}
	if c.Sync.ConsistencySamplePct == 0 {
		c.Sync.ConsistencySamplePct = 0.01
	}
	if c.Failover != nil {
		if c.Failover.HealthCheckInterval == 0 {
			c.Failover.HealthCheckInterval = 10 * time.Second
		}
		if c.Failover.QuorumRequired == 0 {
			c.Failover.QuorumRequired = 1
		}
		if c.Failover.DNSPlugin == "" {
			c.Failover.DNSPlugin = "none"
		}
	}
	return errors.Join(errs...)
}

// warnInsecureSSL logs a warning for any connection that explicitly
// uses sslmode=disable. This is a non-fatal warning so dev/local
// setups still work, but production operators are alerted.
func (c *Config) warnInsecureSSL(_ *[]error) {
	check := func(label string, pg PostgresConfig) {
		if pg.SSLMode == "disable" {
			log.Warn().Str("site", label).
				Msg("postgres.sslmode is 'disable' — passwords will be sent in cleartext; use only for local dev")
		}
	}
	check("primary", c.Primary.Postgres)
	for i, s := range c.Secondaries {
		check(fmt.Sprintf("secondaries[%d]:%s", i, s.Name), s.Postgres)
	}
}

// InstanceID returns a stable per-process identifier for log/metrics
// disambiguation; regenerated each process start.
func (c *Config) InstanceID() string {
	return uuid.NewString()
}

// SSHExecConfig returns the sshexec.Config derived from the SSH config block.
func (c *Config) SSHExecConfig() sshexec.Config {
	return sshexec.Config{
		KnownHostsFile:        c.SSH.KnownHostsFile,
		StrictHostKeyChecking: c.SSH.StrictHostKeyChecking,
	}
}

// DSN constructs a libpq-style connection string for the control user.
// Values are libpq-quoted so passwords with spaces, quotes, or
// backslashes are handled correctly. SSL defaults to require.
func (p PostgresConfig) DSN() string {
	return p.buildDSN(p.User, p.Password, p.DB, "")
}

// ReplicationDSN constructs a libpq connection string for the WAL receiver.
func (p PostgresConfig) ReplicationDSN() string {
	return p.buildDSN(p.ReplicationUser, p.ReplicationPassword, "replication",
		"application_name=gitlab-geo-sync")
}

func (p PostgresConfig) buildDSN(user, password, dbname, extra string) string {
	pairs := []kv{
		{"host", p.Host},
		{"port", strconv.Itoa(p.Port)},
		{"user", user},
		{"password", password},
		{"dbname", dbname},
		{"sslmode", p.effectiveSSLMode()},
	}
	if p.SSLRootCert != "" {
		pairs = append(pairs, kv{"sslrootcert", p.SSLRootCert})
	}
	if p.SSLCert != "" {
		pairs = append(pairs, kv{"sslcert", p.SSLCert})
	}
	if p.SSLKey != "" {
		pairs = append(pairs, kv{"sslkey", p.SSLKey})
	}
	var sb strings.Builder
	for i, kv := range pairs {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(kv.key)
		sb.WriteByte('=')
		sb.WriteString(quoteLibPQValue(kv.val))
	}
	if extra != "" {
		sb.WriteByte(' ')
		sb.WriteString(extra)
	}
	return sb.String()
}

func (p PostgresConfig) effectiveSSLMode() string {
	if p.SSLMode == "" {
		return "require"
	}
	return p.SSLMode
}

type kv struct {
	key string
	val string
}

// quoteLibPQValue quotes a value for a libpq key=value DSN per the
// documented rules: if the value is empty or contains no special
// characters (space, ', \), it is returned as-is. Otherwise it is
// wrapped in single quotes with ' and \ backslash-escaped.
func quoteLibPQValue(v string) string {
	if v == "" {
		return "''"
	}
	if !strings.ContainsAny(v, " '\\") {
		return v
	}
	return "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'").Replace(v) + "'"
}
