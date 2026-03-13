package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (a *App) handleListSoftware(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	limit := listLimitParam(r.URL.Query().Get("limit"), 50)
	offset := intParamDefault(r.URL.Query().Get("offset"), 0)

	baseQuery := `SELECT
        software.id,
        software.manufacturerid,
        software.stitle AS title,
        software.sversion AS version,
        software.slicenseinfo,
        software.sinfo,
        software.purchdate,
        COALESCE((
            SELECT MAX(contracts.currentenddate)
            FROM contract2soft
            JOIN contracts ON contracts.id = contract2soft.contractid
            WHERE contract2soft.softid = software.id
        ), 0) AS maintend,
        software.licqty,
        software.lictype,
        agents.title AS manufacturer,
        COALESCE((
            SELECT CASE software.lictype
                WHEN 1 THEN SUM(COALESCE(items.cpuno, 0))
                WHEN 2 THEN SUM(COALESCE(items.cpuno, 0) * COALESCE(items.corespercpu, 0))
                ELSE COUNT(items.id)
            END
            FROM item2soft
            JOIN items ON items.id = item2soft.itemid
            WHERE item2soft.softid = software.id
        ), 0) AS usedqty,
        COALESCE((
            SELECT GROUP_CONCAT(vendors.title, ', ')
            FROM soft2inv
            JOIN invoices ON invoices.id = soft2inv.invid
            LEFT JOIN agents AS vendors ON vendors.id = invoices.vendorid
            WHERE soft2inv.softid = software.id
        ), '') AS vendor,
        COALESCE((
            SELECT GROUP_CONCAT(CAST(invoices.id AS TEXT), ', ')
            FROM soft2inv
            JOIN invoices ON invoices.id = soft2inv.invid
            WHERE soft2inv.softid = software.id
        ), '') AS invoice,
        COALESCE((
            SELECT GROUP_CONCAT(inst.entry, ' | ')
            FROM (
                SELECT
                    '(' || items.id || ') ' ||
                    COALESCE(itemman.title, '') || ' ' ||
                    COALESCE(items.model, '') ||
                    CASE WHEN TRIM(COALESCE(items.dnsname, '')) = '' THEN '' ELSE ' ' || TRIM(items.dnsname) END AS entry
                FROM item2soft
                JOIN items ON items.id = item2soft.itemid
                LEFT JOIN agents AS itemman ON itemman.id = items.manufacturerid
                WHERE item2soft.softid = software.id
                ORDER BY items.id
            ) AS inst
        ), '') AS installedon,
        COALESCE((SELECT GROUP_CONCAT(tags.name, ', ') FROM tags JOIN tag2software ON tags.id = tag2software.tagid WHERE tag2software.softwareid = software.id), '') AS tags
    FROM software
    JOIN agents ON agents.id = software.manufacturerid`

	where := ""
	args := []interface{}{}
	if search != "" {
		where = `WHERE (
            CAST(software_view.id AS TEXT) LIKE ? OR
            software_view.manufacturer LIKE ? OR
            software_view.title LIKE ? OR
            software_view.version LIKE ? OR
            (CASE WHEN COALESCE(software_view.purchdate, 0) > 0 THEN date(software_view.purchdate, 'unixepoch') ELSE '' END) LIKE ? OR
            (CASE WHEN COALESCE(software_view.maintend, 0) > 0 THEN date(software_view.maintend, 'unixepoch') ELSE '' END) LIKE ? OR
            software_view.slicenseinfo LIKE ? OR
            software_view.sinfo LIKE ? OR
            software_view.tags LIKE ? OR
            CAST(software_view.licqty AS TEXT) LIKE ? OR
            CAST(software_view.usedqty AS TEXT) LIKE ? OR
            (CAST(software_view.usedqty AS TEXT) || '/' || CAST(software_view.licqty AS TEXT)) LIKE ? OR
            (CAST(software_view.usedqty AS TEXT) || ' / ' || CAST(software_view.licqty AS TEXT)) LIKE ? OR
            software_view.vendor LIKE ? OR
            software_view.invoice LIKE ? OR
            REPLACE(software_view.installedon, ' | ', ' ') LIKE ? OR
            software_view.installedon LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q, q, q, q, q, q, q, q, q, q, q, q, q)
	}

	query := `SELECT * FROM (` + baseQuery + `) AS software_view
    ` + where + `
    ORDER BY software_view.id DESC`
	if limit >= 0 {
		query += `
    LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleGetSoftware(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	rows, err := a.fetchRows(`SELECT * FROM software WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "software not found")
		return
	}

	payload := rows[0]
	payload["itemLinks"], _ = a.fetchIDList(`SELECT itemid FROM item2soft WHERE softid = ?`, id)
	payload["invoiceLinks"], _ = a.fetchIDList(`SELECT invid FROM soft2inv WHERE softid = ?`, id)
	payload["contractLinks"], _ = a.fetchIDList(`SELECT contractid FROM contract2soft WHERE softid = ?`, id)
	payload["fileLinks"], _ = a.fetchIDList(`SELECT fileid FROM software2file WHERE softwareid = ?`, id)
	payload["tags"], _ = a.fetchStringList(`SELECT tags.name FROM tags JOIN tag2software ON tag2software.tagid = tags.id WHERE tag2software.softwareid = ? ORDER BY tags.name`, id)

	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handleCreateSoftware(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	var req softwarePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Version) == "" || req.Manufacturer == 0 {
		writeError(w, http.StatusBadRequest, "title, version and manufacturerId are required")
		return
	}

	pd, err := parseDateInput(req.PurchaseDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid purchaseDate")
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`INSERT INTO software (invoiceid, slicenseinfo, manufacturerid, stitle, sversion, sinfo, purchdate, licqty, lictype)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nullableInt(req.InvoiceID), req.SLicenseInfo, req.Manufacturer, req.Title, req.Version, req.Info, nullableInt64Value(pd), req.LicenseQty, req.LicenseType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := result.LastInsertId()

	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2soft WHERE softid = ?`, `INSERT INTO item2soft (softid, itemid) VALUES (?, ?)`, newID, req.ItemLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM soft2inv WHERE softid = ?`, `INSERT INTO soft2inv (softid, invid) VALUES (?, ?)`, newID, req.InvoiceLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2soft WHERE softid = ?`, `INSERT INTO contract2soft (softid, contractid) VALUES (?, ?)`, newID, req.ContractLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM software2file WHERE softwareid = ?`, `INSERT INTO software2file (softwareid, fileid) VALUES (?, ?)`, newID, req.FileLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE software SET id = id WHERE id = ?", newID)
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *App) handleUpdateSoftware(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req softwarePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Version) == "" || req.Manufacturer == 0 {
		writeError(w, http.StatusBadRequest, "title, version and manufacturerId are required")
		return
	}

	pd, err := parseDateInput(req.PurchaseDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid purchaseDate")
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	oldFileIDs, err := a.fetchIDListTx(tx, `SELECT fileid FROM software2file WHERE softwareid = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = tx.Exec(`UPDATE software SET invoiceid = ?, slicenseinfo = ?, manufacturerid = ?, stitle = ?, sversion = ?, sinfo = ?, purchdate = ?, licqty = ?, lictype = ? WHERE id = ?`,
		nullableInt(req.InvoiceID), req.SLicenseInfo, req.Manufacturer, req.Title, req.Version, req.Info, nullableInt64Value(pd), req.LicenseQty, req.LicenseType, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2soft WHERE softid = ?`, `INSERT INTO item2soft (softid, itemid) VALUES (?, ?)`, id, req.ItemLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM soft2inv WHERE softid = ?`, `INSERT INTO soft2inv (softid, invid) VALUES (?, ?)`, id, req.InvoiceLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2soft WHERE softid = ?`, `INSERT INTO contract2soft (softid, contractid) VALUES (?, ?)`, id, req.ContractLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM software2file WHERE softwareid = ?`, `INSERT INTO software2file (softwareid, fileid) VALUES (?, ?)`, id, req.FileLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.cleanupRemovedFileLinksTx(tx, oldFileIDs, req.FileLinks, req.CleanupFileLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE software SET id = id WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *App) handleDeleteSoftware(w http.ResponseWriter, r *http.Request) {
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

	if _, err := tx.Exec(`DELETE FROM software2file WHERE softwareid = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if _, err := tx.Exec(`DELETE FROM software WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`DELETE FROM item2soft WHERE softid = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = a.execLogged(r.Context(), user, clientIP(r), "DELETE FROM software WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleMutateSoftwareTag(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	softwareID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid software id")
		return
	}

	var req tagMutationPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Action = strings.TrimSpace(strings.ToLower(req.Action))
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	tagID, err := a.ensureTagLogged(r.Context(), user, clientIP(r), req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	switch req.Action {
	case "add", "associate", "":
		_, err = a.execLogged(r.Context(), user, clientIP(r), `INSERT INTO tag2software (tagid, softwareid) SELECT ?, ? WHERE NOT EXISTS (SELECT 1 FROM tag2software WHERE tagid = ? AND softwareid = ?)`,
			tagID, softwareID, tagID, softwareID)
	case "remove", "delete", "deassociate":
		_, err = a.execLogged(r.Context(), user, clientIP(r), `DELETE FROM tag2software WHERE tagid = ? AND softwareid = ?`, tagID, softwareID)
	default:
		writeError(w, http.StatusBadRequest, "action must be add or remove")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleListInvoices(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	limit := listLimitParam(r.URL.Query().Get("limit"), 50)
	offset := intParamDefault(r.URL.Query().Get("offset"), 0)

	baseQuery := `SELECT
        invoices.id,
        invoices.vendorid,
        invoices.buyerid,
        invoices.number,
        invoices.date,
        invoices.description,
        vendor.title AS vendor,
        buyer.title AS buyer,
        COALESCE((
            SELECT GROUP_CONCAT(
                '(' || CAST(files.id AS TEXT) || ') ' ||
                CASE
                    WHEN TRIM(COALESCE(files.title, '')) = '' THEN COALESCE(files.fname, '')
                    WHEN TRIM(COALESCE(files.fname, '')) = '' THEN COALESCE(files.title, '')
                    ELSE COALESCE(files.title, '') || ' / ' || COALESCE(files.fname, '')
                END,
                ' | '
            )
            FROM files
            JOIN invoice2file ON files.id = invoice2file.fileid
            WHERE invoice2file.invoiceid = invoices.id
            ORDER BY files.id
        ), '') AS files
    FROM invoices
    LEFT JOIN agents AS vendor ON vendor.id = invoices.vendorid
    LEFT JOIN agents AS buyer ON buyer.id = invoices.buyerid`

	where := ""
	args := []interface{}{}
	if search != "" {
		where = `WHERE (
            CAST(invoice_view.id AS TEXT) LIKE ? OR
            invoice_view.vendor LIKE ? OR
            invoice_view.buyer LIKE ? OR
            (CASE WHEN COALESCE(invoice_view.date, 0) > 0 THEN date(invoice_view.date, 'unixepoch') ELSE '' END) LIKE ? OR
            invoice_view.number LIKE ? OR
            invoice_view.description LIKE ? OR
            invoice_view.files LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q, q, q)
	}

	query := `SELECT * FROM (` + baseQuery + `) AS invoice_view
    ` + where + `
    ORDER BY invoice_view.id DESC`
	if limit >= 0 {
		query += `
    LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleGetInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	rows, err := a.fetchRows(`SELECT * FROM invoices WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}

	payload := rows[0]
	payload["itemLinks"], _ = a.fetchIDList(`SELECT itemid FROM item2inv WHERE invid = ?`, id)
	payload["softwareLinks"], _ = a.fetchIDList(`SELECT softid FROM soft2inv WHERE invid = ?`, id)
	payload["contractLinks"], _ = a.fetchIDList(`SELECT contractid FROM contract2inv WHERE invid = ?`, id)
	payload["fileLinks"], _ = a.fetchIDList(`SELECT fileid FROM invoice2file WHERE invoiceid = ?`, id)

	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	var req invoicePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.VendorID == 0 || req.BuyerID == 0 || strings.TrimSpace(req.Number) == "" || strings.TrimSpace(req.Date) == "" {
		writeError(w, http.StatusBadRequest, "vendorId, buyerId, number and date are required")
		return
	}

	d, err := parseDateInput(req.Date)
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

	result, err := tx.Exec(`INSERT INTO invoices (vendorid, buyerid, number, description, date) VALUES (?, ?, ?, ?, ?)`,
		req.VendorID, req.BuyerID, req.Number, req.Description, d)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := result.LastInsertId()

	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2inv WHERE invid = ?`, `INSERT INTO item2inv (invid, itemid) VALUES (?, ?)`, newID, req.ItemLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM soft2inv WHERE invid = ?`, `INSERT INTO soft2inv (invid, softid) VALUES (?, ?)`, newID, req.SoftwareLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2inv WHERE invid = ?`, `INSERT INTO contract2inv (invid, contractid) VALUES (?, ?)`, newID, req.ContractLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM invoice2file WHERE invoiceid = ?`, `INSERT INTO invoice2file (invoiceid, fileid) VALUES (?, ?)`, newID, req.FileLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE invoices SET id = id WHERE id = ?", newID)
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *App) handleUpdateInvoice(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req invoicePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.VendorID == 0 || req.BuyerID == 0 || strings.TrimSpace(req.Number) == "" || strings.TrimSpace(req.Date) == "" {
		writeError(w, http.StatusBadRequest, "vendorId, buyerId, number and date are required")
		return
	}

	d, err := parseDateInput(req.Date)
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

	oldFileIDs, err := a.fetchIDListTx(tx, `SELECT fileid FROM invoice2file WHERE invoiceid = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = tx.Exec(`UPDATE invoices SET vendorid = ?, buyerid = ?, number = ?, description = ?, date = ? WHERE id = ?`,
		req.VendorID, req.BuyerID, req.Number, req.Description, d, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2inv WHERE invid = ?`, `INSERT INTO item2inv (invid, itemid) VALUES (?, ?)`, id, req.ItemLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM soft2inv WHERE invid = ?`, `INSERT INTO soft2inv (invid, softid) VALUES (?, ?)`, id, req.SoftwareLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2inv WHERE invid = ?`, `INSERT INTO contract2inv (invid, contractid) VALUES (?, ?)`, id, req.ContractLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM invoice2file WHERE invoiceid = ?`, `INSERT INTO invoice2file (invoiceid, fileid) VALUES (?, ?)`, id, req.FileLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.cleanupRemovedFileLinksTx(tx, oldFileIDs, req.FileLinks, req.CleanupFileLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE invoices SET id = id WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *App) handleDeleteInvoice(w http.ResponseWriter, r *http.Request) {
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

	if _, err := tx.Exec(`DELETE FROM invoice2file WHERE invoiceid = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if _, err := tx.Exec(`DELETE FROM invoices WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`DELETE FROM item2inv WHERE invid = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`UPDATE software SET invoiceid = '' WHERE invoiceid = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = a.execLogged(r.Context(), user, clientIP(r), "DELETE FROM invoices WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleListContracts(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	where := ""
	args := []interface{}{}
	if search != "" {
		where = `WHERE (
            CAST(contracts.id AS TEXT) LIKE ? OR
            CAST(COALESCE(contracts.parentid, '') AS TEXT) LIKE ? OR
            contracttypes.name LIKE ? OR
            contractor.title LIKE ? OR
            contracts.number LIKE ? OR
            contracts.title LIKE ? OR
            (CASE WHEN COALESCE(contracts.startdate, 0) > 0 THEN date(contracts.startdate, 'unixepoch') ELSE '' END) LIKE ? OR
            (CASE WHEN COALESCE(contracts.currentenddate, 0) > 0 THEN date(contracts.currentenddate, 'unixepoch') ELSE '' END) LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q, q, q, q)
	}

	query := `SELECT contracts.id, contracts.parentid, contracts.contractorid, contractor.title AS contractor, contracttypes.name AS type, contracts.number, contracts.title, contracts.startdate, contracts.currentenddate
        FROM contracts
        JOIN contracttypes ON contracttypes.id = contracts.type
        LEFT JOIN agents AS contractor ON contractor.id = contracts.contractorid
        ` + where + `
        ORDER BY contracts.id DESC, contracts.parentid DESC`
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleGetContract(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rows, err := a.fetchRows(`SELECT * FROM contracts WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "contract not found")
		return
	}

	payload := rows[0]
	payload["itemLinks"], _ = a.fetchIDList(`SELECT itemid FROM contract2item WHERE contractid = ?`, id)
	payload["softwareLinks"], _ = a.fetchIDList(`SELECT softid FROM contract2soft WHERE contractid = ?`, id)
	payload["invoiceLinks"], _ = a.fetchIDList(`SELECT invid FROM contract2inv WHERE contractid = ?`, id)
	payload["fileLinks"], _ = a.fetchIDList(`SELECT fileid FROM contract2file WHERE contractid = ?`, id)

	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handleCreateContract(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	var req contractPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Number) == "" || req.TypeID == 0 || req.ContractorID == 0 || strings.TrimSpace(req.StartDate) == "" || strings.TrimSpace(req.CurrentEnd) == "" {
		writeError(w, http.StatusBadRequest, "missing mandatory fields")
		return
	}

	sd, err := parseDateInput(req.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid startDate")
		return
	}
	ed, err := parseDateInput(req.CurrentEnd)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid currentEndDate")
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`INSERT INTO contracts (type, subtype, parentid, title, number, description, comments, totalcost, contractorid, startdate, currentenddate, renewals)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.TypeID, req.SubTypeID, nullableInt(req.ParentID), req.Title, req.Number, req.Description, req.Comments, req.TotalCost, req.ContractorID, sd, ed, req.Renewals)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := result.LastInsertId()

	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2item WHERE contractid = ?`, `INSERT INTO contract2item (contractid, itemid) VALUES (?, ?)`, newID, req.ItemLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2soft WHERE contractid = ?`, `INSERT INTO contract2soft (contractid, softid) VALUES (?, ?)`, newID, req.SoftwareLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2inv WHERE contractid = ?`, `INSERT INTO contract2inv (contractid, invid) VALUES (?, ?)`, newID, req.InvoiceLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2file WHERE contractid = ?`, `INSERT INTO contract2file (contractid, fileid) VALUES (?, ?)`, newID, req.FileLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE contracts SET id = id WHERE id = ?", newID)
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *App) handleUpdateContract(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req contractPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Number) == "" || req.TypeID == 0 || req.ContractorID == 0 || strings.TrimSpace(req.StartDate) == "" || strings.TrimSpace(req.CurrentEnd) == "" {
		writeError(w, http.StatusBadRequest, "missing mandatory fields")
		return
	}

	sd, err := parseDateInput(req.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid startDate")
		return
	}
	ed, err := parseDateInput(req.CurrentEnd)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid currentEndDate")
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	oldFileIDs, err := a.fetchIDListTx(tx, `SELECT fileid FROM contract2file WHERE contractid = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = tx.Exec(`UPDATE contracts SET
        type = ?, subtype = ?, title = ?, number = ?, description = ?, comments = ?, contractorid = ?,
        startdate = ?, currentenddate = ?, renewals = ?, parentid = ?, totalcost = ?
        WHERE id = ?`,
		req.TypeID, req.SubTypeID, req.Title, req.Number, req.Description, req.Comments, req.ContractorID, sd, ed, req.Renewals, nullableInt(req.ParentID), req.TotalCost, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2item WHERE contractid = ?`, `INSERT INTO contract2item (contractid, itemid) VALUES (?, ?)`, id, req.ItemLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2soft WHERE contractid = ?`, `INSERT INTO contract2soft (contractid, softid) VALUES (?, ?)`, id, req.SoftwareLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2inv WHERE contractid = ?`, `INSERT INTO contract2inv (contractid, invid) VALUES (?, ?)`, id, req.InvoiceLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2file WHERE contractid = ?`, `INSERT INTO contract2file (contractid, fileid) VALUES (?, ?)`, id, req.FileLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.cleanupRemovedFileLinksTx(tx, oldFileIDs, req.FileLinks, req.CleanupFileLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE contracts SET id = id WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *App) handleDeleteContract(w http.ResponseWriter, r *http.Request) {
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
		`DELETE FROM contract2item WHERE contractid = ?`,
		`DELETE FROM contract2soft WHERE contractid = ?`,
		`DELETE FROM contract2inv WHERE contractid = ?`,
		`DELETE FROM contract2file WHERE contractid = ?`,
		`DELETE FROM contracts WHERE id = ?`,
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

	_, _ = a.execLogged(r.Context(), user, clientIP(r), "DELETE FROM contracts WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleListContractEvents(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid contract id")
		return
	}
	rows, err := a.fetchRows(`SELECT * FROM contractevents WHERE contractid = ? ORDER BY startdate, id DESC`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleCreateContractEvent(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	contractID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid contract id")
		return
	}

	var req contractEventPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	sd, err := parseDateInput(req.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid startDate")
		return
	}
	ed, err := parseDateInput(req.EndDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid endDate")
		return
	}

	res, err := a.execLogged(r.Context(), user, clientIP(r), `INSERT INTO contractevents (contractid, siblingid, startdate, enddate, description) VALUES (?, ?, ?, ?, ?)`,
		contractID, req.SiblingID, sd, ed, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	newID, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *App) handleUpdateContractEvent(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	eventID, err := intParam(chi.URLParam(r, "eventId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	var req contractEventPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	sd, err := parseDateInput(req.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid startDate")
		return
	}
	ed, err := parseDateInput(req.EndDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid endDate")
		return
	}

	_, err = a.execLogged(r.Context(), user, clientIP(r), `UPDATE contractevents SET siblingid = ?, startdate = ?, enddate = ?, description = ? WHERE id = ?`,
		req.SiblingID, sd, ed, req.Description, eventID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": eventID})
}

func (a *App) handleDeleteContractEvent(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	eventID, err := intParam(chi.URLParam(r, "eventId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	_, err = a.execLogged(r.Context(), user, clientIP(r), `DELETE FROM contractevents WHERE id = ?`, eventID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
