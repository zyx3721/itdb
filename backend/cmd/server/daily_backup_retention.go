package server

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultDailyBackupRetentionDays = 0

func loadDailyBackupRetentionDays() int {
	raw := strings.TrimSpace(os.Getenv("ITDB_DAILY_BACKUP_RETENTION_DAYS"))
	if raw == "" {
		return defaultDailyBackupRetentionDays
	}

	days, err := strconv.Atoi(raw)
	if err != nil || days < 0 {
		log.Printf("Invalid ITDB_DAILY_BACKUP_RETENTION_DAYS value %q, disabling daily backup retention cleanup", raw)
		return defaultDailyBackupRetentionDays
	}
	return days
}

func cleanupExpiredDailyBackups(backupDir string, retentionDays int, now time.Time) error {
	if retentionDays <= 0 {
		return nil
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}

	cutoff := beginningOfDay(now).AddDate(0, 0, -retentionDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		backupDate, ok := parseDailyBackupDate(entry.Name())
		if !ok || !backupDate.Before(cutoff) {
			continue
		}

		path := filepath.Join(backupDir, entry.Name())
		if err := os.Remove(path); err != nil {
			return err
		}
		log.Printf("Expired daily backup removed: %s", path)
	}
	return nil
}

func beginningOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func parseDailyBackupDate(name string) (time.Time, bool) {
	if !strings.HasPrefix(name, "itdb-") || !strings.HasSuffix(name, ".db") {
		return time.Time{}, false
	}

	dateText := strings.TrimSuffix(strings.TrimPrefix(name, "itdb-"), ".db")
	backupDate, err := time.ParseInLocation("20060102", dateText, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return backupDate, true
}
