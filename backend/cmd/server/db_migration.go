/*
项目名称：itdb
文件名称：db_migration.go
创建时间：2026-05-11 15:30:00

系统用户：Jerion
作　　者：Jerion
联系邮箱：416685476@qq.com
功能描述：旧平台数据库迁移转换功能，处理 settings 和 statustypes 表结构差异
*/

package server

import (
	"database/sql"
	"fmt"
	"log"

	"itdb-backend/cmd/common/statustypes"
)

// migrateOldDatabaseSchema 将旧平台数据库结构迁移到新平台
// 主要处理：
// 1. settings 表：从旧结构（包含 companytitle, dateformat 等字段）转换为新结构（仅 LDAP 字段）
// 2. statustypes 表：添加 color 字段，并为固定的 4 个状态类型设置颜色
func migrateOldDatabaseSchema(db *sql.DB) error {
	log.Println("Checking and migrating legacy database schema...")

	// 1. 迁移 settings 表
	if err := migrateSettingsTable(db); err != nil {
		return fmt.Errorf("migrate settings table failed: %w", err)
	}

	// 2. 迁移 statustypes 表
	if err := migrateStatusTypesTable(db); err != nil {
		return fmt.Errorf("migrate statustypes table failed: %w", err)
	}

	log.Println("Database schema migration completed")
	return nil
}

