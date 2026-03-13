package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"itdb-backend/cmd/common/primitives"
	"itdb-backend/cmd/common/statustypes"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var allowedFloorplanImageExtensions = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".gif":  {},
	".bmp":  {},
	".webp": {},
	".svg":  {},
	".avif": {},
}

var allowedInvoiceUploadExtensions = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".gif":  {},
	".bmp":  {},
	".webp": {},
	".svg":  {},
	".avif": {},
	".pdf":  {},
}

func isInvoiceFileType(fileTypeID int64, typeName string) bool {
	text := strings.TrimSpace(typeName)
	return fileTypeID == 3 || strings.EqualFold(text, "invoice") || text == "发票"
}

func (a *App) fetchRows(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rowsToMaps(rows)
}

func rowsToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			rowMap[col] = normalizeDBValue(values[i])
		}
		out = append(out, rowMap)
	}

	return out, rows.Err()
}

func normalizeDBValue(v interface{}) interface{} {
	switch t := v.(type) {
	case []byte:
		return string(t)
	default:
		return t
	}
}

func (a *App) fetchIDList(query string, arg interface{}) ([]int64, error) {
	rows, err := a.db.Query(query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id sql.NullInt64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id.Valid {
			ids = append(ids, id.Int64)
		}
	}
	return ids, rows.Err()
}

func (a *App) fetchIDListTx(tx *sql.Tx, query string, arg interface{}) ([]int64, error) {
	rows, err := tx.Query(query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id sql.NullInt64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id.Valid {
			ids = append(ids, id.Int64)
		}
	}
	return ids, rows.Err()
}

