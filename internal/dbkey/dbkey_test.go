package dbkey

import (
	"context"
	"testing"

	"github.com/anomalyco/gitlab-geo-sync/internal/sshexec"
)

func TestDbKeyRegexMatchesQuoted(t *testing.T) {
	line := []byte(`gitlab_rails['db_key_base'] = 'abc123def456`)
	m := dbKeyRe.FindSubmatch(line)
	if m == nil {
		t.Fatal("regex did not match quoted form")
	}
	if string(m[1]) != "abc123def456" {
		t.Errorf("got %q", m[1])
	}
}

func TestDbKeyRegexMatchesUnquoted(t *testing.T) {
	line := []byte(`gitlab_rails['db_key_base'] = abc123def456`)
	m := dbKeyRe.FindSubmatch(line)
	if m == nil {
		t.Fatal("regex did not match unquoted form")
	}
	if string(m[1]) != "abc123def456" {
		t.Errorf("got %q", m[1])
	}
}

func TestDbKeyRegexNoMatch(t *testing.T) {
	line := []byte(`# gitlab_rails['db_key_base'] = 'something'`)
	m := dbKeyRe.FindSubmatch(line)
	if m != nil {
		t.Error("regex should not match commented-out line")
	}
}

func TestDbKeyRegexMatchesLeadingSpaces(t *testing.T) {
	line := []byte(`  gitlab_rails['db_key_base'] = 'mykey123`)
	m := dbKeyRe.FindSubmatch(line)
	if m == nil {
		t.Fatal("regex did not match line with leading spaces")
	}
	if string(m[1]) != "mykey123" {
		t.Errorf("got %q", m[1])
	}
}

func TestFetchKeyEmptySSHHost(t *testing.T) {
	_, err := fetchKey(context.Background(), "", sshexec.Default)
	if err == nil {
		t.Fatal("expected error for empty ssh_host")
	}
}