// migrateSettingsTable 迁移 settings 表结构
// 旧结构：companytitle, dateformat, currency, lang, version, timezone, dbversion, useldap, ldap_server, ldap_dn, ldap_getusers, ldap_getusers_filter
// 新结构：useldap, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter
func migrateSettingsTable(db *sql.DB) error {
	// 检查 settings 表是否存在
	var tableName string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='settings'`).Scan(&tableName)
	if err == sql.ErrNoRows {
		log.Println("settings table does not exist, skipping migration")
		return nil
	}
	if err != nil {
		return err
	}

	// 获取 settings 表的所有列
	columns, err := getTableColumns(db, "settings")
	if err != nil {
		return err
	}

	// 检查是否是旧结构（包含 companytitle, dateformat 等字段）
	hasOldFields := false
	for _, col := range columns {
		if col == "companytitle" || col == "dateformat" || col == "currency" || col == "lang" || col == "version" || col == "timezone" || col == "dbversion" {
			hasOldFields = true
			break
		}
	}

	if !hasOldFields {
		log.Println("settings table is already using the current schema, skipping migration")
		return nil
	}

	log.Println("Legacy settings table detected, starting migration...")

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 创建新结构的 settings 表
	if _, err := tx.Exec(`CREATE TABLE settings_new (
		useldap integer default 0,
		ldap_server TEXT,
		ldap_dn TEXT,
		ldap_bind_dn TEXT,
		ldap_bind_password TEXT,
		ldap_getusers TEXT,
		ldap_getusers_filter TEXT
	)`); err != nil {
		return err
	}

	// 从旧表复制 LDAP 相关字段（如果存在）
	hasColumns := make(map[string]bool)
	for _, col := range columns {
		hasColumns[col] = true
	}

	useLDAPExpr := "0"
	if hasColumns["useldap"] {
		useLDAPExpr = "COALESCE(useldap, 0)"
	}
	ldapServerExpr := "''"
	if hasColumns["ldap_server"] {
		ldapServerExpr = "COALESCE(ldap_server, '')"
	}
	ldapDNExpr := "''"
	if hasColumns["ldap_dn"] {
		ldapDNExpr = "COALESCE(ldap_dn, '')"
	}
	ldapBindDNExpr := "''"
	if hasColumns["ldap_bind_dn"] {
		ldapBindDNExpr = "COALESCE(ldap_bind_dn, '')"
	}
	ldapBindPasswordExpr := "''"
	if hasColumns["ldap_bind_password"] {
		ldapBindPasswordExpr = "COALESCE(ldap_bind_password, '')"
	}
	ldapGetUsersExpr := "''"
	if hasColumns["ldap_getusers"] {
		ldapGetUsersExpr = "COALESCE(ldap_getusers, '')"
	}
	ldapGetUsersFilterExpr := "''"
	if hasColumns["ldap_getusers_filter"] {
		ldapGetUsersFilterExpr = "COALESCE(ldap_getusers_filter, '')"
	}

	// 复制数据
	copyQuery := fmt.Sprintf(`INSERT INTO settings_new (useldap, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter)
		SELECT %s, %s, %s, %s, %s, %s, %s
		FROM settings
		LIMIT 1`, useLDAPExpr, ldapServerExpr, ldapDNExpr, ldapBindDNExpr, ldapBindPasswordExpr, ldapGetUsersExpr, ldapGetUsersFilterExpr)

	if _, err := tx.Exec(copyQuery); err != nil {
		return err
	}

	// 删除旧表
	if _, err := tx.Exec(`DROP TABLE settings`); err != nil {
		return err
	}

	// 重命名新表
	if _, err := tx.Exec(`ALTER TABLE settings_new RENAME TO settings`); err != nil {
		return err
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("settings table migration completed")
	return nil
}

// migrateStatusTypesTable 迁移 statustypes 表结构
// 旧结构：id, statusdesc
// 新结构：id, statusdesc, color
func migrateStatusTypesTable(db *sql.DB) error {
	// 检查 statustypes 表是否存在
	var tableName string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='statustypes'`).Scan(&tableName)
	if err == sql.ErrNoRows {
		log.Println("statustypes table does not exist, skipping migration")
		return nil
	}
	if err != nil {
		return err
	}

	// 获取 statustypes 表的所有列
	columns, err := getTableColumns(db, "statustypes")
	if err != nil {
		return err
	}

	// 检查是否已有 color 字段
	hasColorField := false
	for _, col := range columns {
		if col == "color" {
			hasColorField = true
			break
		}
	}

	if hasColorField {
		log.Println("statustypes table already has color column, checking color values...")
		return updateStatusTypesColors(db)
	}

	log.Println("Legacy statustypes table without color column detected, starting migration...")

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 创建新结构的 statustypes 表
	if _, err := tx.Exec(`CREATE TABLE statustypes_new (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		statusdesc TEXT,
		color TEXT
	)`); err != nil {
		return err
	}

	// 从旧表复制数据，并为固定的 4 个状态类型设置颜色
	rows, err := tx.Query(`SELECT id, statusdesc FROM statustypes ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	fixedColors := statustypes.FixedStatusTypeColors()

	for rows.Next() {
		var id int64
		var statusDesc string
		if err := rows.Scan(&id, &statusDesc); err != nil {
			return err
		}

		// 根据状态描述获取对应的颜色
		color, ok := fixedColors[statusDesc]
		if !ok {
			color = "" // 非固定状态类型，颜色为空
		}

		if _, err := tx.Exec(`INSERT INTO statustypes_new (id, statusdesc, color) VALUES (?, ?, ?)`,
			id, statusDesc, color); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// 删除旧表
	if _, err := tx.Exec(`DROP TABLE statustypes`); err != nil {
		return err
	}

	// 重命名新表
	if _, err := tx.Exec(`ALTER TABLE statustypes_new RENAME TO statustypes`); err != nil {
		return err
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("statustypes table migration completed")
	return nil
}

// updateStatusTypesColors 更新 statustypes 表中固定状态类型的颜色
func updateStatusTypesColors(db *sql.DB) error {
	fixedColors := statustypes.FixedStatusTypeColors()

	for statusDesc, color := range fixedColors {
		result, err := db.Exec(`UPDATE statustypes SET color = ? WHERE statusdesc = ? AND (color IS NULL OR color = '')`,
			color, statusDesc)
		if err != nil {
			return err
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.Printf("Updated status type %q color to %s", statusDesc, color)
		}
	}

	return nil
}

// getTableColumns 获取表的所有列名
func getTableColumns(db *sql.DB, tableName string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notNull int
		var dfltValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}

	return columns, rows.Err()
}
