package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xuri/excelize/v2"
)

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Mode = strings.ToLower(strings.TrimSpace(req.Mode))
	if req.Mode == "" {
		req.Mode = "local"
	}

	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	var user SessionUser
	var pass string
	row := a.db.QueryRow(`SELECT id, username, pass, usertype FROM users WHERE username = ? LIMIT 1`, req.Username)
	if err := row.Scan(&user.ID, &user.Username, &pass, &user.UserType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if req.Mode == "ldap" {
		if err := authenticateLDAPUser(a.db, a.cfg.JWTSecret, req.Username, req.Password); err != nil {
			if isLDAPInvalidCredentialsError(err) {
				writeError(w, http.StatusUnauthorized, "invalid username or password")
				return
			}
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
	} else {
		ok, legacyPlaintext := verifyPassword(pass, req.Password)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		if legacyPlaintext {
			a.upgradeLegacyUserPassword(r.Context(), user.ID, req.Password)
		}
	}

	if strings.EqualFold(user.Username, "admin") {
		user.UserType = 0
	}

	expiry := time.Now().Add(48 * time.Hour)
	claims := AuthClaims{
		UserID:   user.ID,
		Username: user.Username,
		UserType: user.UserType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(a.cfg.JWTSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, authLoginResponse{Token: signed, User: user})
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := currentUser(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var userdesc string
	if err := a.db.QueryRow(`SELECT userdesc FROM users WHERE id = ?`, user.ID).Scan(&userdesc); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"userType": user.UserType,
		"userDesc": userdesc,
	})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	lookups := map[string]interface{}{}

	queries := map[string]string{
		"itemtypes":        `SELECT id, typedesc, hassoftware FROM itemtypes ORDER BY typedesc`,
		"dpttypes":         `SELECT id, dptname FROM dpttypes ORDER BY dptname`,
		"statustypes":      `SELECT id, statusdesc, color FROM statustypes ORDER BY id`,
		"filetypes":        `SELECT id, typedesc FROM filetypes ORDER BY id`,
		"contracttypes":    `SELECT id, name FROM contracttypes ORDER BY id`,
		"contractsubtypes": `SELECT id, contypeid, name FROM contractsubtypes ORDER BY name`,
		"tags": `SELECT
            tags.id,
            tags.name,
            (SELECT COUNT(*) FROM tag2item WHERE tagid = tags.id) AS itemCount,
            (SELECT COUNT(*) FROM tag2software WHERE tagid = tags.id) AS softwareCount
        FROM tags ORDER BY name`,
		"agents":        `SELECT id, type, title FROM agents ORDER BY title`,
		"users":         `SELECT id, username, usertype FROM users ORDER BY username`,
		"locations":     `SELECT id, name, floor FROM locations ORDER BY name`,
		"locareas":      `SELECT id, locationid, areaname FROM locareas ORDER BY areaname`,
		"racks":         `SELECT id, locationid, locareaid, label, usize, model, revnums FROM racks ORDER BY id`,
		"items_ref":     `SELECT id, label, model FROM items ORDER BY id DESC`,
		"software_ref":  `SELECT id, stitle, sversion FROM software ORDER BY id DESC`,
		"invoices_ref":  `SELECT id, number FROM invoices ORDER BY id DESC`,
		"contracts_ref": `SELECT id, number, title FROM contracts ORDER BY id DESC`,
		"files_ref":     `SELECT files.id, files.title, files.fname, files.date, files.type, filetypes.typedesc AS typeDesc FROM files LEFT JOIN filetypes ON filetypes.id = files.type ORDER BY files.id DESC`,
	}

	for key, q := range queries {
		rows, e := a.fetchRows(q)
		if e != nil {
			writeError(w, http.StatusInternalServerError, e.Error())
			return
		}
		lookups[key] = rows
	}

	writeJSON(w, http.StatusOK, lookups)
}

func (a *App) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	stats := map[string]int64{}
	tables := []string{"items", "software", "invoices", "contracts", "files", "agents", "users", "locations", "racks"}

	for _, table := range tables {
		query := "SELECT COUNT(*) FROM " + table
		var count int64
		if err := a.db.QueryRow(query).Scan(&count); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		stats[table] = count
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"counts": stats})
}

