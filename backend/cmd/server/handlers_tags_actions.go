package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func (a *App) handleListTags(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	where := ""
	args := []interface{}{}
	if search != "" {
		where = " WHERE tags.name LIKE ? "
		args = append(args, "%"+search+"%")
	}
	query := `SELECT tags.id, tags.name,
        (SELECT COUNT(*) FROM tag2item WHERE tagid = tags.id) AS itemCount,
        (SELECT COUNT(*) FROM tag2software WHERE tagid = tags.id) AS softwareCount
        FROM tags ` + where + ` ORDER BY tags.name`
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleSuggestTags(w http.ResponseWriter, r *http.Request) {
	term := strings.TrimSpace(r.URL.Query().Get("term"))
	rows, err := a.fetchRows(`SELECT name FROM tags WHERE name LIKE ? ORDER BY name LIMIT 20`, "%"+term+"%")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, asString(row["name"]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(asString(body["name"]))
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	tagID, err := a.ensureTagLogged(r.Context(), user, clientIP(r), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": tagID})
}

func (a *App) handleUpdateTag(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(asString(body["name"]))
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	var dup int64
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM tags WHERE LOWER(name) = LOWER(?) AND id <> ?`, name, id).Scan(&dup); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if dup > 0 {
		writeError(w, http.StatusConflict, "tag already exists")
		return
	}

	if _, err := a.execLogged(r.Context(), user, clientIP(r), `UPDATE tags SET name = ? WHERE id = ?`, name, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *App) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var itemCount, softwareCount int64
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM tag2item WHERE tagid = ?`, id).Scan(&itemCount); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM tag2software WHERE tagid = ?`, id).Scan(&softwareCount); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if itemCount > 0 || softwareCount > 0 {
		writeError(w, http.StatusConflict, fmt.Sprintf("tag has associations (items=%d software=%d)", itemCount, softwareCount))
		return
	}

	if _, err := a.execLogged(r.Context(), user, clientIP(r), `DELETE FROM tags WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleListTagItems(w http.ResponseWriter, r *http.Request) {
	tagID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tag id")
		return
	}
	rows, err := a.fetchRows(`SELECT items.id, agents.title || ' ' || items.model || ' [' || itemtypes.typedesc || ', ID:' || items.id || ']' AS txt
        FROM agents, items, itemtypes
        WHERE agents.id = items.manufacturerid
          AND items.itemtypeid = itemtypes.id
          AND items.id IN (SELECT itemid FROM tag2item WHERE tagid = ?)`, tagID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleListTagSoftware(w http.ResponseWriter, r *http.Request) {
	tagID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tag id")
		return
	}
	rows, err := a.fetchRows(`SELECT software.id, agents.title || ' ' || software.stitle || ' ' || software.sversion || ' [ID:' || software.id || ']' AS txt
        FROM agents, software
        WHERE agents.id = software.manufacturerid
          AND software.id IN (SELECT softwareid FROM tag2software WHERE tagid = ?)`, tagID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleListItemActions(w http.ResponseWriter, r *http.Request) {
	itemID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	rows, err := a.fetchRows(`SELECT * FROM actions WHERE itemid = ? ORDER BY actiondate, id`, itemID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleCreateItemAction(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	itemID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	var req itemActionPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Description = strings.TrimSpace(req.Description)
	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}

	actionDate := time.Now().Unix()
	if strings.TrimSpace(req.ActionDate) != "" {
		actionDate, err = parseDateInput(req.ActionDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid actionDate")
			return
		}
	}

	res, err := a.execLogged(r.Context(), user, clientIP(r), `INSERT INTO actions (itemid, actiondate, description, invoiceinfo, isauto, entrydate) VALUES (?, ?, ?, ?, 0, ?)`,
		itemID, actionDate, req.Description, req.InvoiceInfo, time.Now().Unix())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *App) handleUpdateItemAction(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	itemID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	actionID, err := intParam(chi.URLParam(r, "actionId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid action id")
		return
	}

	var req itemActionPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Description = strings.TrimSpace(req.Description)
	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}

	actionDate := time.Now().Unix()
	if strings.TrimSpace(req.ActionDate) != "" {
		actionDate, err = parseDateInput(req.ActionDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid actionDate")
			return
		}
	}

	res, err := a.execLogged(r.Context(), user, clientIP(r), `UPDATE actions SET actiondate = ?, description = ?, invoiceinfo = ?, isauto = 0 WHERE id = ? AND itemid = ?`,
		actionDate, req.Description, req.InvoiceInfo, actionID, itemID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "action not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": actionID})
}

func (a *App) handleDeleteItemAction(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	itemID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	actionID, err := intParam(chi.URLParam(r, "actionId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid action id")
		return
	}

	res, err := a.execLogged(r.Context(), user, clientIP(r), `DELETE FROM actions WHERE id = ? AND itemid = ?`, actionID, itemID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "action not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
