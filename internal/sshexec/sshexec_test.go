package sshexec

import (
	"testing"
)

func TestEffectiveStrictHostKeyChecking(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{"default", Config{}, "accept-new"},
		{"with known_hosts", Config{KnownHostsFile: "/etc/known_hosts"}, "yes"},
		{"explicit override", Config{KnownHostsFile: "/etc/known_hosts", StrictHostKeyChecking: "no"}, "no"},
		{"explicit accept-new", Config{StrictHostKeyChecking: "accept-new"}, "accept-new"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.cfg.EffectiveStrictHostKeyChecking()
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtraArgsIncludesKnownHostsFile(t *testing.T) {
	cfg := Config{KnownHostsFile: "/etc/known_hosts"}
	args := cfg.ExtraArgs()
	found := false
	for i, a := range args {
		if a == "-o" && i+1 < len(args) && args[i+1] == "UserKnownHostsFile=/etc/known_hosts" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected UserKnownHostsFile in args, got %v", args)
	}
}

func TestExtraArgsDefaultNoKnownHostsFile(t *testing.T) {
	cfg := Config{}
	args := cfg.ExtraArgs()
	for i, a := range args {
		if a == "-o" && i+1 < len(args) && args[i+1] == "StrictHostKeyChecking=accept-new" {
			return
		}
	}
	t.Errorf("expected StrictHostKeyChecking=accept-new in args, got %v", args)
}

func TestSSHString(t *testing.T) {
	cfg := Config{KnownHostsFile: "/etc/known_hosts"}
	s := cfg.SSHString()
	if s != "ssh -o StrictHostKeyChecking=yes -o UserKnownHostsFile=/etc/known_hosts" {
		t.Errorf("got %q", s)
	}
}

func TestCheckHost(t *testing.T) {
	if err := CheckHost(""); err == nil {
		t.Error("expected error for empty host")
	}
	if err := CheckHost("example.com:22"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
