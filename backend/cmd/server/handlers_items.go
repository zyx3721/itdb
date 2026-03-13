package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func (a *App) handleListItems(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	limit := listLimitParam(r.URL.Query().Get("limit"), 50)
	offset := intParamDefault(r.URL.Query().Get("offset"), 0)

	where := ""
	args := []interface{}{}
	if search != "" {
		where = `WHERE (
            CAST(items.id AS TEXT) LIKE ? OR
            items.label LIKE ? OR
            itemtypes.typedesc LIKE ? OR
            agents.title LIKE ? OR
            items.model LIKE ? OR
            items.sn LIKE ? OR
            (CASE WHEN COALESCE(items.purchasedate, 0) > 0 THEN date(items.purchasedate, 'unixepoch') ELSE '' END) LIKE ? OR
            items.ipv4 LIKE ? OR
            dpttypes.dptname LIKE ? OR
            items.principal LIKE ? OR
            statustypes.statusdesc LIKE ? OR
            (
                CASE
                    WHEN TRIM(COALESCE(locations.name,'') || COALESCE(locareas.areaname,'')) = '' THEN '-'
                    ELSE COALESCE(locations.name,'') || '-' || COALESCE(locareas.areaname,'')
                END
            ) LIKE ? OR
            (
                CASE
                    WHEN TRIM(COALESCE(racks.label,'') || COALESCE(items.switchport,'')) = '' THEN '-'
                    ELSE TRIM(COALESCE(racks.label,'') || ' ' || COALESCE(items.switchport,''))
                END
            ) LIKE ? OR
            items.function LIKE ? OR
            COALESCE((
                SELECT GROUP_CONCAT(COALESCE(swman.title, '') || ' ' || COALESCE(sw.stitle, '') || COALESCE(sw.sversion, ''), ', ')
                FROM software AS sw
                JOIN item2soft AS i2s ON sw.id = i2s.softid
                LEFT JOIN agents AS swman ON swman.id = sw.manufacturerid
                WHERE i2s.itemid = items.id
            ), '') LIKE ? OR
            items.remadmip LIKE ? OR
            COALESCE((SELECT GROUP_CONCAT(tags.name, ', ') FROM tags JOIN tag2item ON tags.id = tag2item.tagid WHERE tag2item.itemid = items.id), '') LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args,
			q, q, q, q, q, q, q, q, q,
			q, q, q, q, q, q, q, q,
		)
	}

	query := `
SELECT
    items.id,
    items.label,
    items.itemtypeid,
    items.manufacturerid,
    items.locationid,
    items.locareaid,
    items.rackid,
    items.userid,
    itemtypes.typedesc AS itemType,
    agents.title AS manufacturer,
    items.model,
    items.sn,
    items.sn2,
    items.sn3,
    items.purchasedate,
    items.warrantymonths,
    items.ipv4,
    items.dnsname,
    items.principal,
    users.username,
    statustypes.statusdesc AS status,
    statustypes.color AS statuscolor,
    dpttypes.dptname AS dpt,
    CASE
        WHEN TRIM(COALESCE(locations.name,'') || COALESCE(locareas.areaname,'')) = '' THEN '-'
        ELSE COALESCE(locations.name,'') || '-' || COALESCE(locareas.areaname,'')
    END AS location,
    COALESCE(locareas.areaname,'') AS area,
    CASE
        WHEN TRIM(COALESCE(racks.label,'') || COALESCE(items.switchport,'')) = '' THEN '-'
        ELSE TRIM(COALESCE(racks.label,'') || ' ' || COALESCE(items.switchport,''))
    END AS rack,
    items.function,
    items.remadmip,
    items.usize,
    items.rackposition,
    items.rackposdepth,
    COALESCE((SELECT GROUP_CONCAT(tags.name, ', ') FROM tags JOIN tag2item ON tags.id = tag2item.tagid WHERE tag2item.itemid = items.id), '') AS tags,
    COALESCE((
        SELECT GROUP_CONCAT(COALESCE(swman.title, '') || ' ' || COALESCE(sw.stitle, '') || COALESCE(sw.sversion, ''), ', ')
        FROM software AS sw
        JOIN item2soft AS i2s ON sw.id = i2s.softid
        LEFT JOIN agents AS swman ON swman.id = sw.manufacturerid
        WHERE i2s.itemid = items.id
    ), '') AS software
FROM items
JOIN itemtypes ON itemtypes.id = items.itemtypeid
JOIN agents ON agents.id = items.manufacturerid
LEFT JOIN statustypes ON statustypes.id = items.status
LEFT JOIN dpttypes ON dpttypes.id = items.dptid
LEFT JOIN users ON users.id = items.userid
LEFT JOIN locations ON locations.id = items.locationid
LEFT JOIN locareas ON locareas.id = items.locareaid
LEFT JOIN racks ON racks.id = items.rackid
` + where + `
ORDER BY items.id DESC`
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

func (a *App) handleGetItem(w http.ResponseWriter, r *http.Request) {
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	rows, err := a.fetchRows(`SELECT * FROM items WHERE id = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}

	payload := rows[0]
	forwardLinks, _ := a.fetchIDList(`SELECT itemid2 FROM itemlink WHERE itemid1 = ?`, id)
	reverseLinks, _ := a.fetchIDList(`SELECT itemid1 FROM itemlink WHERE itemid2 = ?`, id)
	linkSet := make(map[int64]struct{})
	for _, linkedID := range forwardLinks {
		if linkedID > 0 {
			linkSet[linkedID] = struct{}{}
		}
	}
	for _, linkedID := range reverseLinks {
		if linkedID > 0 {
			linkSet[linkedID] = struct{}{}
		}
	}
	itemLinks := make([]int64, 0, len(linkSet))
	for linkedID := range linkSet {
		itemLinks = append(itemLinks, linkedID)
	}
	sort.Slice(itemLinks, func(i, j int) bool { return itemLinks[i] < itemLinks[j] })
	payload["itemLinks"] = itemLinks
	payload["invoiceLinks"], _ = a.fetchIDList(`SELECT invid FROM item2inv WHERE itemid = ?`, id)
	payload["softwareLinks"], _ = a.fetchIDList(`SELECT softid FROM item2soft WHERE itemid = ?`, id)
	payload["contractLinks"], _ = a.fetchIDList(`SELECT contractid FROM contract2item WHERE itemid = ?`, id)
	payload["fileLinks"], _ = a.fetchIDList(`SELECT fileid FROM item2file WHERE itemid = ?`, id)
	payload["tags"], _ = a.fetchStringList(`SELECT tags.name FROM tags JOIN tag2item ON tag2item.tagid = tags.id WHERE tag2item.itemid = ? ORDER BY tags.name`, id)
	actions, _ := a.fetchRows(`SELECT * FROM actions WHERE itemid = ? ORDER BY actiondate`, id)
	payload["actions"] = actions

	writeJSON(w, http.StatusOK, payload)
}

