// Package readonly enforces read-only mode on a secondary GitLab instance
// via SSH. It does this by enabling GitLab's maintenance mode (which
// blocks writes at the application layer) and pausing Sidekiq so no
// background jobs run on the secondary while it is a replica.
//
// This is standard GitLab omnibus administration, not EE-specific.
package readonly

import (
	"context"
	"fmt"
	"os/exec"
)

// Enable puts the secondary into read-only mode:
//  1. Enable maintenance mode (blocks web writes with a 503).
//  2. Sidekiq paused (no job processing on the replica).
func Enable(ctx context.Context, sshHost string, dryRun bool) error {
	return runSSH(ctx, sshHost, dryRun,
		"sudo gitlab-ctl deploy-registry-readonly start",
	)
}

// Disable restores normal read-write mode on the secondary (used after
// promotion to primary).
func Disable(ctx context.Context, sshHost string, dryRun bool) error {
	return runSSH(ctx, sshHost, dryRun,
		"sudo gitlab-ctl deploy-registry-readonly stop",
	)
}

func runSSH(ctx context.Context, sshHost string, dryRun bool, cmd string) error {
	if sshHost == "" {
		return fmt.Errorf("ssh_host not configured")
	}
	full := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		sshHost,
		cmd,
	}
	c := exec.CommandContext(ctx, "ssh", full...)
	if dryRun {
		fmt.Printf("[dry-run] ssh %s\n", cmd)
		return nil
	}
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh %s: %w: %s", sshHost, err, string(out))
	}
	return nil
}