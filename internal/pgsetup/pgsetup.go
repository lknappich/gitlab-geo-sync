// Package pgsetup implements `geoctl pg setup` — bootstraps a secondary
// PostgreSQL as a physical streaming replica of the primary using
// pg_basebackup, then writes the standby configuration.
package pgsetup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Options controls a pg_basebackup-based standby bootstrap.
type Options struct {
	// PrimaryDSN is a libpq connection string for the replication user
	// on the primary, e.g. "host=... user=replicator password=...".
	PrimaryDSN string
	// DataDir is the secondary's PGDATA, e.g. /var/lib/postgresql/data.
	DataDir string
	// SlotName is an optional physical replication slot to create/use.
	SlotName string
	// DryRun prints commands without executing.
	DryRun bool
}

// Run performs the setup: validates the data dir is empty or absent,
// runs pg_basebackup, writes standby.signal and primary_conninfo into
// postgresql.auto.conf.
func Run(ctx context.Context, opts Options) error {
	if opts.DataDir == "" {
		return fmt.Errorf("data_dir is required")
	}
	if opts.PrimaryDSN == "" {
		return fmt.Errorf("primary_dsn is required")
	}

	if err := checkDataDir(opts.DataDir); err != nil {
		return err
	}

	args := []string{
		"-D", opts.DataDir,
		"-d", opts.PrimaryDSN,
		"-X", "stream",
		"-S", "main", // main WAL segment stream; overridden by SlotName below
		"-c", "fast",
		"-R", // write standby.signal + primary_conninfo
		"-P",
	}
	if opts.SlotName != "" {
		args = append(args, "-S", opts.SlotName, "--create-slot")
	}

	cmd := exec.CommandContext(ctx, "pg_basebackup", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if opts.DryRun {
		fmt.Printf("[dry-run] %s %s\n", cmd.Path, strings.Join(args, " "))
		return nil
	}
	fmt.Printf("running pg_basebackup into %s ...\n", opts.DataDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_basebackup: %w", err)
	}

	// pg_basebackup -R writes primary_conninfo to postgresql.auto.conf,
	// but the application_name may be the default. Override it so the
	// primary's pg_stat_replication row matches the secondary's name.
	if err := appendConnInfoAppname(opts.DataDir, opts.SlotName); err != nil {
		return err
	}

	fmt.Printf("standby bootstrap complete. data_dir=%s\n", opts.DataDir)
	return nil
}

// checkDataDir ensures the target is empty/absent so we don't clobber
// an existing data directory.
func checkDataDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err == nil && len(entries) > 0 {
		return fmt.Errorf("data_dir %s is not empty (refusing to overwrite)", dir)
	}
	return nil
}

// appendConnInfoAppname appends/updates the primary_conninfo line in
// postgresql.auto.conf so application_name is set for pg_stat_replication.
func appendConnInfoAppname(dataDir, appName string) error {
	if appName == "" {
		return nil
	}
	path := filepath.Join(dataDir, "postgresql.auto.conf")
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read postgresql.auto.conf: %w", err)
	}
	lines := strings.Split(string(content), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "primary_conninfo = ") {
			if !strings.Contains(line, "application_name=") {
				lines[i] = strings.TrimRight(line, "'") +
					" application_name=" + appName + "'"
			}
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, fmt.Sprintf(
			"primary_conninfo = '%s application_name=%s'",
			strings.Trim(string(content), "\n'"), appName))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600)
}