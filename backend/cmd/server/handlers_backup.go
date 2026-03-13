package server

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func (a *App) handleDownloadDatabaseBackup(w http.ResponseWriter, r *http.Request) {
	dbPath := strings.TrimSpace(a.cfg.DBPath)
	if dbPath == "" || strings.EqualFold(dbPath, ":memory:") {
		writeError(w, http.StatusBadRequest, "database backup is unavailable for in-memory database")
		return
	}

	tempDir, err := os.MkdirTemp("", "itdb-db-backup-*")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.RemoveAll(tempDir)

	backupPath := filepath.Join(tempDir, "itdb.db")
	escapedBackupPath := strings.ReplaceAll(backupPath, "'", "''")
	if _, err := a.db.ExecContext(r.Context(), fmt.Sprintf("VACUUM INTO '%s'", escapedBackupPath)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	file, err := os.Open(backupPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fileName := fmt.Sprintf("itdb-%s.db", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	http.ServeContent(w, r, fileName, info.ModTime(), file)
}

func (a *App) handleDownloadFullBackup(w http.ResponseWriter, r *http.Request) {
	projectRoot, err := locateProjectRoot(a.cfg.DBPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fileName := fmt.Sprintf("itdb-%s.tar.gz", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))

	gzipWriter := gzip.NewWriter(w)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	if err := writeTarDirectoryHeader(tarWriter, "itdb/"); err != nil {
		return
	}

	walkErr := filepath.WalkDir(projectRoot, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(projectRoot, currentPath)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		relPath = filepath.ToSlash(relPath)
		if shouldSkipFullBackupPath(relPath) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = path.Join("itdb", relPath)
		if entry.IsDir() {
			if !strings.HasSuffix(header.Name, "/") {
				header.Name += "/"
			}
			return tarWriter.WriteHeader(header)
		}

		if !entry.Type().IsRegular() {
			return nil
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		file, err := os.Open(currentPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(tarWriter, file)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
	if walkErr != nil {
		if !errors.Is(walkErr, r.Context().Err()) {
			log.Printf("full backup stream failed: %v", walkErr)
		}
		return
	}
}

func shouldSkipFullBackupPath(relPath string) bool {
	switch {
	case relPath == "frontend/node_modules":
		return true
	case strings.HasPrefix(relPath, "frontend/node_modules/"):
		return true
	case relPath == "frontend/dist":
		return true
	case strings.HasPrefix(relPath, "frontend/dist/"):
		return true
	default:
		return false
	}
}

func writeTarDirectoryHeader(tw *tar.Writer, name string) error {
	header := &tar.Header{
		Name:     name,
		Mode:     0o755,
		Typeflag: tar.TypeDir,
		ModTime:  time.Now(),
	}
	return tw.WriteHeader(header)
}

func locateProjectRoot(dbPath string) (string, error) {
	candidates := make([]string, 0, 8)
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd, filepath.Dir(wd))
	}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates, exeDir, filepath.Dir(exeDir))
	}
	if absDBPath, err := filepath.Abs(strings.TrimSpace(dbPath)); err == nil {
		dbDir := filepath.Dir(absDBPath)
		candidates = append(candidates, dbDir, filepath.Dir(dbDir), filepath.Dir(filepath.Dir(dbDir)))
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if isProjectRoot(candidate) {
			return candidate, nil
		}
	}

	return "", errors.New("project root not found")
}

func isProjectRoot(dir string) bool {
	frontendInfo, frontendErr := os.Stat(filepath.Join(dir, "frontend"))
	backendInfo, backendErr := os.Stat(filepath.Join(dir, "backend"))
	return frontendErr == nil && backendErr == nil && frontendInfo.IsDir() && backendInfo.IsDir()
}