func (a *App) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())

	var req itemPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := validateItem(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	purchDate, err := parseDateInput(req.PurchaseDate)
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

	result, err := tx.Exec(`INSERT INTO items (
        label, itemtypeid, function, manufacturerid,
        warrinfo, model, sn, sn2, sn3, origin,
        warrantymonths, purchasedate, purchprice, dnsname,
        dptid, principal, locationid, locareaid, userid,
        maintenanceinfo, comments, ispart, rackid, rackposition,
        rackposdepth, rackmountable, usize, status,
        macs, ipv4, ipv6, remadmip, hd, cpu, cpuno,
        corespercpu, ram, raid, raidconfig, panelport,
        switchid, switchport, ports
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Label,
		req.ItemTypeID,
		req.Function,
		req.ManufacturerID,
		req.WarrInfo,
		req.Model,
		req.SN,
		req.SN2,
		req.SN3,
		req.Origin,
		nullableInt(req.WarrantyMonths),
		nullableInt64Value(purchDate),
		req.PurchPrice,
		req.DNSName,
		nullableInt(req.DptID),
		req.Principal,
		nullableInt(req.LocationID),
		nullableInt(req.LocAreaID),
		nullableInt(req.UserID),
		req.MaintenanceInfo,
		req.Comments,
		req.IsPart,
		nullableInt(req.RackID),
		nullableInt(req.RackPosition),
		nullableInt(req.RackPosDepth),
		req.RackMountable,
		nullableInt(req.USize),
		req.Status,
		req.MACs,
		req.IPv4,
		req.IPv6,
		req.RemAdmIP,
		req.HD,
		req.CPU,
		req.CPUNo,
		req.CoresPerCPU,
		req.RAM,
		req.Raid,
		req.RaidConfig,
		req.PanelPort,
		nullableInt(req.SwitchID),
		req.SwitchPort,
		req.Ports,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	newID, err := result.LastInsertId()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := a.replaceUndirectedItemLinksTx(tx, newID, req.ItemLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2inv WHERE itemid = ?`, `INSERT INTO item2inv (itemid, invid) VALUES (?, ?)`, newID, req.InvoiceLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2soft WHERE itemid = ?`, `INSERT INTO item2soft (itemid, softid) VALUES (?, ?)`, newID, req.SoftwareLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2item WHERE itemid = ?`, `INSERT INTO contract2item (itemid, contractid) VALUES (?, ?)`, newID, req.ContractLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2file WHERE itemid = ?`, `INSERT INTO item2file (itemid, fileid) VALUES (?, ?)`, newID, req.FileLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	now := time.Now().Unix()
	_, err = tx.Exec(`INSERT INTO actions (itemid, actiondate, description, invoiceinfo, isauto, entrydate) VALUES (?, ?, ?, '', 1, ?)`,
		newID,
		now,
		fmt.Sprintf("Added by %s", user.Username),
		now,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE items SET id = id WHERE id = ?", newID)
	writeJSON(w, http.StatusCreated, map[string]int64{"id": newID})
}

func (a *App) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req itemPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validateItem(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	purchDate, err := parseDateInput(req.PurchaseDate)
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

	var currentUserID sql.NullInt64
	_ = tx.QueryRow(`SELECT userid FROM items WHERE id = ?`, id).Scan(&currentUserID)
	oldFileIDs, err := a.fetchIDListTx(tx, `SELECT fileid FROM item2file WHERE itemid = ?`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = tx.Exec(`UPDATE items SET
        itemtypeid = ?, function = ?, manufacturerid = ?, label = ?,
        warrinfo = ?, model = ?, sn = ?, sn2 = ?, sn3 = ?,
        locationid = ?, locareaid = ?, origin = ?, warrantymonths = ?,
        purchasedate = ?, purchprice = ?, dnsname = ?, userid = ?, dptid = ?, principal = ?,
        comments = ?, maintenanceinfo = ?, ispart = ?, hd = ?, cpu = ?, cpuno = ?, corespercpu = ?,
        ram = ?, raid = ?, raidconfig = ?, rackmountable = ?, rackid = ?, rackposition = ?,
        rackposdepth = ?, usize = ?, status = ?, macs = ?, ipv4 = ?, ipv6 = ?, remadmip = ?,
        panelport = ?, switchid = ?, switchport = ?, ports = ?
        WHERE id = ?`,
		req.ItemTypeID, req.Function, req.ManufacturerID, req.Label,
		req.WarrInfo, req.Model, req.SN, req.SN2, req.SN3,
		nullableInt(req.LocationID), nullableInt(req.LocAreaID), req.Origin, nullableInt(req.WarrantyMonths),
		nullableInt64Value(purchDate), req.PurchPrice, req.DNSName, nullableInt(req.UserID), nullableInt(req.DptID), req.Principal,
		req.Comments, req.MaintenanceInfo, req.IsPart, req.HD, req.CPU, req.CPUNo, req.CoresPerCPU,
		req.RAM, req.Raid, req.RaidConfig, req.RackMountable, nullableInt(req.RackID), nullableInt(req.RackPosition),
		nullableInt(req.RackPosDepth), nullableInt(req.USize), req.Status, req.MACs, req.IPv4, req.IPv6, req.RemAdmIP,
		req.PanelPort, nullableInt(req.SwitchID), req.SwitchPort, req.Ports,
		id,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !equalNullInt64(currentUserID, req.UserID) {
		_, _ = tx.Exec(`INSERT INTO actions (itemid, actiondate, description, invoiceinfo, isauto, entrydate)
            VALUES (?, ?, ?, '', 1, ?)`,
			id,
			time.Now().Unix(),
			fmt.Sprintf("Updated user by %s", user.Username),
			time.Now().Unix(),
		)
	}

	var lastDesc string
	var lastEntry sql.NullInt64
	_ = tx.QueryRow(`SELECT description, entrydate FROM actions WHERE itemid = ? ORDER BY entrydate DESC LIMIT 1`, id).Scan(&lastDesc, &lastEntry)
	updateDesc := fmt.Sprintf("Updated by %s", user.Username)
	if lastDesc != updateDesc || !sameDay(lastEntry.Int64, time.Now().Unix()) {
		_, _ = tx.Exec(`INSERT INTO actions (itemid, actiondate, description, invoiceinfo, isauto, entrydate)
            VALUES (?, ?, ?, '', 1, ?)`, id, time.Now().Unix(), updateDesc, time.Now().Unix())
	}

	if err := a.replaceUndirectedItemLinksTx(tx, id, req.ItemLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2inv WHERE itemid = ?`, `INSERT INTO item2inv (itemid, invid) VALUES (?, ?)`, id, req.InvoiceLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2soft WHERE itemid = ?`, `INSERT INTO item2soft (itemid, softid) VALUES (?, ?)`, id, req.SoftwareLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM contract2item WHERE itemid = ?`, `INSERT INTO contract2item (itemid, contractid) VALUES (?, ?)`, id, req.ContractLinks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.replaceIDLinksTx(tx, `DELETE FROM item2file WHERE itemid = ?`, `INSERT INTO item2file (itemid, fileid) VALUES (?, ?)`, id, req.FileLinks); err != nil {
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

	_, _ = a.execLogged(r.Context(), user, clientIP(r), "UPDATE items SET id = id WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (a *App) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
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

	if _, err := tx.Exec(`DELETE FROM item2file WHERE itemid = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	statements := []struct {
		q    string
		args []interface{}
	}{
		{`DELETE FROM item2inv WHERE itemid = ?`, []interface{}{id}},
		{`DELETE FROM item2soft WHERE itemid = ?`, []interface{}{id}},
		{`DELETE FROM itemlink WHERE itemid1 = ? OR itemid2 = ?`, []interface{}{id, id}},
		{`UPDATE tag2item SET itemid = NULL WHERE itemid = ?`, []interface{}{id}},
		{`DELETE FROM items WHERE id = ?`, []interface{}{id}},
	}

	for _, st := range statements {
		if _, err := tx.Exec(st.q, st.args...); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, _ = a.execLogged(r.Context(), user, clientIP(r), "DELETE FROM items WHERE id = ?", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleMutateItemTag(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	itemID, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
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
		_, err = a.execLogged(r.Context(), user, clientIP(r), `INSERT INTO tag2item (tagid, itemid) SELECT ?, ? WHERE NOT EXISTS (SELECT 1 FROM tag2item WHERE tagid = ? AND itemid = ?)`,
			tagID, itemID, tagID, itemID)
	case "remove", "delete", "deassociate":
		_, err = a.execLogged(r.Context(), user, clientIP(r), `DELETE FROM tag2item WHERE tagid = ? AND itemid = ?`, tagID, itemID)
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
