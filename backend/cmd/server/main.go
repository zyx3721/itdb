package server

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"itdb-backend/cmd/common/localizer"
	"itdb-backend/cmd/common/statustypes"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	_ "modernc.org/sqlite"
)

type contextKey string

const userKey contextKey = "itdb_user"

type Config struct {
	ServerAddr   string
	DBPath       string
	UploadDir    string
	JWTSecret    string
	HistoryLimit int64
	CORSOrigins  []string
}

type App struct {
	db   *sql.DB
	dbMu sync.Mutex
	cfg  Config
}

type SessionUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	UserType int64  `json:"userType"`
}

type AuthClaims struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	UserType int64  `json:"userType"`
	jwt.RegisteredClaims
}

type jsonError struct {
	Error string `json:"error"`
}

func Run() {
	loadDotEnvIfPresent()
	cfg := loadConfig()
	if err := ensureDatabaseInitialized(cfg); err != nil {
		log.Fatalf("数据库初始化失败: %s", localizer.LocalizeMessage(err.Error()))
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		log.Fatalf("打开数据库失败: %s", localizer.LocalizeMessage(err.Error()))
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := setupSQLite(db); err != nil {
		log.Fatalf("初始化 SQLite 参数失败: %s", localizer.LocalizeMessage(err.Error()))
	}
	if err := ensureStatusTypeColorSchema(db, cfg.DBPath); err != nil {
		log.Fatalf("状态类型字段升级失败: %s", localizer.LocalizeMessage(err.Error()))
	}

	if err := ensureItemTypeSoftwareDefaults(db); err != nil {
		log.Fatalf("硬件类型支持软件默认值初始化失败: %s", localizer.LocalizeMessage(err.Error()))
	}

	if err := ensureViewHistorySchema(db); err != nil {
		log.Fatalf("最近历史记录表初始化失败: %s", localizer.LocalizeMessage(err.Error()))
	}
	if err := ensureSettingsSchema(db, cfg.DBPath); err != nil {
		log.Fatalf("系统设置表升级失败: %s", localizer.LocalizeMessage(err.Error()))
	}
	if err := ensureSettingsSecretsEncrypted(db, cfg.JWTSecret); err != nil {
		log.Fatalf("系统设置敏感信息加密失败: %s", localizer.LocalizeMessage(err.Error()))
	}

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("创建上传目录失败: %s", localizer.LocalizeMessage(err.Error()))
	}

	app := &App{db: db, cfg: cfg}
	router := app.routes()

	go app.startDailyBackup()

	addr := cfg.ServerAddr
	log.Printf("ITDB Go API 已启动，监听地址: %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("服务运行失败: %s", localizer.LocalizeMessage(err.Error()))
	}
}

func loadConfig() Config {
	history := int64(1000)
	if s := os.Getenv("ITDB_HISTORY_LIMIT"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil && v > 0 {
			history = v
		}
	}

	return Config{
		ServerAddr:   loadServerAddr(),
		DBPath:       getenv("ITDB_DB_PATH", "./data/itdb.db"),
		UploadDir:    getenv("ITDB_UPLOAD_DIR", "./data/files"),
		JWTSecret:    getenv("ITDB_JWT_SECRET", "itdb-change-me"),
		HistoryLimit: history,
		CORSOrigins:  parseCSV(getenv("ITDB_CORS_ORIGINS", "*")),
	}
}

func loadServerAddr() string {
	if addr := strings.TrimSpace(os.Getenv("ITDB_SERVER_ADDR")); addr != "" {
		return addr
	}
	if addr := strings.TrimSpace(os.Getenv("ADDR")); addr != "" {
		return addr
	}
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	return net.JoinHostPort("127.0.0.1", port)
}

func loadDotEnvIfPresent() {
	seen := map[string]struct{}{}
	candidates := []string{".env", filepath.Join("backend", ".env")}

	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), ".env"))
	}

	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if err := loadDotEnvFile(candidate); err == nil {
			return
		}
	}
}

func loadDotEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) || (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}
		if strings.TrimSpace(os.Getenv(key)) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func setupSQLite(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA case_sensitive_like = 0;",
		"PRAGMA encoding = \"UTF-8\";",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return err
		}
	}
	return nil
}

func ensureStatusTypeColorSchema(db *sql.DB, dbPath string) error {
	hasColor, err := sqliteColumnExists(db, "statustypes", "color")
	if err != nil {
		return err
	}
	if !hasColor {
		backupPath, err := backupDatabaseBeforeAlter(db, dbPath, "statustypes-color")
		if err != nil {
			return err
		}
		log.Printf("数据库备份完成: %s", backupPath)

		if _, err := db.Exec(`ALTER TABLE statustypes ADD COLUMN color TEXT`); err != nil {
			return err
		}
	}

	for desc, color := range statustypes.FixedStatusTypeColors() {
		if _, err := db.Exec(`UPDATE statustypes SET color = ? WHERE TRIM(statusdesc) = ?`, color, desc); err != nil {
			return err
		}
	}
	return nil
}

func ensureItemTypeSoftwareDefaults(db *sql.DB) error {
	var total, supported int64
	if err := db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(CASE WHEN COALESCE(hassoftware, 0) = 1 THEN 1 ELSE 0 END), 0) FROM itemtypes`).Scan(&total, &supported); err != nil {
		return err
	}
	if total > 0 && supported == 0 {
		if _, err := db.Exec(`UPDATE itemtypes SET hassoftware = 1`); err != nil {
			return err
		}
		log.Printf("已将硬件类型“支持软件”默认值更新为“是”")
	}
	return nil
}

func ensureViewHistorySchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS viewhist (id INTEGER PRIMARY KEY AUTOINCREMENT, url, description)`)
	return err
}

