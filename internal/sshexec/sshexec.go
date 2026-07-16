// Package sshexec centralizes SSH command construction so that host-key
// checking policy is enforced uniformly across all call sites. Every
// SSH invocation in the codebase should go through this package rather
// than building exec.CommandContext("ssh", ...) inline.
package sshexec

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Config controls SSH options applied to every connection.
type Config struct {
	// KnownHostsFile is the path to a known_hosts file. When set,
	// -o UserKnownHostsFile=<path> is passed and
	// StrictHostKeyChecking defaults to "yes". When empty,
	// StrictHostKeyChecking defaults to "accept-new" (TOFU).
	KnownHostsFile string

	// StrictHostKeyChecking overrides the default. Valid values:
	// "yes", "no", "accept-new". When empty, defaults to "yes" if
	// KnownHostsFile is set, otherwise "accept-new".
	StrictHostKeyChecking string
}

// EffectiveStrictHostKeyChecking returns the resolved checking mode.
func (c Config) EffectiveStrictHostKeyChecking() string {
	if c.StrictHostKeyChecking != "" {
		return c.StrictHostKeyChecking
	}
	if c.KnownHostsFile != "" {
		return "yes"
	}
	return "accept-new"
}

// ExtraArgs returns the -o options to inject into an ssh command.
func (c Config) ExtraArgs() []string {
	args := []string{
		"-o", "StrictHostKeyChecking=" + c.EffectiveStrictHostKeyChecking(),
	}
	if c.KnownHostsFile != "" {
		args = append(args, "-o", "UserKnownHostsFile="+c.KnownHostsFile)
	}
	return args
}

// Command builds an exec.Cmd for an SSH session to host running remoteCmd.
// The caller is responsible for setting stdout/stderr and calling Run()
// or CombinedOutput().
func (c Config) Command(ctx context.Context, host, remoteCmd string) *exec.Cmd {
	args := append(c.ExtraArgs(), host, remoteCmd)
	return exec.CommandContext(ctx, "ssh", args...)
}

// CombinedOutput runs the SSH command and returns combined output.
func (c Config) CombinedOutput(ctx context.Context, host, remoteCmd string) ([]byte, error) {
	cmd := c.Command(ctx, host, remoteCmd)
	return cmd.CombinedOutput()
}

// SSHString builds the ssh options string for use in rsync's -e flag,
// e.g. "ssh -o StrictHostKeyChecking=yes -o UserKnownHostsFile=/path".
func (c Config) SSHString() string {
	return "ssh " + strings.Join(c.ExtraArgs(), " ")
}

// Default is the fallback Config used when no site-level SSH config is
// provided. It uses accept-new (TOFU) so out-of-the-box behavior matches
// the previous code, but operators can pin host keys via config.
var Default = Config{}

// CheckHost returns an error if host is empty.
func CheckHost(host string) error {
	if host == "" {
		return fmt.Errorf("ssh_host not configured")
	}
	return nil
}