func (a *App) fetchStringList(query string, arg interface{}) ([]string, error) {
	rows, err := a.db.Query(query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := []string{}
	for rows.Next() {
		var v sql.NullString
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if v.Valid {
			values = append(values, v.String)
		}
	}
	return values, rows.Err()
}

func (a *App) replaceIDLinksTx(tx *sql.Tx, deleteQuery, insertQuery string, ownerID int64, links []int64) error {
	if _, err := tx.Exec(deleteQuery, ownerID); err != nil {
		return err
	}
	for _, linkedID := range links {
		if linkedID == 0 {
			continue
		}
		if _, err := tx.Exec(insertQuery, ownerID, linkedID); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) replaceUndirectedItemLinksTx(tx *sql.Tx, ownerID int64, links []int64) error {
	if _, err := tx.Exec(`DELETE FROM itemlink WHERE itemid1 = ? OR itemid2 = ?`, ownerID, ownerID); err != nil {
		return err
	}

	seen := make(map[[2]int64]struct{})
	for _, linkedID := range links {
		if linkedID == 0 || linkedID == ownerID {
			continue
		}

		left := ownerID
		right := linkedID
		if left > right {
			left, right = right, left
		}
		key := [2]int64{left, right}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		if _, err := tx.Exec(`INSERT INTO itemlink (itemid1, itemid2) VALUES (?, ?)`, left, right); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) countFileLinksTx(tx *sql.Tx, fileID int64) (int64, error) {
	var count int64
	query := `
SELECT
    (SELECT COUNT(*) FROM software2file WHERE fileid = ?) +
    (SELECT COUNT(*) FROM invoice2file WHERE fileid = ?) +
    (SELECT COUNT(*) FROM item2file WHERE fileid = ?) +
    (SELECT COUNT(*) FROM contract2file WHERE fileid = ?)
`
	err := tx.QueryRow(query, fileID, fileID, fileID, fileID).Scan(&count)
	return count, err
}

func (a *App) deleteFileTx(tx *sql.Tx, fileID int64) error {
	var fname sql.NullString
	_ = tx.QueryRow(`SELECT fname FROM files WHERE id = ?`, fileID).Scan(&fname)

	queries := []string{
		`DELETE FROM files WHERE id = ?`,
		`DELETE FROM invoice2file WHERE fileid = ?`,
		`DELETE FROM software2file WHERE fileid = ?`,
		`DELETE FROM item2file WHERE fileid = ?`,
		`DELETE FROM contract2file WHERE fileid = ?`,
	}

	for _, q := range queries {
		if _, err := tx.Exec(q, fileID); err != nil {
			return err
		}
	}

	if strings.TrimSpace(fname.String) != "" {
		_ = os.Remove(filepath.Join(a.cfg.UploadDir, fname.String))
	}

	return nil
}

func (a *App) cleanupRemovedFileLinksTx(tx *sql.Tx, previous []int64, current []int64, cleanupTargets []int64) error {
	if len(cleanupTargets) == 0 {
		return nil
	}
	previousSet := make(map[int64]struct{}, len(previous))
	for _, fid := range previous {
		if fid > 0 {
			previousSet[fid] = struct{}{}
		}
	}
	currentSet := make(map[int64]struct{}, len(current))
	for _, fid := range current {
		if fid > 0 {
			currentSet[fid] = struct{}{}
		}
	}

	seen := make(map[int64]struct{}, len(cleanupTargets))
	for _, fid := range cleanupTargets {
		if fid <= 0 {
			continue
		}
		if _, duplicated := seen[fid]; duplicated {
			continue
		}
		seen[fid] = struct{}{}
		if _, existed := previousSet[fid]; !existed {
			continue
		}
		if _, exists := currentSet[fid]; exists {
			continue
		}
		links, err := a.countFileLinksTx(tx, fid)
		if err != nil {
			return err
		}
		if links == 0 {
			if err := a.deleteFileTx(tx, fid); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *App) ensureTag(name string) (int64, error) {
	var id sql.NullInt64
	err := a.db.QueryRow(`SELECT id FROM tags WHERE LOWER(name) = LOWER(?) LIMIT 1`, strings.TrimSpace(name)).Scan(&id)
	if err == nil && id.Valid {
		return id.Int64, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	res, err := a.db.Exec(`INSERT INTO tags (name) VALUES (?)`, strings.TrimSpace(name))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (a *App) ensureTagLogged(ctx context.Context, user SessionUser, ip, name string) (int64, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return 0, errors.New("name is required")
	}

	var id sql.NullInt64
	err := a.db.QueryRowContext(ctx, `SELECT id FROM tags WHERE LOWER(name) = LOWER(?) LIMIT 1`, trimmed).Scan(&id)
	if err == nil && id.Valid {
		return id.Int64, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	res, err := a.execLogged(ctx, user, ip, `INSERT INTO tags (name) VALUES (?)`, trimmed)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func getMultipartFileHeader(form *multipart.Form, key string) (*multipart.FileHeader, error) {
	if form == nil || form.File == nil {
		return nil, errors.New("missing multipart form")
	}

	files := form.File[key]
	if len(files) == 0 {
		return nil, errors.New("missing file")
	}

	return files[0], nil
}

func (a *App) storeUploadedFile(fileHeader *multipart.FileHeader, fileTypeID int64, title string) (string, error) {
	if fileHeader == nil {
		return "", errors.New("missing file")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" {
		ext = ".bin"
	}

	var typeName sql.NullString
	_ = a.db.QueryRow(`SELECT typedesc FROM filetypes WHERE id = ?`, fileTypeID).Scan(&typeName)
	if isInvoiceFileType(fileTypeID, typeName.String) {
		if _, ok := allowedInvoiceUploadExtensions[ext]; !ok {
			return "", errors.New("invoice file must be an image or pdf")
		}
	}
	prefix := primitives.BuildFilenameSegment(typeName.String)
	titlePart := primitives.BuildFilenameSegment(title)
	name := fmt.Sprintf("%s-%s-%s%s", prefix, titlePart, shortUUID(), ext)
	filePath := filepath.Join(a.cfg.UploadDir, name)

	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return strings.ToLower(name), nil
}

func (a *App) storeFloorplanFile(fileHeader *multipart.FileHeader, locationName string) (string, error) {
	if fileHeader == nil {
		return "", errors.New("missing file")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if _, ok := allowedFloorplanImageExtensions[ext]; !ok {
		return "", errors.New("floorplan file must be an image")
	}

	locationPart := primitives.BuildFilenameSegment(locationName)
	name := fmt.Sprintf("floorplan-%s-%s%s", locationPart, shortUUID(), ext)
	filePath := filepath.Join(a.cfg.UploadDir, strings.ToLower(name))

	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return strings.ToLower(name), nil
}

func enforceDictionaryDeleteRules(tx *sql.Tx, name string, id int64) error {
	switch name {
	case "itemtypes":
		var count int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM items WHERE itemtypeid = ?`, id).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("该硬件类型已被 %d 条硬件记录使用，无法删除", count)
		}
	case "filetypes":
		if id <= 10 {
			return errors.New("内置文件类型不可删除")
		}
		var count int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM files WHERE type = ?`, id).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("该文件类型已被 %d 个文件记录使用，无法删除", count)
		}
	case "statustypes":
		var desc sql.NullString
		err := tx.QueryRow(`SELECT statusdesc FROM statustypes WHERE id = ?`, id).Scan(&desc)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if statustypes.IsProtectedStatusType(id, desc.String) {
			return errors.New("内置状态类型不可删除")
		}
		var count int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM items WHERE status = ?`, id).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("该状态类型已被 %d 条硬件记录使用，无法删除", count)
		}
	case "dpttypes":
		var count int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM items WHERE dptid = ?`, id).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("该所属部门已被 %d 条硬件记录使用，无法删除", count)
		}
	case "contracttypes":
		if id <= 1 {
			return errors.New("内置合同类型不可删除")
		}
		var count int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM contracts WHERE type = ?`, id).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("该合同类型已被 %d 条合同记录使用，无法删除", count)
		}
		if _, err := tx.Exec(`DELETE FROM contractsubtypes WHERE contypeid = ?`, id); err != nil {
			return err
		}
	case "tags":
		var itemCount, softCount int64
		if err := tx.QueryRow(`SELECT COUNT(tagid) FROM tag2item WHERE tagid = ?`, id).Scan(&itemCount); err != nil {
			return err
		}
		if err := tx.QueryRow(`SELECT COUNT(tagid) FROM tag2software WHERE tagid = ?`, id).Scan(&softCount); err != nil {
			return err
		}
		if itemCount > 0 || softCount > 0 {
			return fmt.Errorf("该标记仍有关联（硬件=%d 软件=%d），无法删除", itemCount, softCount)
		}
	case "contractsubtypes":
		return nil
	default:
		return errors.New("不支持的字典类型")
	}
	return nil
}

func enforceDictionaryUpdateRules(tx *sql.Tx, name string, id int64) error {
	switch name {
	case "filetypes":
		if id <= 10 {
			return errors.New("内置文件类型不可编辑")
		}
	case "statustypes":
		var desc sql.NullString
		if err := tx.QueryRow(`SELECT statusdesc FROM statustypes WHERE id = ?`, id).Scan(&desc); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("状态类型不存在")
			}
			return err
		}
		if statustypes.IsProtectedStatusType(id, desc.String) {
			return errors.New("内置状态类型不可编辑")
		}
	}
	return nil
}

func dictionaryUniqueTextRule(name string) (table string, column string, label string, scopeColumn string, err error) {
	switch name {
	case "itemtypes":
		return "itemtypes", "typedesc", "描述", "", nil
	case "filetypes":
		return "filetypes", "typedesc", "描述", "", nil
	case "statustypes":
		return "statustypes", "statusdesc", "描述", "", nil
	case "dpttypes":
		return "dpttypes", "dptname", "部门名称", "", nil
	case "contracttypes":
		return "contracttypes", "name", "类型名称", "", nil
	case "contractsubtypes":
		return "contractsubtypes", "name", "子类型名称", "contypeid", nil
	case "tags":
		return "tags", "name", "名称", "", nil
	default:
		return "", "", "", "", errors.New("不支持的字典类型")
	}
}

func enforceDictionaryUniqueText(tx *sql.Tx, name string, body map[string]interface{}, excludeID int64) error {
	table, column, label, scopeColumn, err := dictionaryUniqueTextRule(name)
	if err != nil {
		return err
	}

	text := strings.TrimSpace(asString(body[column]))
	if text == "" {
		return nil
	}

	query := fmt.Sprintf("SELECT id FROM %s WHERE LOWER(TRIM(COALESCE(%s, ''))) = LOWER(TRIM(?))", table, column)
	args := []interface{}{text}
	if scopeColumn != "" {
		query += fmt.Sprintf(" AND COALESCE(%s, 0) = ?", scopeColumn)
		args = append(args, asInt64(body[scopeColumn]))
	}
	if excludeID > 0 {
		query += " AND id <> ?"
		args = append(args, excludeID)
	}
	query += " LIMIT 1"

	var existingID int64
	if err := tx.QueryRow(query, args...).Scan(&existingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	return fmt.Errorf("%s“%s”已存在", label, text)
}

func dictionaryInsert(name string, body map[string]interface{}) (string, []interface{}, error) {
	switch name {
	case "itemtypes":
		hassoftware := int64(1)
		if raw, ok := body["hassoftware"]; ok {
			hassoftware = asInt64(raw)
		}
		return `INSERT INTO itemtypes (typedesc, hassoftware) VALUES (?, ?)`, []interface{}{asString(body["typedesc"]), hassoftware}, nil
	case "filetypes":
		return `INSERT INTO filetypes (typedesc) VALUES (?)`, []interface{}{asString(body["typedesc"])}, nil
	case "statustypes":
		desc := asString(body["statusdesc"])
		color := statustypes.NormalizeHexColor(asString(body["color"]))
		if fixedColor, ok := statustypes.FixedStatusTypeColor(desc); ok {
			color = fixedColor
		}
		return `INSERT INTO statustypes (statusdesc, color) VALUES (?, ?)`, []interface{}{desc, color}, nil
	case "dpttypes":
		return `INSERT INTO dpttypes (dptname) VALUES (?)`, []interface{}{asString(body["dptname"])}, nil
	case "contracttypes":
		return `INSERT INTO contracttypes (name) VALUES (?)`, []interface{}{asString(body["name"])}, nil
	case "contractsubtypes":
		return `INSERT INTO contractsubtypes (name, contypeid) VALUES (?, ?)`, []interface{}{asString(body["name"]), asInt64(body["contypeid"])}, nil
	case "tags":
		return `INSERT INTO tags (name) VALUES (?)`, []interface{}{asString(body["name"])}, nil
	default:
		return "", nil, errors.New("不支持的字典类型")
	}
}

func dictionaryUpdate(name string, id int64, body map[string]interface{}) (string, []interface{}, error) {
	switch name {
	case "itemtypes":
		return `UPDATE itemtypes SET typedesc = ?, hassoftware = ? WHERE id = ?`, []interface{}{asString(body["typedesc"]), asInt64(body["hassoftware"]), id}, nil
	case "filetypes":
		return `UPDATE filetypes SET typedesc = ? WHERE id = ?`, []interface{}{asString(body["typedesc"]), id}, nil
	case "statustypes":
		desc := asString(body["statusdesc"])
		color := statustypes.NormalizeHexColor(asString(body["color"]))
		if fixedColor, ok := statustypes.FixedStatusTypeColor(desc); ok {
			color = fixedColor
		}
		return `UPDATE statustypes SET statusdesc = ?, color = ? WHERE id = ?`, []interface{}{desc, color, id}, nil
	case "dpttypes":
		return `UPDATE dpttypes SET dptname = ? WHERE id = ?`, []interface{}{asString(body["dptname"]), id}, nil
	case "contracttypes":
		return `UPDATE contracttypes SET name = ? WHERE id = ?`, []interface{}{asString(body["name"]), id}, nil
	case "contractsubtypes":
		return `UPDATE contractsubtypes SET name = ?, contypeid = ? WHERE id = ?`, []interface{}{asString(body["name"]), asInt64(body["contypeid"]), id}, nil
	case "tags":
		return `UPDATE tags SET name = ? WHERE id = ?`, []interface{}{asString(body["name"]), id}, nil
	default:
		return "", nil, errors.New("不支持的字典类型")
	}
}

func dictionaryDelete(name string) (string, error) {
	switch name {
	case "itemtypes":
		return `DELETE FROM itemtypes WHERE id = ?`, nil
	case "filetypes":
		return `DELETE FROM filetypes WHERE id = ?`, nil
	case "statustypes":
		return `DELETE FROM statustypes WHERE id = ?`, nil
	case "dpttypes":
		return `DELETE FROM dpttypes WHERE id = ?`, nil
	case "contracttypes":
		return `DELETE FROM contracttypes WHERE id = ?`, nil
	case "contractsubtypes":
		return `DELETE FROM contractsubtypes WHERE id = ?`, nil
	case "tags":
		return `DELETE FROM tags WHERE id = ?`, nil
	default:
		return "", errors.New("不支持的字典类型")
	}
}

func (a *App) execTxLogged(ctx context.Context, tx *sql.Tx, user SessionUser, ip, query string, args ...interface{}) (sql.Result, error) {
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	normalized := strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(normalized, "SELECT") {
		return res, nil
	}

	histSQL := query
	if len(args) > 0 {
		histSQL = fmt.Sprintf("%s | args=%v", query, args)
	}
	now := time.Now().Unix()
	_, _ = tx.ExecContext(ctx, `INSERT INTO history (date, sql, authuser, ip) VALUES (?, ?, ?, ?)`, now, histSQL, user.Username, ip)
	if a.cfg.HistoryLimit > 0 {
		_, _ = tx.ExecContext(ctx, `DELETE FROM history WHERE id < (SELECT COALESCE(MAX(id), 0) - ? FROM history)`, a.cfg.HistoryLimit)
	}
	return res, nil
}
