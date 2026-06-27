package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupExpiredDailyBackups(t *testing.T) {
	backupDir := t.TempDir()
	now := time.Date(2026, 6, 27, 1, 0, 0, 0, time.Local)

	createFile(t, backupDir, "itdb-20260626.db")
	createFile(t, backupDir, "itdb-20260620.db")
	createFile(t, backupDir, "itdb-before-import-database-20260601-000000.db")
	createFile(t, backupDir, "notes.txt")

	if err := cleanupExpiredDailyBackups(backupDir, 3, now); err != nil {
		t.Fatalf("cleanupExpiredDailyBackups failed: %v", err)
	}

	assertExists(t, backupDir, "itdb-20260626.db")
	assertMissing(t, backupDir, "itdb-20260620.db")
	assertExists(t, backupDir, "itdb-before-import-database-20260601-000000.db")
	assertExists(t, backupDir, "notes.txt")
}

func TestCleanupExpiredDailyBackupsDisabled(t *testing.T) {
	backupDir := t.TempDir()
	now := time.Date(2026, 6, 27, 1, 0, 0, 0, time.Local)

	createFile(t, backupDir, "itdb-20250620.db")

	if err := cleanupExpiredDailyBackups(backupDir, 0, now); err != nil {
		t.Fatalf("cleanupExpiredDailyBackups failed: %v", err)
	}

	assertExists(t, backupDir, "itdb-20250620.db")
}

func createFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("backup"), 0o644); err != nil {
		t.Fatalf("create file %s failed: %v", name, err)
	}
}

func assertExists(t *testing.T, dir, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatalf("expected %s to exist: %v", name, err)
	}
}

func assertMissing(t *testing.T, dir, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, stat err=%v", name, err)
	}
}
