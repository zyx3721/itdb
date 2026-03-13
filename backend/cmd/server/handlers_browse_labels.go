package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type browseNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Leaf     bool   `json:"leaf"`
	Resource string `json:"resource,omitempty"`
	EntityID int64  `json:"entityId,omitempty"`
}

func toBrowseNodes(rows []map[string]interface{}, idPrefix string, leaf bool, resource string) []browseNode {
	nodes := make([]browseNode, 0, len(rows))
	for _, row := range rows {
		id := asInt64(row["id"])
		label := strings.TrimSpace(asString(row["nodetext"]))
		if label == "" {
			label = strings.TrimSpace(asString(row["name"]))
		}
		if label == "" {
			label = strings.TrimSpace(asString(row["typedesc"]))
		}
		if label == "" {
			label = strings.TrimSpace(asString(row["username"]))
		}
		nodes = append(nodes, browseNode{
			ID:       idPrefix + strconv.FormatInt(id, 10),
			Label:    label,
			Leaf:     leaf,
			Resource: resource,
			EntityID: id,
		})
	}
	return nodes
}

func (a *App) handleBrowseTree(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" || id == "0" {
		writeJSON(w, http.StatusOK, []browseNode{
			{ID: "itemtypes", Label: "硬件类型", Leaf: false},
			{ID: "showusers", Label: "用户", Leaf: false},
			{ID: "showagents", Label: "代理", Leaf: false},
		})
		return
	}

	switch id {
	case "showusers":
		rows, err := a.fetchRows(`SELECT id, username AS nodetext FROM users ORDER BY username`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "users:", false, ""))
		return
	case "itemtypes":
		rows, err := a.fetchRows(`SELECT id, typedesc AS nodetext FROM itemtypes ORDER BY typedesc`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "itemtypes:", false, ""))
		return
	case "showagents":
		writeJSON(w, http.StatusOK, []browseNode{
			{ID: "agents:items", Label: "硬件厂商", Leaf: false},
			{ID: "agents:software", Label: "软件厂商", Leaf: false},
			{ID: "agents:vendors", Label: "供应商", Leaf: false},
			{ID: "agents:buyers", Label: "采购方", Leaf: false},
			{ID: "agents:contractors", Label: "承包方", Leaf: false},
		})
		return
	case "agents:items":
		rows, err := a.fetchRows(`SELECT id, title AS nodetext FROM agents WHERE type & 8 ORDER BY nodetext`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "agenthw:", false, ""))
		return
	case "agents:software":
		rows, err := a.fetchRows(`SELECT id, title AS nodetext FROM agents WHERE type & 2 ORDER BY nodetext`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "agentsw:", false, ""))
		return
	case "agents:vendors":
		rows, err := a.fetchRows(`SELECT id, title AS nodetext FROM agents WHERE type & 4 ORDER BY nodetext`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "agentvendor:", false, ""))
		return
	case "agents:buyers":
		rows, err := a.fetchRows(`SELECT id, title AS nodetext FROM agents WHERE type & 1 ORDER BY nodetext`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "agentbuyer:", false, ""))
		return
	case "agents:contractors":
		rows, err := a.fetchRows(`SELECT id, title AS nodetext FROM agents WHERE type & 16 ORDER BY nodetext`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "agentcontractor:", false, ""))
		return
	}

	if strings.HasPrefix(id, "users:") {
		userID, err := intParam(strings.TrimPrefix(id, "users:"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user node id")
			return
		}
		rows, err := a.fetchRows(`SELECT items.id, agents.title || ' ' || items.model || ' [' || itemtypes.typedesc || ', ID:' || items.id || ']' AS nodetext
            FROM items, agents, itemtypes
            WHERE items.itemtypeid = itemtypes.id AND items.userid = ? AND agents.id = items.manufacturerid
            ORDER BY agents.title`, userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "item:", true, "items"))
		return
	}

	if strings.HasPrefix(id, "itemtypes:") {
		typeID, err := intParam(strings.TrimPrefix(id, "itemtypes:"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid itemtype node id")
			return
		}
		rows, err := a.fetchRows(`SELECT items.id, agents.title || ' ' || items.model || ' [' || itemtypes.typedesc || ', ID:' || items.id || ']' AS nodetext
            FROM items, agents, itemtypes
            WHERE items.itemtypeid = itemtypes.id AND agents.id = items.manufacturerid AND items.itemtypeid = ?
            ORDER BY agents.title`, typeID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "item:", true, "items"))
		return
	}

	if strings.HasPrefix(id, "agenthw:") {
		agentID, err := intParam(strings.TrimPrefix(id, "agenthw:"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid hardware agent node id")
			return
		}
		rows, err := a.fetchRows(`SELECT items.id, items.model || ' [' || itemtypes.typedesc || ', ID:' || items.id || ']' AS nodetext
            FROM items, itemtypes
            WHERE items.manufacturerid = ? AND items.itemtypeid = itemtypes.id
            ORDER BY nodetext`, agentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "item:", true, "items"))
		return
	}

	if strings.HasPrefix(id, "agentsw:") {
		agentID, err := intParam(strings.TrimPrefix(id, "agentsw:"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid software agent node id")
			return
		}
		rows, err := a.fetchRows(`SELECT software.id, software.stitle || ' ' || software.sversion AS nodetext
            FROM software WHERE manufacturerid = ?`, agentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "software:", true, "software"))
		return
	}

	if strings.HasPrefix(id, "agentvendor:") {
		agentID, err := intParam(strings.TrimPrefix(id, "agentvendor:"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid vendor agent node id")
			return
		}
		rows, err := a.fetchRows(`SELECT invoices.id, invoices.number || ' ' || date(invoices.date, 'unixepoch') AS nodetext
            FROM invoices WHERE vendorid = ? ORDER BY invoices.date`, agentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "invoice:", true, "invoices"))
		return
	}

	if strings.HasPrefix(id, "agentbuyer:") {
		agentID, err := intParam(strings.TrimPrefix(id, "agentbuyer:"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid buyer agent node id")
			return
		}
		rows, err := a.fetchRows(`SELECT invoices.id, agents.title || ' ' || invoices.number || ' ' || date(invoices.date, 'unixepoch') AS nodetext
            FROM invoices, agents
            WHERE invoices.buyerid = ? AND agents.id = invoices.vendorid
            ORDER BY invoices.date`, agentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "invoice:", true, "invoices"))
		return
	}

	if strings.HasPrefix(id, "agentcontractor:") {
		agentID, err := intParam(strings.TrimPrefix(id, "agentcontractor:"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid contractor node id")
			return
		}
		rows, err := a.fetchRows(`SELECT contracts.id, contracts.number || ' ' || date(contracts.startdate, 'unixepoch') AS nodetext
            FROM contracts WHERE contractorid = ? ORDER BY contracts.startdate`, agentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toBrowseNodes(rows, "contract:", true, "contracts"))
		return
	}

	writeJSON(w, http.StatusOK, []browseNode{})
}

