package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type reportDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Query       string `json:"-"`
	ChartType   string `json:"chartType,omitempty"`
	ChartX      string `json:"chartX,omitempty"`
	ChartY      string `json:"chartY,omitempty"`
	ChartLimit  int    `json:"chartLimit,omitempty"`
}

func allReportDefinitions() []reportDefinition {
	return []reportDefinition{
		{
			Name:        "itemperagent",
			Description: "厂商（Agent）对应资产数量",
			Query: `SELECT COUNT(*) AS totalcount, agents.title AS Agent, agents.id AS ID
                    FROM items, agents
                    WHERE agents.id = items.manufacturerid
                    GROUP BY manufacturerid
                    ORDER BY totalcount DESC`,
			ChartType:  "pie",
			ChartX:     "Agent",
			ChartY:     "totalcount",
			ChartLimit: 15,
		},
		{
			Name:        "softwareperagent",
			Description: "厂商（Agent）对应软件数量",
			Query: `SELECT COUNT(*) AS totalcount, agents.title AS Agent, agents.id AS ID
                    FROM software, agents
                    WHERE agents.id = software.manufacturerid
                    GROUP BY manufacturerid
                    ORDER BY totalcount DESC`,
			ChartType:  "pie",
			ChartX:     "Agent",
			ChartY:     "totalcount",
			ChartLimit: 15,
		},
		{
			Name:        "invoicesperagent",
			Description: "供应商（Agent）对应发票数量",
			Query: `SELECT COUNT(*) AS totalcount, agents.title AS Agent, agents.id AS ID
                    FROM invoices, agents
                    WHERE agents.id = invoices.vendorid
                    GROUP BY vendorid
                    ORDER BY totalcount DESC`,
			ChartType:  "pie",
			ChartX:     "Agent",
			ChartY:     "totalcount",
			ChartLimit: 15,
		},
		{
			Name:        "itemsperlocation",
			Description: "各位置资产数量",
			Query: `SELECT COUNT(*) AS totalcount, locations.name || ' Floor:' || locations.floor AS Location
                    FROM items, agents, locations
                    WHERE agents.id = items.manufacturerid AND items.locationid = locations.id
                    GROUP BY locationid
                    ORDER BY totalcount DESC`,
			ChartType:  "pie",
			ChartX:     "Location",
			ChartY:     "totalcount",
			ChartLimit: 15,
		},
		{
			Name:        "percsupitems",
			Description: "资产维保状态统计",
			Query: `SELECT
                    '未过保' AS Type, (SELECT COUNT(id) FROM items WHERE ((purchasedate+warrantymonths*30*24*60*60-strftime('%s'))/(60*60*24)) > 1 AND purchasedate > 0 AND warrantymonths > 0) AS Items
                    UNION SELECT
                    '已过保' AS Type, (SELECT COUNT(id) FROM items WHERE ((purchasedate+warrantymonths*30*24*60*60-strftime('%s'))/(60*60*24)) <= 1 AND purchasedate > 0 AND warrantymonths > 0) AS Items
                    UNION SELECT
                    '未知' AS Type, (SELECT COUNT(id) FROM items WHERE purchasedate = 0 OR purchasedate IS NULL OR warrantymonths = 0 OR warrantymonths IS NULL) AS Items
                    UNION SELECT
                    '总数' AS Type, (SELECT COUNT(id) FROM items) AS Items`,
			ChartType:  "pie",
			ChartX:     "Type",
			ChartY:     "Items",
			ChartLimit: 15,
		},
		{
			Name:        "itemlistperlocation",
			Description: "按位置输出资产清单",
			Query: `SELECT items.id AS ID, typedesc AS type, agents.title AS manufacturer, model, dnsname,
                    locations.name || ' Floor:' || locations.floor || ' Area:' || (SELECT locareas.areaname FROM locareas WHERE locareas.id = items.locareaid) AS Location
                    FROM items
                    INNER JOIN agents ON agents.id = items.manufacturerid
                    INNER JOIN locations ON items.locationid = locations.id
                    INNER JOIN itemtypes ON itemtypes.id = items.itemtypeid
                    ORDER BY items.locationid, typedesc DESC`,
		},
		{
			Name:        "itemsendwarranty",
			Description: "临近维保到期资产（前后360天）",
			Query: `SELECT report_rows.ID, report_rows.ipv4, report_rows.type, report_rows.manufacturer,
                    report_rows.model, report_rows.dnsname, report_rows.label, report_rows.RemainingDays
                    FROM (
                        SELECT items.id AS ID, items.ipv4, itemtypes.typedesc AS type, agents.title AS manufacturer,
                               items.model, items.dnsname, items.label,
                               ((strftime('%s', items.purchasedate, 'unixepoch', '+' || items.warrantymonths || ' months') - strftime('%s', 'now'))/(60*60*24)) AS RemainingDays
                        FROM items, itemtypes, agents
                        WHERE agents.id = manufacturerid
                          AND itemtypes.id = items.itemtypeid
                    ) AS report_rows
                    WHERE report_rows.RemainingDays > -360
                      AND report_rows.RemainingDays < 360
                    ORDER BY report_rows.RemainingDays`,
		},
		{
			Name:        "allips",
			Description: "已配置 IPv4 的资产列表",
			Query: `SELECT items.id AS ID, items.ipv4, items.ipv6, typedesc AS type, agents.title AS manufacturer, model, dnsname, label
                    FROM items, itemtypes, agents
                    WHERE agents.id = manufacturerid AND itemtypes.id = items.itemtypeid AND ipv4 <> ''
                    ORDER BY ipv4`,
		},
		{
			Name:        "noinvoice",
			Description: "未关联发票资产",
			Query: `SELECT items.id AS ID, typedesc AS type, agents.title AS manufacturer, model,
                    strftime('%Y-%m-%d', purchasedate, 'unixepoch') AS PurchaseDate
                    FROM items, itemtypes, agents
                    WHERE agents.id = manufacturerid
                      AND itemtypes.id = items.itemtypeid
                      AND items.id NOT IN (SELECT itemid FROM item2inv)`,
		},
		{
			Name:        "nolocation",
			Description: "未配置位置资产",
			Query: `SELECT items.id AS ID, typedesc AS type, agents.title AS manufacturer, model
                    FROM items, itemtypes, agents
                    WHERE agents.id = manufacturerid
                      AND itemtypes.id = items.itemtypeid
                      AND (locationid = '' OR locationid IS NULL OR locationid = 0)`,
		},
		{
			Name:        "depreciation3",
			Description: "资产折旧估值（3年）",
			Query: `SELECT items.id AS ID, typedesc AS type, agents.title AS manufacturer, model,
                    strftime('%Y-%m-%d', purchasedate, 'unixepoch') AS PurchaseDate,
                    purchprice AS PurchasePrice,
                    CAST(((strftime('%s', 'now') - purchasedate)/(60*60*24*30.4)*(purchasedate AND 1)) AS INTEGER) AS Months,
                    (purchprice - purchprice/36*CAST(((strftime('%s', 'now') - purchasedate)/(60*60*24*30.4)*(purchasedate AND 1)) AS INTEGER)) AS CurrentValue
                    FROM items, itemtypes, agents
                    WHERE agents.id = manufacturerid AND itemtypes.id = items.itemtypeid`,
		},
		{
			Name:        "depreciation5",
			Description: "资产折旧估值（5年）",
			Query: `SELECT items.id AS ID, typedesc AS type, agents.title AS manufacturer, model,
                    strftime('%Y-%m-%d', purchasedate, 'unixepoch') AS PurchaseDate,
                    purchprice AS PurchasePrice,
                    CAST(((strftime('%s', 'now') - purchasedate)/(60*60*24*30.4)*(purchasedate AND 1)) AS INTEGER) AS Months,
                    (purchprice - purchprice/60*CAST(((strftime('%s', 'now') - purchasedate)/(60*60*24*30.4)*(purchasedate AND 1)) AS INTEGER)) AS CurrentValue
                    FROM items, itemtypes, agents
                    WHERE agents.id = manufacturerid AND itemtypes.id = items.itemtypeid`,
		},
	}
}

