package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (a *App) handleListLocations(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	baseQuery := `SELECT
        locations.id,
        locations.name,
        locations.floor,
        locations.floorplanfn,
        COALESCE((
          SELECT GROUP_CONCAT(area_rows.areaname, ', ')
          FROM (
            SELECT TRIM(COALESCE(locareas.areaname, '')) AS areaname
            FROM locareas
            WHERE locareas.locationid = locations.id
            ORDER BY locareas.id
          ) AS area_rows
          WHERE area_rows.areaname <> ''
        ), '') AS areaname
      FROM locations`

	where := ""
	args := []interface{}{}
	if search != "" {
		where = `WHERE (
            CAST(location_view.id AS TEXT) LIKE ? OR
            location_view.name LIKE ? OR
            location_view.floor LIKE ? OR
            location_view.areaname LIKE ? OR
            location_view.floorplanfn LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q)
	}

	query := `SELECT * FROM (` + baseQuery + `) AS location_view
        ` + where + `
        ORDER BY location_view.id DESC`
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleGetLocation(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rows, err := a.fetchRows(`SELECT * FROM locations WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	payload := rows[0]
	payload["areas"], _ = a.fetchRows(`SELECT * FROM locareas WHERE locationid = ? ORDER BY areaname`, id)
	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handleDownloadLocationFloorplan(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var floorplan, locationName sql.NullString
	if err := a.db.QueryRow(`SELECT floorplanfn, name FROM locations WHERE id = ?`, id).Scan(&floorplan, &locationName); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "floorplan not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fileName := strings.TrimSpace(floorplan.String)
	if fileName == "" {
		writeError(w, http.StatusNotFound, "floorplan not found")
		return
	}
	if fileName != filepath.Base(fileName) {
		writeError(w, http.StatusBadRequest, "invalid floorplan name")
		return
	}
	path := filepath.Join(a.cfg.UploadDir, fileName)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "floorplan not found")
		return
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	displayName := fileName
	if name := strings.TrimSpace(locationName.String); name != "" {
		displayName = fmt.Sprintf("%s%s", name, ext)
	}
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	setContentDispositionHeader(w, "inline", displayName)
	http.ServeFile(w, r, path)
}

func (a *App) handleCreateLocation(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/") {
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "invalid multipart form")
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		floor := strings.TrimSpace(r.FormValue("floor"))
		if name == "" || floor == "" {
			writeError(w, http.StatusBadRequest, "name and floor are required")
			return
		}
		var floorplan string
		fileHeader, _ := getMultipartFileHeader(r.MultipartForm, "file")
		if fileHeader != nil {
			saved, err := a.storeFloorplanFile(fileHeader, name)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			floorplan = saved
		}
		res, err := a.execLogged(r.Context(), user, clientIP(r), `INSERT INTO locations (name, floor, floorplanfn) VALUES (?, ?, ?)`, name, floor, floorplan)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		newID, _ := res.LastInsertId()
		writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
		return
	}

	var req locationPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Floor) == "" {
		writeError(w, http.StatusBadRequest, "name and floor are required")
		return
	}
	res, err := a.execLogged(r.Context(), user, clientIP(r), `INSERT INTO locations (name, floor) VALUES (?, ?)`, req.Name, req.Floor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *App) handleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/") {
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "invalid multipart form")
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		floor := strings.TrimSpace(r.FormValue("floor"))
		if name == "" || floor == "" {
			writeError(w, http.StatusBadRequest, "name and floor are required")
			return
		}

		tx, err := a.db.BeginTx(r.Context(), nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`UPDATE locations SET name = ?, floor = ? WHERE id = ?`, name, floor, id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		fileHeader, _ := getMultipartFileHeader(r.MultipartForm, "file")
		if fileHeader != nil {
			var oldName string
			_ = tx.QueryRow(`SELECT floorplanfn FROM locations WHERE id = ?`, id).Scan(&oldName)
			saved, err := a.storeFloorplanFile(fileHeader, name)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if _, err := tx.Exec(`UPDATE locations SET floorplanfn = ? WHERE id = ?`, saved, id); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if strings.TrimSpace(oldName) != "" {
				_ = os.Remove(filepath.Join(a.cfg.UploadDir, oldName))
			}
		}

		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE locations SET id = id WHERE id = ?", id)
		writeJSON(w, http.StatusOK, map[string]int64{"id": id})
		return
	}

	var req locationPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Floor) == "" {
		writeError(w, http.StatusBadRequest, "name and floor are required")
		return
	}
	_, err = a.execLogged(r.Context(), user, clientIP(r), `UPDATE locations SET name = ?, floor = ? WHERE id = ?`, req.Name, req.Floor, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *App) handleDeleteLocation(w http.ResponseWriter, r *http.Request) {
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

	var floorplan sql.NullString
	_ = tx.QueryRow(`SELECT floorplanfn FROM locations WHERE id = ?`, id).Scan(&floorplan)
	if _, err := tx.Exec(`DELETE FROM locations WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`UPDATE items SET locationid = 0 WHERE locationid = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if strings.TrimSpace(floorplan.String) != "" {
		_ = os.Remove(filepath.Join(a.cfg.UploadDir, floorplan.String))
	}
	_, _ = a.execLogged(r.Context(), user, clientIP(r), "DELETE FROM locations WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleListLocAreas(w http.ResponseWriter, r *http.Request) {
	locationID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location id")
		return
	}
	rows, err := a.fetchRows(`SELECT * FROM locareas WHERE locationid = ? ORDER BY areaname`, locationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleCreateLocArea(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	locationID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location id")
		return
	}
	var req locAreaPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.AreaName) == "" {
		writeError(w, http.StatusBadRequest, "areaName is required")
		return
	}
	res, err := a.execLogged(r.Context(), user, clientIP(r), `INSERT INTO locareas (locationid, areaname) VALUES (?, ?)`, locationID, req.AreaName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *App) handleUpdateLocArea(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	locationID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location id")
		return
	}
	areaID, err := intParam(chi.URLParam(r, "areaId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid area id")
		return
	}
	var req locAreaPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.AreaName) == "" {
		writeError(w, http.StatusBadRequest, "areaName is required")
		return
	}
	_, err = a.execLogged(r.Context(), user, clientIP(r), `UPDATE locareas SET locationid = ?, areaname = ? WHERE id = ?`, locationID, req.AreaName, areaID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": areaID})
}

func (a *App) handleDeleteLocArea(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	areaID, err := intParam(chi.URLParam(r, "areaId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid area id")
		return
	}
	var links int64
	err = a.db.QueryRow(`
        SELECT
          (SELECT COUNT(*) FROM items WHERE locareaid = ?) +
          (SELECT COUNT(*) FROM racks WHERE locareaid = ?)
    `, areaID, areaID).Scan(&links)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if links > 0 {
		writeError(w, http.StatusConflict, fmt.Sprintf("area has %d associations", links))
		return
	}
	_, err = a.execLogged(r.Context(), user, clientIP(r), `DELETE FROM locareas WHERE id = ?`, areaID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleListRacks(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	baseQuery := `SELECT
        racks.id,
        racks.label,
        racks.usize,
        racks.depth,
        racks.locationid,
        racks.locareaid,
        racks.model,
        racks.comments,
        racks.revnums,
        locations.name AS location,
        locareas.areaname AS area,
        COUNT(items.id) AS population,
        COALESCE(SUM(items.usize), 0) AS occupation
        FROM racks
    LEFT JOIN items ON items.rackid = racks.id
    LEFT JOIN locations ON locations.id = racks.locationid
    LEFT JOIN locareas ON locareas.id = racks.locareaid
    GROUP BY racks.id`

	where := ""
	args := []interface{}{}
	if search != "" {
		where = `WHERE (
            CAST(rack_view.id AS TEXT) LIKE ? OR
            CAST(rack_view.occupation AS TEXT) LIKE ? OR
            CAST(rack_view.population AS TEXT) LIKE ? OR
            CAST(rack_view.usize AS TEXT) LIKE ? OR
            CAST(rack_view.depth AS TEXT) LIKE ? OR
            rack_view.location LIKE ? OR
            rack_view.area LIKE ? OR
            rack_view.label LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q, q, q, q)
	}

	query := `SELECT * FROM (` + baseQuery + `) AS rack_view
    ` + where + `
    ORDER BY rack_view.id DESC`
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleGetRack(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rows, err := a.fetchRows(`SELECT * FROM racks WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "rack not found")
		return
	}
	writeJSON(w, http.StatusOK, rows[0])
}

func (a *App) handleCreateRack(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	var req rackPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.USize == 0 || req.Depth == 0 {
		writeError(w, http.StatusBadRequest, "uSize and depth are required")
		return
	}
	res, err := a.execLogged(r.Context(), user, clientIP(r), `INSERT INTO racks (locationid, usize, depth, comments, model, label, revnums, locareaid) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.LocationID, req.USize, req.Depth, req.Comments, req.Model, req.Label, req.RevNums, nullableInt(req.LocAreaID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func validateRackItemBounds(ctx context.Context, tx *sql.Tx, rackID, totalUnits, revNums int64) (string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, COALESCE(model, ''), COALESCE(rackposition, 0), COALESCE(usize, 0) FROM items WHERE rackid = ? ORDER BY id`, rackID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	reverse := revNums == 1
	for rows.Next() {
		var itemID int64
		var model string
		var rackPosition int64
		var units int64
		if err := rows.Scan(&itemID, &model, &rackPosition, &units); err != nil {
			return "", err
		}
		if rackPosition <= 0 || units <= 0 {
			continue
		}

		unitsToCheck := make([]int64, 0, units)
		for offset := int64(0); offset < units; offset++ {
			unit := rackPosition - offset
			if reverse {
				unit = rackPosition + offset
			}
			unitsToCheck = append(unitsToCheck, unit)
		}
		for _, unit := range unitsToCheck {
			if unit >= 1 && unit <= totalUnits {
				continue
			}
			if strings.TrimSpace(model) == "" {
				model = "-"
			}
			return fmt.Sprintf("硬件 %d（%s）超出机架边界", itemID, model), nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return "", nil
}

func (a *App) handleUpdateRack(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req rackPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.USize == 0 || req.Depth == 0 {
		writeError(w, http.StatusBadRequest, "uSize and depth are required")
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()
	validationError, err := validateRackItemBounds(r.Context(), tx, id, req.USize, req.RevNums)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if validationError != "" {
		writeError(w, http.StatusBadRequest, validationError)
		return
	}
	if _, err := tx.Exec(`UPDATE racks SET
        locationid = ?,
        locareaid = ?,
        usize = ?,
        revnums = ?,
        depth = ?,
        model = ?,
        comments = ?,
        label = ?
        WHERE id = ?`, req.LocationID, nullableInt(req.LocAreaID), req.USize, req.RevNums, req.Depth, req.Model, req.Comments, req.Label, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`UPDATE items SET locationid = ?, locareaid = ? WHERE rackid = ?`, req.LocationID, nullableInt(req.LocAreaID), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE racks SET id = id WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *App) handleDeleteRack(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var nitems int64
	if err := a.db.QueryRow(`SELECT COUNT(id) FROM items WHERE rackid = ?`, id).Scan(&nitems); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if nitems > 0 {
		writeError(w, http.StatusConflict, "rack has associated items")
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE items SET rackid = '' WHERE rackid = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`DELETE FROM racks WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = a.execLogged(r.Context(), user, clientIP(r), "DELETE FROM racks WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleListDictionaries(w http.ResponseWriter, r *http.Request) {
	payload := map[string]interface{}{}
	tables := map[string]string{
		"itemtypes":        `SELECT id, typedesc, hassoftware FROM itemtypes ORDER BY id ASC`,
		"filetypes":        `SELECT id, typedesc FROM filetypes ORDER BY id`,
		"statustypes":      `SELECT id, statusdesc, color FROM statustypes ORDER BY id`,
		"dpttypes":         `SELECT id, dptname FROM dpttypes ORDER BY id ASC`,
		"contracttypes":    `SELECT id, name FROM contracttypes ORDER BY id`,
		"contractsubtypes": `SELECT id, contypeid, name FROM contractsubtypes ORDER BY id ASC`,
		"tags": `SELECT
            tags.id,
            tags.name,
            (SELECT COUNT(*) FROM tag2item WHERE tagid = tags.id) AS itemCount,
            (SELECT COUNT(*) FROM tag2software WHERE tagid = tags.id) AS softwareCount
        FROM tags ORDER BY id ASC`,
	}
	for name, query := range tables {
		rows, err := a.fetchRows(query)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		payload[name] = rows
	}
	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handleCreateDictionaryRow(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	name := chi.URLParam(r, "name")
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	query, args, err := dictionaryInsert(name, body)
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

	if err := enforceDictionaryUniqueText(tx, name, body, 0); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	res, err := a.execTxLogged(r.Context(), tx, user, clientIP(r), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (a *App) handleUpdateDictionaryRow(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	name := chi.URLParam(r, "name")
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
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	if err := enforceDictionaryUpdateRules(tx, name, id); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err := enforceDictionaryUniqueText(tx, name, body, id); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	query, args, err := dictionaryUpdate(name, id, body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := a.execTxLogged(r.Context(), tx, user, clientIP(r), query, args...); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *App) handleDeleteDictionaryRow(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	name := chi.URLParam(r, "name")
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

	if err := enforceDictionaryDeleteRules(tx, name, id); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	query, err := dictionaryDelete(name)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := tx.Exec(query, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = a.execLogged(r.Context(), user, clientIP(r), query, id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
