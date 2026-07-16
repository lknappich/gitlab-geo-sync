package pgsetup

import (
	"context"
	"os"
	"path/filepath"
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
