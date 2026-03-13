package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

const viewHistoryKeepLimit = 40

type viewHistoryRequest struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

func (a *App) handleListViewHistory(w http.ResponseWriter, r *http.Request) {
	rows, err := a.fetchRows(`SELECT id, url, description FROM viewhist ORDER BY id DESC LIMIT ?`, viewHistoryKeepLimit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleCreateViewHistory(w http.ResponseWriter, r *http.Request) {
	var req viewHistoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.URL = strings.TrimSpace(req.URL)
	req.Description = strings.TrimSpace(req.Description)
	if req.URL == "" || req.Description == "" {
		writeError(w, http.StatusBadRequest, "url and description are required")
		return
	}

	var lastURL string
	err := a.db.QueryRow(`SELECT url FROM viewhist ORDER BY id DESC LIMIT 1`).Scan(&lastURL)
	if err != nil && err != sql.ErrNoRows {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if strings.TrimSpace(lastURL) != req.URL {
		result, err := a.db.Exec(`INSERT INTO viewhist (url, description) VALUES (?, ?)`, req.URL, req.Description)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		lastID, _ := result.LastInsertId()
		lastKeep := lastID - viewHistoryKeepLimit
		if lastKeep > 0 {
			if _, err := a.db.Exec(`DELETE FROM viewhist WHERE id < ?`, lastKeep); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