func findReportByName(name string) (reportDefinition, bool) {
	for _, report := range allReportDefinitions() {
		if report.Name == name {
			return report, true
		}
	}
	return reportDefinition{}, false
}

func (a *App) handleListReports(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, allReportDefinitions())
}

func (a *App) handleRunReport(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(chi.URLParam(r, "name"))
	report, ok := findReportByName(name)
	if !ok {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}

	limit := intParamDefault(r.URL.Query().Get("limit"), 1000)
	if limit <= 0 {
		limit = 1000
	}
	query := report.Query + " LIMIT ?"
	rows, err := a.fetchRows(query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	chartRows := []map[string]interface{}{}
	if report.ChartType == "pie" && report.ChartX != "" && report.ChartY != "" {
		maxRows := report.ChartLimit
		if maxRows <= 0 {
			maxRows = 15
		}
		for _, row := range rows {
			if len(chartRows) >= maxRows {
				break
			}
			if strings.EqualFold(asString(row[report.ChartX]), "Total") {
				continue
			}
			chartRows = append(chartRows, map[string]interface{}{
				"x": asString(row[report.ChartX]),
				"y": asInt64(row[report.ChartY]),
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"meta": map[string]interface{}{
			"name":        report.Name,
			"description": report.Description,
			"chartType":   report.ChartType,
			"chartX":      report.ChartX,
			"chartY":      report.ChartY,
			"chartLimit":  report.ChartLimit,
		},
		"rows":  rows,
		"chart": chartRows,
	})
}
