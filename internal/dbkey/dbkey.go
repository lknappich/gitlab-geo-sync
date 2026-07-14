// Package dbkey checks that the secondary's db_key_base matches the
// primary's. The db_key_base is the GitLab Rails secret used to encrypt
// webhook secrets, access tokens, 2FA seeds, etc. For a true 1:1 replica
// the secondary MUST share this key so the GitLab application itself
// (not our tool) can decrypt columns on the secondary.
//
// This package reads the key from each site's gitlab.rb config file via
// SSH. It never decrypts anything — it only compares the key bytes.
package dbkey

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var dbKeyRe = regexp.MustCompile(`^\s*gitlab_rails\['db_key_base'\]\s*=\s*['"]?([A-Za-z0-9_-]+)`)

// Check fetches the db_key_base from primary and secondary via SSH and
// compares them. Returns nil if they match, an error describing the
// mismatch otherwise. The key value itself is never logged.
func Check(ctx context.Context, primarySSH, secondarySSH string) error {
	pKey, err := fetchKey(ctx, primarySSH)
	if err != nil {
		return fmt.Errorf("primary db_key_base: %w", err)
	}
	sKey, err := fetchKey(ctx, secondarySSH)
	if err != nil {
		return fmt.Errorf("secondary db_key_base: %w", err)
	}
	if pKey != sKey {
		return fmt.Errorf("db_key_base mismatch: primary and secondary keys differ (update the secondary's /etc/gitlab/gitlab.rb and run `gitlab-ctl reconfigure`)")
	}
	return nil
}

// fetchKey SSHes to host and extracts the db_key_base. It checks both
// /etc/gitlab/gitlab.rb (when explicitly set) and the generated
// /var/opt/gitlab/gitlab-rails/etc/secrets.yml (Omnibus default location).
// Returns the key value or an error. The key is never printed by callers.
func fetchKey(ctx context.Context, sshHost string) (string, error) {
	if sshHost == "" {
		return "", fmt.Errorf("ssh_host not configured")
	}

	// Try without sudo first (file may be group-readable), then with sudo.
	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "StrictHostKeyChecking=accept-new",
		sshHost,
		"grep 'db_key_base' /var/opt/gitlab/gitlab-rails/etc/secrets.yml 2>/dev/null || grep \"gitlab_rails\\['db_key_base'\\]\" /etc/gitlab/gitlab.rb 2>/dev/null || sudo grep 'db_key_base' /var/opt/gitlab/gitlab-rails/etc/secrets.yml 2>/dev/null || true",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// sudo might need a password; retry without sudo
		cmd2 := exec.CommandContext(ctx, "ssh",
			"-o", "StrictHostKeyChecking=accept-new",
			sshHost,
			"grep 'db_key_base' /var/opt/gitlab/gitlab-rails/etc/secrets.yml 2>/dev/null || grep \"gitlab_rails\\['db_key_base'\\]\" /etc/gitlab/gitlab.rb 2>/dev/null || true",
		)
		out, err = cmd2.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("ssh %s: %w", sshHost, err)
		}
	}
	m := dbKeyRe.FindSubmatch(out)
	if m == nil {
		// Try the YAML format (db_key_base: <key>)
		yamlRe := regexp.MustCompile(`db_key_base:\s*['"]?([A-Za-z0-9_-]+)`)
		m = yamlRe.FindSubmatch(out)
	}
	if m == nil {
		return "", fmt.Errorf("db_key_base not found in secrets.yml or gitlab.rb on %s", sshHost)
	}
	return strings.TrimSpace(string(m[1])), nil
}