func ensureSettingsSchema(db *sql.DB, dbPath string) error {
	legacyColumns := []string{"companytitle", "dateformat", "currency", "lang", "version", "timezone", "dbversion"}
	currentColumns := []string{"useldap", "ldap_server", "ldap_dn", "ldap_bind_dn", "ldap_bind_password", "ldap_getusers", "ldap_getusers_filter"}
	tableExists, err := sqliteTableExists(db, "settings")
	if err != nil {
		return err
	}

	hasColumns := map[string]bool{}
	needsMigration := false
	if tableExists {
		for _, columnName := range append(append([]string{}, legacyColumns...), currentColumns...) {
			hasColumn, err := sqliteColumnExists(db, "settings", columnName)
			if err != nil {
				return err
			}
			hasColumns[columnName] = hasColumn
		}
		for _, columnName := range legacyColumns {
			if hasColumns[columnName] {
				needsMigration = true
				break
			}
		}
		if !needsMigration {
			for _, columnName := range currentColumns {
				if !hasColumns[columnName] {
					needsMigration = true
					break
				}
			}
		}
	}

	if tableExists && needsMigration {
		trimmedDBPath := strings.TrimSpace(dbPath)
		if trimmedDBPath != "" && !strings.EqualFold(trimmedDBPath, ":memory:") {
			backupPath, err := backupDatabaseBeforeAlter(db, dbPath, "settings-schema")
			if err != nil {
				return err
			}
			log.Printf("数据库备份完成: %s", backupPath)
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`CREATE TABLE settings_new (useldap integer default 0, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter)`); err != nil {
			return err
		}

		useLDAPExpr := "0"
		if hasColumns["useldap"] {
			useLDAPExpr = "COALESCE(useldap, 0)"
		}
		ldapServerExpr := "''"
		if hasColumns["ldap_server"] {
			ldapServerExpr = "COALESCE(ldap_server, '')"
		}
		ldapDNExpr := "''"
		if hasColumns["ldap_dn"] {
			ldapDNExpr = "COALESCE(ldap_dn, '')"
		}
		ldapBindDNExpr := "''"
		if hasColumns["ldap_bind_dn"] {
			ldapBindDNExpr = "COALESCE(ldap_bind_dn, '')"
		}
		ldapBindPasswordExpr := "''"
		if hasColumns["ldap_bind_password"] {
			ldapBindPasswordExpr = "COALESCE(ldap_bind_password, '')"
		}
		ldapGetUsersExpr := "''"
		if hasColumns["ldap_getusers"] {
			ldapGetUsersExpr = "COALESCE(ldap_getusers, '')"
		}
		ldapGetUsersFilterExpr := "''"
		if hasColumns["ldap_getusers_filter"] {
			ldapGetUsersFilterExpr = "COALESCE(ldap_getusers_filter, '')"
		}

		copyQuery := fmt.Sprintf(`INSERT INTO settings_new (useldap, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter)
			SELECT %s, %s, %s, %s, %s, %s, %s
			FROM settings
			LIMIT 1`, useLDAPExpr, ldapServerExpr, ldapDNExpr, ldapBindDNExpr, ldapBindPasswordExpr, ldapGetUsersExpr, ldapGetUsersFilterExpr)
		if _, err := tx.Exec(copyQuery); err != nil {
			return err
		}
		if _, err := tx.Exec(`DROP TABLE settings`); err != nil {
			return err
		}
		if _, err := tx.Exec(`ALTER TABLE settings_new RENAME TO settings`); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return ensureSettingsRow(db)
}

func ensureSettingsRow(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS settings (useldap integer default 0, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter)`); err != nil {
		return err
	}

	var count int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM settings`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	_, err := db.Exec(`INSERT INTO settings (useldap, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter) VALUES (0, '', '', '', '', '', '')`)
	return err
}

func ensureSettingsSecretsEncrypted(db *sql.DB, cipherKey string) error {
	var (
		rowID       int64
		bindPassRaw sql.NullString
	)
	err := db.QueryRow(`SELECT rowid, ldap_bind_password FROM settings LIMIT 1`).Scan(&rowID, &bindPassRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	encrypted, changed, err := encryptSettingsSecretIfNeeded(bindPassRaw.String, cipherKey)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	_, err = db.Exec(`UPDATE settings SET ldap_bind_password = ? WHERE rowid = ?`, encrypted, rowID)
	return err
}

func sqliteTableExists(db *sql.DB, tableName string) (bool, error) {
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1`, tableName).Scan(&name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func sqliteColumnExists(db *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid      int
			name     string
			colType  string
			notNull  int
			defaultV sql.NullString
			pk       int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultV, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, columnName) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func backupDatabaseBeforeAlter(db *sql.DB, dbPath, tag string) (string, error) {
	if strings.TrimSpace(dbPath) == "" || strings.EqualFold(strings.TrimSpace(dbPath), ":memory:") {
		return "", errors.New("内存数据库无法生成文件备份")
	}

	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return "", err
	}
	baseName := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	if baseName == "" {
		baseName = "itdb"
	}
	backupDir := filepath.Join(filepath.Dir(absPath), "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}

	stamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s-before-%s-%s.db", baseName, tag, stamp))
	escapedBackupPath := strings.ReplaceAll(backupPath, "'", "''")
	backupSQL := fmt.Sprintf("VACUUM INTO '%s'", escapedBackupPath)
	if _, err := db.Exec(backupSQL); err != nil {
		return "", err
	}
	return backupPath, nil
}

// startDailyBackup 每天 0 点自动备份数据库
func (a *App) startDailyBackup() {
	dbPath := strings.TrimSpace(a.cfg.DBPath)
	if dbPath == "" || strings.EqualFold(dbPath, ":memory:") {
		log.Printf("内存数据库，跳过每日自动备份")
		return
	}

	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		time.Sleep(next.Sub(now))

		a.dbMu.Lock()
		absPath, err := filepath.Abs(dbPath)
		if err != nil {
			a.dbMu.Unlock()
			log.Printf("每日备份: 解析路径失败: %v", err)
			continue
		}

		backupDir := filepath.Join(filepath.Dir(absPath), "backups")
		os.MkdirAll(backupDir, 0o755)
		stamp := time.Now().Format("20060102")
		backupPath := filepath.Join(backupDir, fmt.Sprintf("itdb-%s.db", stamp))

		escapedPath := strings.ReplaceAll(backupPath, "'", "''")
		_, err = a.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", escapedPath))
		a.dbMu.Unlock()

		if err != nil {
			log.Printf("每日备份失败: %v", err)
		} else {
			log.Printf("每日备份完成: %s", backupPath)
		}
	}
}

func (a *App) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(a.corsMiddleware)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Post("/api/auth/login", a.handleLogin)
	r.Group(func(r chi.Router) {
		r.Use(a.authMiddleware)
		r.Get("/api/auth/me", a.handleMe)
		r.Post("/api/auth/logout", a.handleLogout)

		r.Get("/api/bootstrap", a.handleBootstrap)
		r.Get("/api/dashboard/summary", a.handleDashboardSummary)
		r.Get("/api/history", a.handleHistory)
		r.Get("/api/history/export", a.handleExportHistory)
		r.Get("/api/view-history", a.handleListViewHistory)
		r.Post("/api/view-history", a.handleCreateViewHistory)
		r.Get("/api/backups/database", a.handleDownloadDatabaseBackup)
		r.Get("/api/backups/full", a.handleDownloadFullBackup)

		r.Get("/api/settings", a.handleGetSettings)
		r.With(a.requireWriteMiddleware).Put("/api/settings", a.handleUpdateSettings)
		r.With(a.requireWriteMiddleware).Post("/api/settings/test-ldap", a.handleTestLDAPConnection)

		r.Get("/api/items", a.handleListItems)
		r.Get("/api/items/{id}", a.handleGetItem)
		r.Get("/api/items/{id}/actions", a.handleListItemActions)
		r.With(a.requireWriteMiddleware).Post("/api/items", a.handleCreateItem)
		r.With(a.requireWriteMiddleware).Put("/api/items/{id}", a.handleUpdateItem)
		r.With(a.requireWriteMiddleware).Delete("/api/items/{id}", a.handleDeleteItem)
		r.With(a.requireWriteMiddleware).Post("/api/items/{id}/tags", a.handleMutateItemTag)
		r.With(a.requireWriteMiddleware).Post("/api/items/{id}/actions", a.handleCreateItemAction)
		r.With(a.requireWriteMiddleware).Put("/api/items/{id}/actions/{actionId}", a.handleUpdateItemAction)
		r.With(a.requireWriteMiddleware).Delete("/api/items/{id}/actions/{actionId}", a.handleDeleteItemAction)

		r.Get("/api/software", a.handleListSoftware)
		r.Get("/api/software/{id}", a.handleGetSoftware)
		r.With(a.requireWriteMiddleware).Post("/api/software", a.handleCreateSoftware)
		r.With(a.requireWriteMiddleware).Put("/api/software/{id}", a.handleUpdateSoftware)
		r.With(a.requireWriteMiddleware).Delete("/api/software/{id}", a.handleDeleteSoftware)
		r.With(a.requireWriteMiddleware).Post("/api/software/{id}/tags", a.handleMutateSoftwareTag)

		r.Get("/api/invoices", a.handleListInvoices)
		r.Get("/api/invoices/{id}", a.handleGetInvoice)
		r.With(a.requireWriteMiddleware).Post("/api/invoices", a.handleCreateInvoice)
		r.With(a.requireWriteMiddleware).Put("/api/invoices/{id}", a.handleUpdateInvoice)
		r.With(a.requireWriteMiddleware).Delete("/api/invoices/{id}", a.handleDeleteInvoice)

		r.Get("/api/contracts", a.handleListContracts)
		r.Get("/api/contracts/{id}", a.handleGetContract)
		r.With(a.requireWriteMiddleware).Post("/api/contracts", a.handleCreateContract)
		r.With(a.requireWriteMiddleware).Put("/api/contracts/{id}", a.handleUpdateContract)
		r.With(a.requireWriteMiddleware).Delete("/api/contracts/{id}", a.handleDeleteContract)
		r.Get("/api/contracts/{id}/events", a.handleListContractEvents)
		r.With(a.requireWriteMiddleware).Post("/api/contracts/{id}/events", a.handleCreateContractEvent)
		r.With(a.requireWriteMiddleware).Put("/api/contracts/{id}/events/{eventId}", a.handleUpdateContractEvent)
		r.With(a.requireWriteMiddleware).Delete("/api/contracts/{id}/events/{eventId}", a.handleDeleteContractEvent)

		r.Get("/api/files", a.handleListFiles)
		r.Get("/api/files/{id}", a.handleGetFile)
		r.Get("/api/files/{id}/download", a.handleDownloadFile)
		r.With(a.requireWriteMiddleware).Post("/api/files", a.handleCreateFile)
		r.With(a.requireWriteMiddleware).Put("/api/files/{id}", a.handleUpdateFile)
		r.With(a.requireWriteMiddleware).Delete("/api/files/{id}", a.handleDeleteFile)

		r.Get("/api/agents", a.handleListAgents)
		r.Get("/api/agents/{id}", a.handleGetAgent)
		r.With(a.requireWriteMiddleware).Post("/api/agents", a.handleCreateAgent)
		r.With(a.requireWriteMiddleware).Put("/api/agents/{id}", a.handleUpdateAgent)
		r.With(a.requireWriteMiddleware).Delete("/api/agents/{id}", a.handleDeleteAgent)

		r.Get("/api/users", a.handleListUsers)
		r.Get("/api/users/{id}", a.handleGetUser)
		r.With(a.requireWriteMiddleware).Post("/api/users", a.handleCreateUser)
		r.With(a.requireWriteMiddleware).Put("/api/users/{id}", a.handleUpdateUser)
		r.With(a.requireWriteMiddleware).Delete("/api/users/{id}", a.handleDeleteUser)

		r.Get("/api/locations", a.handleListLocations)
		r.Get("/api/locations/{id}", a.handleGetLocation)
		r.Get("/api/locations/{id}/floorplan", a.handleDownloadLocationFloorplan)
		r.With(a.requireWriteMiddleware).Post("/api/locations", a.handleCreateLocation)
		r.With(a.requireWriteMiddleware).Put("/api/locations/{id}", a.handleUpdateLocation)
		r.With(a.requireWriteMiddleware).Delete("/api/locations/{id}", a.handleDeleteLocation)
		r.Get("/api/locations/{id}/areas", a.handleListLocAreas)
		r.With(a.requireWriteMiddleware).Post("/api/locations/{id}/areas", a.handleCreateLocArea)
		r.With(a.requireWriteMiddleware).Put("/api/locations/{id}/areas/{areaId}", a.handleUpdateLocArea)
		r.With(a.requireWriteMiddleware).Delete("/api/locations/{id}/areas/{areaId}", a.handleDeleteLocArea)

		r.Get("/api/racks", a.handleListRacks)
		r.Get("/api/racks/{id}", a.handleGetRack)
		r.With(a.requireWriteMiddleware).Post("/api/racks", a.handleCreateRack)
		r.With(a.requireWriteMiddleware).Put("/api/racks/{id}", a.handleUpdateRack)
		r.With(a.requireWriteMiddleware).Delete("/api/racks/{id}", a.handleDeleteRack)

		r.Get("/api/dictionaries", a.handleListDictionaries)
		r.With(a.requireWriteMiddleware).Post("/api/dictionaries/{name}", a.handleCreateDictionaryRow)
		r.With(a.requireWriteMiddleware).Put("/api/dictionaries/{name}/{id}", a.handleUpdateDictionaryRow)
		r.With(a.requireWriteMiddleware).Delete("/api/dictionaries/{name}/{id}", a.handleDeleteDictionaryRow)

		r.Get("/api/tags", a.handleListTags)
		r.Get("/api/tags/suggest", a.handleSuggestTags)
		r.Get("/api/tags/{id}/items", a.handleListTagItems)
		r.Get("/api/tags/{id}/software", a.handleListTagSoftware)
		r.With(a.requireWriteMiddleware).Post("/api/tags", a.handleCreateTag)
		r.With(a.requireWriteMiddleware).Put("/api/tags/{id}", a.handleUpdateTag)
		r.With(a.requireWriteMiddleware).Delete("/api/tags/{id}", a.handleDeleteTag)

		r.Get("/api/reports", a.handleListReports)
		r.Get("/api/reports/{name}", a.handleRunReport)

		r.With(a.requireWriteMiddleware, middleware.Timeout(300*time.Second)).Post("/api/import/database", a.handleImportDatabase)

		r.Get("/api/browse/tree", a.handleBrowseTree)
		r.Get("/api/labels/items", a.handleListLabelItems)
		r.Get("/api/labels/presets", a.handleListLabelPresets)
		r.Post("/api/labels/preview", a.handlePreviewLabels)
		r.With(a.requireWriteMiddleware).Post("/api/labels/presets", a.handleCreateLabelPreset)
		r.With(a.requireWriteMiddleware).Delete("/api/labels/presets/{id}", a.handleDeleteLabelPreset)
	})

	return r
}

func (a *App) corsMiddleware(next http.Handler) http.Handler {
	allowAll := false
	allowed := map[string]struct{}{}
	for _, origin := range a.cfg.CORSOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
			break
		}
		allowed[origin] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if auth == "" {
			writeError(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}

		tokenParts := strings.SplitN(auth, " ", 2)
		if len(tokenParts) != 2 || !strings.EqualFold(tokenParts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "invalid Authorization header")
			return
		}

		tokenStr := strings.TrimSpace(tokenParts[1])
		claims := &AuthClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(a.cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		user := SessionUser{ID: claims.UserID, Username: claims.Username, UserType: claims.UserType}
		ctx := context.WithValue(r.Context(), userKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) requireWriteMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := currentUser(r.Context())
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		if user.UserType != 0 && !strings.EqualFold(user.Username, "admin") {
			writeError(w, http.StatusForbidden, "read-only user")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func currentUser(ctx context.Context) (SessionUser, error) {
	raw := ctx.Value(userKey)
	if raw == nil {
		return SessionUser{}, errors.New("missing user")
	}
	user, ok := raw.(SessionUser)
	if !ok {
		return SessionUser{}, errors.New("invalid user context")
	}
	return user, nil
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseCSV(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, jsonError{Error: localizer.LocalizeMessage(message)})
}

func (a *App) execLogged(ctx context.Context, user SessionUser, ip string, query string, args ...interface{}) (sql.Result, error) {
	res, err := a.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	normalized := strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(normalized, "SELECT") {
		return res, nil
	}

	if user.Username == "" {
		user.Username = "unknown"
	}

	histSQL := query
	if len(args) > 0 {
		histSQL = fmt.Sprintf("%s | args=%v", query, args)
	}

	now := time.Now().Unix()
	_, _ = a.db.ExecContext(ctx, `INSERT INTO history (date, sql, authuser, ip) VALUES (?, ?, ?, ?)`, now, histSQL, user.Username, ip)

	if a.cfg.HistoryLimit > 0 {
		_, _ = a.db.ExecContext(ctx, `DELETE FROM history WHERE id < (SELECT COALESCE(MAX(id), 0) - ? FROM history)`, a.cfg.HistoryLimit)
	}

	return res, nil
}
