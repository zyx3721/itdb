package server

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// handleImportDatabase 接收上传的 .db 文件替换当前数据库
func (a *App) handleImportDatabase(w http.ResponseWriter, r *http.Request) {
	dbPath := strings.TrimSpace(a.cfg.DBPath)
	if dbPath == "" || strings.EqualFold(dbPath, ":memory:") {
		writeError(w, http.StatusBadRequest, "内存数据库不支持导入操作")
		return
	}

	// 限制上传大小 500MB
	r.Body = http.MaxBytesReader(w, r.Body, 500<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "文件过大或格式错误")
		return
	}

	uploaded, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "未找到上传文件")
		return
	}
	defer uploaded.Close()

	// 写入临时文件
	tmpFile, err := os.CreateTemp("", "itdb-import-*.db")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建临时文件失败")
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, uploaded); err != nil {
		tmpFile.Close()
		writeError(w, http.StatusInternalServerError, "保存上传文件失败")
		return
	}
	tmpFile.Close()

	// 验证上传的文件是有效的 ITDB 数据库
	if err := validateSQLiteFile(tmpPath); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 加锁，防止替换期间有其他请求访问数据库
	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "解析数据库路径失败")
		return
	}

	// 备份当前数据库到 data/backups/ 目录（VACUUM INTO）
	backupPath, err := backupDatabaseBeforeAlter(a.db, dbPath, "import-database")
	if err != nil {
		log.Printf("导入前备份失败: %v", err)
		writeError(w, http.StatusInternalServerError, "备份当前数据库失败，导入已取消")
		return
	}
	log.Printf("导入前备份完成: %s", backupPath)

	// 关闭当前数据库连接
	a.db.Close()

	// 清理 WAL 相关文件
	os.Remove(absPath + "-wal")
	os.Remove(absPath + "-shm")

	// 直接覆盖写入新文件
	if err := copyFile(tmpPath, absPath); err != nil {
		log.Printf("复制导入文件失败: %v, 正在从备份回滚", err)
		os.Remove(absPath)
		copyFile(backupPath, absPath)
		a.db, _ = sql.Open("sqlite", dbPath)
		setupSQLite(a.db)
		a.db.SetMaxOpenConns(1)
		a.db.SetMaxIdleConns(1)
		a.db.SetConnMaxLifetime(0)
		a.db.SetConnMaxIdleTime(0)
		writeError(w, http.StatusInternalServerError, "导入失败，已恢复原数据库")
		return
	}

	// 打开新数据库
	newDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Printf("打开导入数据库失败: %v, 正在从备份回滚", err)
		os.Remove(absPath)
		copyFile(backupPath, absPath)
		a.db, _ = sql.Open("sqlite", dbPath)
		setupSQLite(a.db)
		a.db.SetMaxOpenConns(1)
		a.db.SetMaxIdleConns(1)
		a.db.SetConnMaxLifetime(0)
		a.db.SetConnMaxIdleTime(0)
		writeError(w, http.StatusInternalServerError, "导入失败，已恢复原数据库")
		return
	}

	newDB.SetMaxOpenConns(1)
	newDB.SetMaxIdleConns(1)
	newDB.SetConnMaxLifetime(0)
	newDB.SetConnMaxIdleTime(0)
	if err := setupSQLite(newDB); err != nil {
		log.Printf("配置导入数据库失败: %v, 正在从备份回滚", err)
		newDB.Close()
		os.Remove(absPath)
		copyFile(backupPath, absPath)
		a.db, _ = sql.Open("sqlite", dbPath)
		setupSQLite(a.db)
		a.db.SetMaxOpenConns(1)
		a.db.SetMaxIdleConns(1)
		a.db.SetConnMaxLifetime(0)
		a.db.SetConnMaxIdleTime(0)
		writeError(w, http.StatusInternalServerError, "导入失败，已恢复原数据库")
		return
	}

	a.db = newDB

	// 检查导入的数据库是否有用户，没有则自动创建 admin
	var userCount int64
	if err := newDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount); err == nil && userCount == 0 {
		adminPass, err := hashPassword("admin123")
		if err == nil {
			newDB.Exec(`INSERT INTO users (username, userdesc, pass, usertype) VALUES (?, ?, ?, ?)`,
				"admin", "administrator", adminPass, 0)
			log.Printf("导入的数据库无用户，已自动创建 admin 账户")
		}
	}

	log.Printf("数据库导入成功，备份文件: %s", backupPath)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "数据库导入成功"})
}

// validateSQLiteFile 验证文件是有效的 ITDB SQLite 数据库
func validateSQLiteFile(path string) error {
	testDB, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("无法打开数据库文件")
	}
	defer testDB.Close()

	var result string
	if err := testDB.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		return fmt.Errorf("数据库完整性校验失败")
	}
	if result != "ok" {
		return fmt.Errorf("数据库完整性校验未通过: %s", result)
	}

	var name string
	err = testDB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='items' LIMIT 1").Scan(&name)
	if err != nil {
		return fmt.Errorf("上传的数据库缺少 items 表，不是有效的 ITDB 数据库")
	}
	return nil
}

// copyFile 跨分区安全复制文件
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