func (a *App) handleHistory(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	limit := intParamDefault(r.URL.Query().Get("limit"), 25)
	offset := intParamDefault(r.URL.Query().Get("offset"), 0)
	if limit <= 0 {
		limit = 25
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	where := ""
	args := []interface{}{}
	if search != "" {
		where = ` WHERE (
            CAST(id AS TEXT) LIKE ? OR
            CAST(date AS TEXT) LIKE ? OR
            authuser LIKE ? OR
            ip LIKE ? OR
            sql LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q)
	}

	countSQL := `SELECT COUNT(*) FROM history` + where
	var total int64
	if err := a.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	query := `SELECT id, date, authuser, ip, sql FROM history` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rows":   rows,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (a *App) handleExportHistory(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	where := ""
	args := []interface{}{}
	if search != "" {
		where = ` WHERE (
            CAST(id AS TEXT) LIKE ? OR
            CAST(date AS TEXT) LIKE ? OR
            authuser LIKE ? OR
            ip LIKE ? OR
            sql LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q)
	}

	query := `SELECT id, date, authuser, ip, sql FROM history` + where + ` ORDER BY id DESC`
	rows, err := a.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	const sheetName = "操作日志"
	book := excelize.NewFile()
	defaultSheet := book.GetSheetName(0)
	index, err := book.NewSheet(sheetName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	book.SetActiveSheet(index)
	if defaultSheet != "" && defaultSheet != sheetName {
		book.DeleteSheet(defaultSheet)
	}

	headers := []string{"ID", "时间", "用户", "IP", "SQL"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		book.SetCellValue(sheetName, cell, header)
	}

	styleID, err := book.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#DCEBFA"},
		},
	})
	if err == nil {
		_ = book.SetCellStyle(sheetName, "A1", "E1", styleID)
	}

	_ = book.SetColWidth(sheetName, "A", "A", 10)
	_ = book.SetColWidth(sheetName, "B", "B", 20)
	_ = book.SetColWidth(sheetName, "C", "C", 18)
	_ = book.SetColWidth(sheetName, "D", "D", 18)
	_ = book.SetColWidth(sheetName, "E", "E", 120)

	rowNum := 2
	for rows.Next() {
		var (
			id       int64
			dateTS   int64
			authuser sql.NullString
			ip       sql.NullString
			sqlText  sql.NullString
		)
		if err := rows.Scan(&id, &dateTS, &authuser, &ip, &sqlText); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		dateText := "-"
		if dateTS > 0 {
			dateText = time.Unix(dateTS, 0).In(time.Local).Format("2006-01-02 15:04:05")
		}

		values := []string{
			fmt.Sprintf("%d", id),
			dateText,
			strings.TrimSpace(authuser.String),
			strings.TrimSpace(ip.String),
			sqlText.String,
		}
		for col, value := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, rowNum)
			book.SetCellValue(sheetName, cell, value)
		}
		rowNum++
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("history_%s.xlsx", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := book.Write(w); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func (a *App) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	if err := ensureSettingsRow(a.db); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rows, err := a.fetchRows(`SELECT useldap, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter FROM settings LIMIT 1`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{})
		return
	}
	if decrypted, err := decryptSettingsSecret(asString(rows[0]["ldap_bind_password"]), a.cfg.JWTSecret); err == nil {
		rows[0]["ldap_bind_password"] = decrypted
	} else {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows[0])
}

func (a *App) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())

	var req settingsPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := ensureSettingsRow(a.db); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	encryptedBindPassword, err := encryptSettingsSecret(req.LDAPBindPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = a.execLogged(r.Context(), user, clientIP(r), `UPDATE settings SET
        useldap = ?,
        ldap_server = ?,
        ldap_dn = ?,
        ldap_bind_dn = ?,
        ldap_bind_password = ?,
        ldap_getusers = ?,
        ldap_getusers_filter = ?`,
		req.UseLDAP,
		strings.TrimSpace(req.LDAPServer),
		strings.TrimSpace(req.LDAPDN),
		strings.TrimSpace(req.LDAPBindDN),
		encryptedBindPassword,
		strings.TrimSpace(req.LDAPGetUsers),
		strings.TrimSpace(req.LDAPGetUsersFilter),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	a.handleGetSettings(w, r)
}

func (a *App) handleTestLDAPConnection(w http.ResponseWriter, r *http.Request) {
	var req settingsPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := testLDAPSettings(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "LDAP 连接成功"})
}