func safeLabelOrderExpr(raw string) string {
	switch strings.TrimSpace(raw) {
	case "id":
		return "items.id"
	case "id_desc":
		return "items.id DESC"
	case "model":
		return "items.model"
	case "status":
		return "items.status"
	default:
		return "itemtypes.typedesc"
	}
}

func (a *App) handleListLabelItems(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	orderExpr := safeLabelOrderExpr(r.URL.Query().Get("orderBy"))
	limit := intParamDefault(r.URL.Query().Get("limit"), 1000)
	offset := intParamDefault(r.URL.Query().Get("offset"), 0)

	where := ""
	args := []interface{}{}
	if search != "" {
		where = `WHERE (
            printf('%04d-%s', items.id, COALESCE(itemtypes.typedesc, '')) LIKE ? OR
            (CAST(items.id AS TEXT) || '-' || COALESCE(itemtypes.typedesc, '')) LIKE ? OR
            (
                COALESCE(agents.title, '-') || '-' ||
                COALESCE(items.model, '-') || '-' ||
                COALESCE(NULLIF(TRIM(items.sn), ''), '-') ||
                CASE
                    WHEN TRIM(COALESCE(items.label, '')) = '' THEN ''
                    ELSE '-' || TRIM(items.label)
                END
            ) LIKE ?
        )`
		q := "%" + search + "%"
		args = append(args, q, q, q)
	}

	query := `SELECT items.id, items.status,
        COALESCE(TRIM(statustypes.statusdesc), '') AS statusdesc,
        COALESCE(TRIM(statustypes.color), '') AS statuscolor,
        itemtypes.typedesc AS itemtype, agents.title AS manufacturer,
        items.model, items.sn, items.sn3, items.label
        FROM items
        JOIN itemtypes ON items.itemtypeid = itemtypes.id
        JOIN agents ON agents.id = items.manufacturerid
        LEFT JOIN statustypes ON statustypes.id = items.status
        ` + where + `
        ORDER BY
            CASE COALESCE(TRIM(statustypes.statusdesc), '')
                WHEN '使用中' THEN 0
                WHEN '库存' THEN 1
                WHEN '有故障' THEN 2
                WHEN '报废' THEN 3
                ELSE 4
            END,
            items.status,
            ` + orderExpr + `,
            itemtypes.typedesc,
            items.manufacturerid,
            items.id,
            items.sn,
            items.sn2,
            items.sn3
        LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range rows {
		id := asInt64(rows[i]["id"])
		itemType := asString(rows[i]["itemtype"])
		manufacturer := asString(rows[i]["manufacturer"])
		model := asString(rows[i]["model"])
		sn := asString(rows[i]["sn"])
		if sn == "" {
			sn = asString(rows[i]["sn3"])
		}
		label := asString(rows[i]["label"])
		labelSuffix := ""
		if label != "" {
			labelSuffix = "-" + label
		}
		rows[i]["text"] = fmt.Sprintf("%04d-%s | %s-%s-%s%s", id, itemType, manufacturer, model, sn, labelSuffix)
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleListLabelPresets(w http.ResponseWriter, r *http.Request) {
	rows, err := a.fetchRows(`SELECT * FROM labelpapers ORDER BY id ASC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (a *App) handleCreateLabelPreset(w http.ResponseWriter, r *http.Request) {
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

	query := `INSERT INTO labelpapers
        (rows, cols, lwidth, lheight, vpitch, hpitch, tmargin, bmargin, lmargin, rmargin, name,
         border, padding, fontsize, headerfontsize, barcodesize, idfontsize, wantbarcode, wantheadertext, wantheaderimage,
         headertext, image, imagewidth, imageheight, papersize, qrtext, wantnotext, wantraligntext)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := a.execLogged(
		r.Context(),
		user,
		clientIP(r),
		query,
		asInt64(body["rows"]),
		asInt64(body["cols"]),
		asString(body["lwidth"]),
		asString(body["lheight"]),
		asString(body["vpitch"]),
		asString(body["hpitch"]),
		asString(body["tmargin"]),
		asString(body["bmargin"]),
		asString(body["lmargin"]),
		asString(body["rmargin"]),
		name,
		asString(body["border"]),
		asString(body["padding"]),
		asString(body["fontsize"]),
		asString(body["headerfontsize"]),
		asString(body["barcodesize"]),
		asString(body["idfontsize"]),
		asInt64(body["wantbarcode"]),
		asInt64(body["wantheadertext"]),
		asInt64(body["wantheaderimage"]),
		asString(body["headertext"]),
		asString(body["image"]),
		asString(body["imagewidth"]),
		asString(body["imageheight"]),
		asString(body["papersize"]),
		asString(body["qrtext"]),
		asInt64(body["wantnotext"]),
		asInt64(body["wantraligntext"]),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (a *App) handleDeleteLabelPreset(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	id, err := intParam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := a.execLogged(r.Context(), user, clientIP(r), `DELETE FROM labelpapers WHERE id = ?`, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type labelPreviewRequest struct {
	ItemIDs    []int64 `json:"itemIds"`
	QRPrefix   string  `json:"qrPrefix"`
	HeaderText string  `json:"headerText"`
}

func makeInClause(ids []int64) (string, []interface{}) {
	ph := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		ph = append(ph, "?")
		args = append(args, id)
	}
	if len(ph) == 0 {
		return "", nil
	}
	return strings.Join(ph, ","), args
}

func (a *App) handlePreviewLabels(w http.ResponseWriter, r *http.Request) {
	var req labelPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	clause, args := makeInClause(req.ItemIDs)
	if clause == "" {
		writeJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	query := `SELECT items.id, itemtypes.typedesc AS itemtype, agents.title AS manufacturer,
        items.model, items.sn, items.sn3, items.label, items.dnsname, items.ipv4, items.ipv6
        FROM items
        JOIN itemtypes ON itemtypes.id = items.itemtypeid
        JOIN agents ON agents.id = items.manufacturerid
        WHERE items.id IN (` + clause + `)
        ORDER BY items.id`

	rows, err := a.fetchRows(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		id := asInt64(row["id"])
		itemType := asString(row["itemtype"])
		manufacturer := asString(row["manufacturer"])
		model := asString(row["model"])
		sn := asString(row["sn"])
		if sn == "" {
			sn = asString(row["sn3"])
		}
		label := asString(row["label"])
		text := fmt.Sprintf("%s %s %s %s", itemType, manufacturer, model, sn)
		qr := strings.TrimSpace(req.QRPrefix) + strconv.FormatInt(id, 10)
		out = append(out, map[string]interface{}{
			"id":           id,
			"itemType":     itemType,
			"manufacturer": manufacturer,
			"model":        model,
			"sn":           sn,
			"label":        label,
			"dnsName":      asString(row["dnsname"]),
			"ipv4":         asString(row["ipv4"]),
			"ipv6":         asString(row["ipv6"]),
			"text":         text,
			"headerText":   req.HeaderText,
			"qrText":       qr,
		})
	}
	writeJSON(w, http.StatusOK, out)
}
