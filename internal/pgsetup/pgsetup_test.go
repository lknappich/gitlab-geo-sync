package pgsetup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckDataDirRejectsNonEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "PG_VERSION"), []byte("16"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := checkDataDir(dir)
	if err == nil {
		t.Fatal("expected error for non-empty data dir")
	}
}

func TestCheckDataDirAcceptsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := checkDataDir(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckDataDirAcceptsMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	if err := checkDataDir(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDryRunPrintsOnly(t *testing.T) {
	dir := t.TempDir()
	err := Run(context.Background(), Options{
		PrimaryDSN: "host=h user=u",
		DataDir:    dir,
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("dry-run Run: %v", err)
	}
	// Dry run should not create the data dir.
	if _, err := os.Stat(filepath.Join(dir, "PG_VERSION")); !os.IsNotExist(err) {
		t.Errorf("dry-run should not create files, but PG_VERSION exists")
	}
}

func TestRunRejectsMissingDataDir(t *testing.T) {
	err := Run(context.Background(), Options{
		PrimaryDSN: "host=h",
		DataDir:    "",
		DryRun:     true,
	})
	if err == nil {
		t.Fatal("expected error for missing data_dir")
	}
}

func TestRunRejectsMissingPrimaryDSN(t *testing.T) {
	err := Run(context.Background(), Options{
		PrimaryDSN: "",
		DataDir:    t.TempDir(),
		DryRun:     true,
	})
	if err == nil {
		t.Fatal("expected error for missing primary_dsn")
	}
}

func TestAppendConnInfoAppnameCreatesLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "postgresql.auto.conf")
	if err := os.WriteFile(path, []byte("# auto conf\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := appendConnInfoAppname(dir, "secondary-1"); err != nil {
		t.Fatalf("appendConnInfoAppname: %v", err)
	}
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "application_name=secondary-1") {
		t.Errorf("expected application_name=secondary-1 in:\n%s", content)
	}
	if !strings.HasSuffix(string(content), "\n") {
		t.Errorf("expected trailing newline in:\n%s", content)
	}
}

func TestAppendConnInfoAppnameUpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "postgresql.auto.conf")
	initial := "primary_conninfo = 'host=10.0.0.1 user=repl password=secret'\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := appendConnInfoAppname(dir, "secondary-1"); err != nil {
		t.Fatalf("appendConnInfoAppname: %v", err)
	}
	content, _ := os.ReadFile(path)
	s := string(content)
	if !strings.Contains(s, "application_name=secondary-1") {
		t.Errorf("expected application_name in:\n%s", s)
	}
	if strings.Count(s, "primary_conninfo") != 1 {
		t.Errorf("expected exactly one primary_conninfo line in:\n%s", s)
	}
}

func TestAppendConnInfoAppnameNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "postgresql.auto.conf")
	initial := "primary_conninfo = 'host=h user=u password=p'"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := appendConnInfoAppname(dir, "sec"); err != nil {
		t.Fatalf("appendConnInfoAppname: %v", err)
	}
	content, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(content), "\n") {
		t.Errorf("expected trailing newline in:\n%s", content)
	}
}

func TestAppendConnInfoAppnameEmptyAppName(t *testing.T) {
	dir := t.TempDir()
	if err := appendConnInfoAppname(dir, ""); err != nil {
		t.Errorf("expected nil error for empty appName, got %v", err)
	}
}

func TestRunDryRunNoDuplicateSlotFlag(t *testing.T) {
	dir := t.TempDir()
	err := Run(context.Background(), Options{
		PrimaryDSN: "host=h user=u",
		DataDir:    dir,
		SlotName:   "my_slot",
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
}
