package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func (a *App) handleListFiles(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	baseQuery := `SELECT files.id, files.title, files.fname, files.type, files.date, filetypes.typedesc,
        (SELECT COUNT(*) FROM software2file WHERE fileid = files.id) +
        (SELECT COUNT(*) FROM invoice2file WHERE fileid = files.id) +
        (SELECT COUNT(*) FROM item2file WHERE fileid = files.id) +
        (SELECT COUNT(*) FROM contract2file WHERE fileid = files.id) AS links
        FROM files
        LEFT JOIN filetypes ON filetypes.id = files.type`

	where := ""
	args := []interface{}{}
	if search != "" {
		where = `WHERE (
            CAST(file_view.id AS TEXT) LIKE ? OR
            file_view.typedesc LIKE ? OR
            file_view.title LIKE ? OR
            file_view.fname LIKE ? OR
            (CASE WHEN COALESCE(file_view.date, 0) > 0 THEN date(file_view.date, 'unixepoch') ELSE '' END) LIKE ? OR
            CAST(file_view.links AS TEXT) LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q, q)
	}

	query := `SELECT * FROM (` + baseQuery + `) AS file_view
        ` + where + `
        ORDER BY file_view.id DESC`
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleGetFile(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	rows, err := a.fetchRows(`SELECT * FROM files WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	payload := rows[0]
	payload["itemLinks"], _ = a.fetchIDList(`SELECT itemid FROM item2file WHERE fileid = ?`, id)
	payload["softwareLinks"], _ = a.fetchIDList(`SELECT softwareid FROM software2file WHERE fileid = ?`, id)
	payload["contractLinks"], _ = a.fetchIDList(`SELECT contractid FROM contract2file WHERE fileid = ?`, id)
	payload["invoiceLinks"], _ = a.fetchIDList(`SELECT invoiceid FROM invoice2file WHERE fileid = ?`, id)

	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var storedName, title sql.NullString
	if err := a.db.QueryRow(`SELECT fname, title FROM files WHERE id = ?`, id).Scan(&storedName, &title); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	name := strings.TrimSpace(storedName.String)
	if name == "" {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	path := filepath.Join(a.cfg.UploadDir, name)
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	downloadName := name
	if t := strings.TrimSpace(title.String); t != "" {
		downloadName = fmt.Sprintf("%s%s", t, filepath.Ext(name))
	}
	setContentDispositionHeader(w, "attachment", downloadName)
	http.ServeFile(w, r, path)
}

func (a *App) handleCreateFile(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	typeID, _ := intParam(strings.TrimSpace(r.FormValue("typeId")))
	dateRaw := strings.TrimSpace(r.FormValue("date"))
	fileHeader, err := getMultipartFileHeader(r.MultipartForm, "file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if title == "" || typeID <= 0 || dateRaw == "" {
		writeError(w, http.StatusBadRequest, "title, typeId and date are required")
		return
	}

	dateValue, err := parseDateInput(dateRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date")
		return
	}

	storedName, err := a.storeUploadedFile(fileHeader, typeID, title)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO files (title, type, fname, uploader, uploaddate, date) VALUES (?, ?, ?, ?, ?, ?)`,
		title, typeID, storedName, user.Username, time.Now().Unix(), dateValue)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := res.LastInsertId()

	itemLinks := parseIDCSV(r.FormValue("itemLinks"))
	softwareLinks := parseIDCSV(r.FormValue("softwareLinks"))
	invoiceLinks := parseIDCSV(r.FormValue("invoiceLinks"))
	contractLinks := parseIDCSV(r.FormValue("contractLinks"))

	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2file WHERE fileid = ?`, `INSERT INTO item2file (fileid, itemid) VALUES (?, ?)`, newID, itemLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM software2file WHERE fileid = ?`, `INSERT INTO software2file (fileid, softwareid) VALUES (?, ?)`, newID, softwareLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM invoice2file WHERE fileid = ?`, `INSERT INTO invoice2file (fileid, invoiceid) VALUES (?, ?)`, newID, invoiceLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2file WHERE fileid = ?`, `INSERT INTO contract2file (fileid, contractid) VALUES (?, ?)`, newID, contractLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE files SET id = id WHERE id = ?", newID)
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *App) handleUpdateFile(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	typeID, _ := intParam(strings.TrimSpace(r.FormValue("typeId")))
	dateRaw := strings.TrimSpace(r.FormValue("date"))
	if title == "" || typeID <= 0 || dateRaw == "" {
		writeError(w, http.StatusBadRequest, "title, typeId and date are required")
		return
	}
	dateValue, err := parseDateInput(dateRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date")
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE files SET title = ?, type = ?, uploader = ?, uploaddate = ?, date = ? WHERE id = ?`,
		title, typeID, user.Username, time.Now().Unix(), dateValue, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fileHeader, _ := getMultipartFileHeader(r.MultipartForm, "file")
	if fileHeader != nil {
		var oldName string
		_ = tx.QueryRow(`SELECT fname FROM files WHERE id = ?`, id).Scan(&oldName)

		storedName, err := a.storeUploadedFile(fileHeader, typeID, title)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if _, err := tx.Exec(`UPDATE files SET fname = ? WHERE id = ?`, storedName, id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if strings.TrimSpace(oldName) != "" {
			_ = os.Remove(filepath.Join(a.cfg.UploadDir, oldName))
		}
	}

	itemLinks := parseIDCSV(r.FormValue("itemLinks"))
	softwareLinks := parseIDCSV(r.FormValue("softwareLinks"))
	invoiceLinks := parseIDCSV(r.FormValue("invoiceLinks"))
	contractLinks := parseIDCSV(r.FormValue("contractLinks"))
	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2file WHERE fileid = ?`, `INSERT INTO item2file (fileid, itemid) VALUES (?, ?)`, id, itemLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM software2file WHERE fileid = ?`, `INSERT INTO software2file (fileid, softwareid) VALUES (?, ?)`, id, softwareLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM invoice2file WHERE fileid = ?`, `INSERT INTO invoice2file (fileid, invoiceid) VALUES (?, ?)`, id, invoiceLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2file WHERE fileid = ?`, `INSERT INTO contract2file (fileid, contractid) VALUES (?, ?)`, id, contractLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE files SET id = id WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *App) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	links, err := a.countFileLinksTx(tx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if links > 0 {
		writeError(w, http.StatusConflict, "file still has associations")
		return
	}
	if err := a.deleteFileTx(tx, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = a.execLogged(r.Context(), user, clientIP(r), "DELETE FROM files WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleListAgents(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	baseQuery := `SELECT
        agents.*,
        TRIM(
            CASE WHEN (agents.type & 1) = 1 THEN '采购方 Buyer ' ELSE '' END ||
            CASE WHEN (agents.type & 2) = 2 THEN '软件厂商 S/W Manufacturer ' ELSE '' END ||
            CASE WHEN (agents.type & 8) = 8 THEN '硬件厂商 H/W Manufacturer ' ELSE '' END ||
            CASE WHEN (agents.type & 4) = 4 THEN '供应商 Vendor ' ELSE '' END ||
            CASE WHEN (agents.type & 16) = 16 THEN '承包方 Contractor ' ELSE '' END
        ) AS typetext
    FROM agents`

	where := ""
	args := []interface{}{}
	if search != "" {
		where = `WHERE (
            CAST(agent_view.id AS TEXT) LIKE ? OR
            agent_view.typetext LIKE ? OR
            agent_view.title LIKE ? OR
            agent_view.contactinfo LIKE ? OR
            agent_view.contacts LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q)
	}

	query := `SELECT * FROM (` + baseQuery + `) AS agent_view
        ` + where + `
        ORDER BY agent_view.title, agent_view.type, agent_view.id`
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rows, err := a.fetchRows(`SELECT * FROM agents WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, rows[0])
}

func (a *App) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	var req agentPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	typeMask := reqTypeMask(req)
	contacts := encodeAgentContacts(req.Contacts)
	urls := encodeAgentURLs(req.URLs)

	res, err := a.execLogged(r.Context(), user, clientIP(r), `INSERT INTO agents (type, title, contactinfo, contacts, urls) VALUES (?, ?, ?, ?, ?)`,
		typeMask, req.Title, req.ContactInfo, contacts, urls)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *App) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req agentPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	typeMask := reqTypeMask(req)
	contacts := encodeAgentContacts(req.Contacts)
	urls := encodeAgentURLs(req.URLs)

	_, err = a.execLogged(r.Context(), user, clientIP(r), `UPDATE agents SET type = ?, title = ?, contactinfo = ?, contacts = ?, urls = ? WHERE id = ?`,
		typeMask, req.Title, req.ContactInfo, contacts, urls, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *App) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	statements := []string{
		`DELETE FROM agents WHERE id = ?`,
		`UPDATE items SET manufacturerid = '' WHERE manufacturerid = ?`,
		`UPDATE invoices SET vendorid = '' WHERE vendorid = ?`,
		`UPDATE invoices SET buyerid = '' WHERE buyerid = ?`,
		`UPDATE software SET manufacturerid = '' WHERE manufacturerid = ?`,
	}
	for _, st := range statements {
		if _, err := tx.Exec(st, id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = a.execLogged(r.Context(), user, clientIP(r), "DELETE FROM agents WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleListUsers(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	baseQuery := `SELECT users.id, users.username, users.userdesc, users.usertype,
        CASE users.usertype
            WHEN 1 THEN '只读 ReadOnly'
            ELSE '完全访问 FullAccess'
        END AS usertypeText,
        (SELECT COUNT(*) FROM items WHERE items.userid = users.id) AS itemCount
        FROM users`

	where := ""
	args := []interface{}{}
	if search != "" {
		where = `WHERE (
            CAST(user_view.id AS TEXT) LIKE ? OR
            user_view.username LIKE ? OR
            user_view.userdesc LIKE ? OR
            user_view.usertypeText LIKE ? OR
            CAST(user_view.itemCount AS TEXT) LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q)
	}

	query := `SELECT * FROM (` + baseQuery + `) AS user_view
        ` + where + `
        ORDER BY user_view.username ASC`
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rows, err := a.fetchRows(`SELECT id, username, userdesc, '' AS pass, cookie1, usertype FROM users WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, rows[0])
}

func (a *App) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	var req userPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}
	if strings.EqualFold(req.Username, "admin") {
		req.UserType = 0
	}
	hashedPass, err := hashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	res, err := a.execLogged(r.Context(), user, clientIP(r), `INSERT INTO users (username, userdesc, pass, usertype) VALUES (?, ?, ?, ?)`,
		req.Username, req.UserDesc, hashedPass, req.UserType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *App) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req userPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}

	var exists int64
	if err := a.db.QueryRow(`SELECT COUNT(id) FROM users WHERE username = ? AND id <> ?`, req.Username, id).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if exists > 0 {
		writeError(w, http.StatusConflict, "username already exists")
		return
	}
	if strings.EqualFold(req.Username, "admin") {
		req.UserType = 0
	}
	if strings.TrimSpace(req.Password) != "" {
		hashedPass, hashErr := hashPassword(req.Password)
		if hashErr != nil {
			writeError(w, http.StatusInternalServerError, hashErr.Error())
			return
		}
		_, err = a.execLogged(r.Context(), user, clientIP(r), `UPDATE users SET username = ?, userdesc = ?, pass = ?, usertype = ? WHERE id = ?`,
			req.Username, req.UserDesc, hashedPass, req.UserType, id)
	} else {
		_, err = a.execLogged(r.Context(), user, clientIP(r), `UPDATE users SET username = ?, userdesc = ?, usertype = ? WHERE id = ?`,
			req.Username, req.UserDesc, req.UserType, id)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *App) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var targetUsername string
	if err := a.db.QueryRow(`SELECT username FROM users WHERE id = ?`, id).Scan(&targetUsername); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if strings.EqualFold(strings.TrimSpace(targetUsername), "admin") {
		writeError(w, http.StatusBadRequest, "cannot remove default admin user")
		return
	}

	adminID := int64(1)
	if err := a.db.QueryRow(`SELECT id FROM users WHERE LOWER(username) = 'admin' LIMIT 1`).Scan(&adminID); err != nil {
		if err != sql.ErrNoRows {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		adminID = 1
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE items SET userid = ? WHERE userid = ?`, adminID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`DELETE FROM users WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = a.execLogged(r.Context(), user, clientIP(r), "DELETE FROM users WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
