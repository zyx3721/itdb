package server

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var databaseBootstrapSQL = []string{
	`CREATE TABLE actions (id INTEGER PRIMARY KEY AUTOINCREMENT,itemid INTEGER, actiondate integer, description, invoiceinfo, isauto, entrydate integer)`,
	`CREATE TABLE agents (id INTEGER PRIMARY KEY AUTOINCREMENT, type integer, title, contactinfo, contacts, urls)`,
	`CREATE TABLE contract2file (contractid integer, fileid integer)`,
	`CREATE TABLE contract2inv (invid integer, contractid integer)`,
	`CREATE TABLE contract2item (itemid integer, contractid integer)`,
	`CREATE TABLE contract2soft (softid integer, contractid integer)`,
	`CREATE TABLE contractevents (id integer primary key autoincrement, siblingid integer, contractid integer, startdate integer, enddate integer, description)`,
	`CREATE TABLE contracts (id integer primary key autoincrement, type integer, parentid integer, title, number, description, comments, totalcost, contractorid integer, startdate integer, currentenddate integer, renewals, subtype integer)`,
	`CREATE TABLE contractsubtypes (id INTEGER PRIMARY KEY AUTOINCREMENT,contypeid integer, name)`,
	`CREATE TABLE contracttypes (id INTEGER PRIMARY KEY AUTOINCREMENT, name)`,
	`CREATE TABLE dpttypes (id INTEGER PRIMARY KEY AUTOINCREMENT, typeid, dptname)`,
	`CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, type, title, fname, uploader, uploaddate, date integer)`,
	`CREATE TABLE filetypes (id INTEGER PRIMARY KEY AUTOINCREMENT, typedesc)`,
	`CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, date integer, sql, authuser, ip)`,
	`CREATE TABLE invoice2file (invoiceid INTEGER,fileid INTEGER)`,
	`CREATE TABLE invoices (id INTEGER PRIMARY KEY, number, date integer, vendorid integer, buyerid integer, description)`,
	`CREATE TABLE item2file (itemid INTEGER, fileid INTEGER)`,
	`CREATE TABLE item2inv (itemid integer, invid integer)`,
	`CREATE TABLE item2soft (itemid INTEGER, softid INTEGER, instdate INTEGER)`,
	`CREATE TABLE itemlink (itemid1, itemid2)`,
	`CREATE TABLE items (id INTEGER PRIMARY KEY AUTOINCREMENT, itemtypeid integer, function, manufacturerid integer, model, sn, sn2, sn3, origin, warrantymonths integer, purchasedate integer, purchprice, dnsname, maintenanceinfo, comments, ispart integer, hd, cpu, ram, raid TEXT, raidconfig TEXT, locationid integer, userid integer, dptid INTEGER, principal TEXT, ipv4, ipv6, usize integer, rackmountable integer, macs, remadmip, panelport, ports integer, switchport, switchid integer, rackid integer, rackposition integer, label, status integer, cpuno INTEGER, corespercpu INTEGER, rackposdepth integer, warrinfo, locareaid number)`,
	`CREATE TABLE itemtypes (id INTEGER PRIMARY KEY AUTOINCREMENT, typeid, typedesc, hassoftware integer)`,
	`CREATE TABLE labelpapers (id INTEGER PRIMARY KEY AUTOINCREMENT,rows integer, cols integer, lwidth real, lheight real, vpitch real, hpitch real, tmargin real, bmargin real, lmargin real, rmargin real, name, border, padding, headerfontsize, idfontsize, wantheadertext, wantheaderimage, headertext, fontsize, wantbarcode, barcodesize, image, imagewidth, imageheight, papersize, qrtext, wantnotext, wantraligntext)`,
	`CREATE TABLE locareas (id  INTEGER PRIMARY KEY AUTOINCREMENT, locationid number, areaname, x1 number, y1 number, x2 number, y2 number)`,
	`CREATE TABLE locations (id INTEGER PRIMARY KEY AUTOINCREMENT, name, floor, floorplanfn)`,
	`CREATE TABLE racks (id INTEGER PRIMARY KEY AUTOINCREMENT, locationid integer, usize integer, depth integer, comments,model,label, revnums integer, locareaid number)`,
	`CREATE TABLE settings (useldap integer default 0, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter)`,
	`CREATE TABLE soft2inv (invid integer, softid integer)`,
	`CREATE TABLE software (id INTEGER PRIMARY KEY autoincrement, invoiceid integer,slicenseinfo ,stype ,manufacturerid integer ,stitle ,sversion ,sinfo ,purchdate integer,licqty integer, lictype integer)`,
	`CREATE TABLE software2file (softwareid INTEGER, fileid INTEGER)`,
	`CREATE TABLE statustypes (id INTEGER PRIMARY KEY AUTOINCREMENT, statusdesc, color TEXT)`,
	`CREATE TABLE tag2item (itemid integer, tagid integer)`,
	`CREATE TABLE tag2software (softwareid integer, tagid integer)`,
	`CREATE TABLE tags (id INTEGER PRIMARY KEY AUTOINCREMENT, name)`,
	`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username, userdesc, pass,cookie1, usertype integer)`,
	`CREATE TABLE viewhist (id INTEGER PRIMARY KEY AUTOINCREMENT, url,description)`,
	`INSERT INTO "itemtypes" ("id", "typeid", "typedesc", "hassoftware") VALUES (1, NULL, '服务器', 1)`,
	`INSERT INTO "itemtypes" ("id", "typeid", "typedesc", "hassoftware") VALUES (2, NULL, '存储', 1)`,
	`INSERT INTO "itemtypes" ("id", "typeid", "typedesc", "hassoftware") VALUES (3, NULL, '交换机', 1)`,
	`INSERT INTO "itemtypes" ("id", "typeid", "typedesc", "hassoftware") VALUES (4, NULL, '电话', 1)`,
	`INSERT INTO "itemtypes" ("id", "typeid", "typedesc", "hassoftware") VALUES (5, NULL, '安防', 1)`,
	`INSERT INTO "contracttypes" ("id", "name") VALUES (1, '支持 & 维护')`,
	`INSERT INTO "statustypes" ("id", "statusdesc", "color") VALUES (1, '使用中', '#2f7fba')`,
	`INSERT INTO "statustypes" ("id", "statusdesc", "color") VALUES (2, '库存', '#16a34a')`,
	`INSERT INTO "statustypes" ("id", "statusdesc", "color") VALUES (3, '有故障', '#dc2626')`,
	`INSERT INTO "statustypes" ("id", "statusdesc", "color") VALUES (4, '报废', '#9ca3af')`,
	`INSERT INTO "filetypes" ("id", "typedesc") VALUES (1, '照片')`,
	`INSERT INTO "filetypes" ("id", "typedesc") VALUES (2, '手册')`,
	`INSERT INTO "filetypes" ("id", "typedesc") VALUES (3, '发票')`,
	`INSERT INTO "filetypes" ("id", "typedesc") VALUES (4, '报价')`,
	`INSERT INTO "filetypes" ("id", "typedesc") VALUES (5, '订单')`,
	`INSERT INTO "filetypes" ("id", "typedesc") VALUES (6, '服务')`,
	`INSERT INTO "filetypes" ("id", "typedesc") VALUES (7, '报告')`,
	`INSERT INTO "filetypes" ("id", "typedesc") VALUES (8, '许可证')`,
	`INSERT INTO "filetypes" ("id", "typedesc") VALUES (9, '合同')`,
	`INSERT INTO "filetypes" ("id", "typedesc") VALUES (10, '其他')`,
	`INSERT INTO labelpapers (
		id, rows, cols, lwidth, lheight, vpitch, hpitch, tmargin, bmargin, lmargin, rmargin,
		name, border, padding, headerfontsize, idfontsize, wantheadertext, wantheaderimage,
		headertext, fontsize, wantbarcode, barcodesize, image, imagewidth, imageheight,
		papersize, qrtext, wantnotext, wantraligntext
	) VALUES (
		1, 6, 2, 96.0, 42.3, 42.3, 98.5, 21.5, 7.7, 2.7, 7.7,
		'Avery6106', 200, 1, 6, 7, 1, 1,
		'Header Text', 6, 0, 20, 'images/itdb.png', 5, 5,
		'A4', NULL, NULL, NULL
	);`,
	`INSERT INTO labelpapers (
		id, rows, cols, lwidth, lheight, vpitch, hpitch, tmargin, bmargin, lmargin, rmargin,
		name, border, padding, headerfontsize, idfontsize, wantheadertext, wantheaderimage,
		headertext, fontsize, wantbarcode, barcodesize, image, imagewidth, imageheight,
		papersize, qrtext, wantnotext, wantraligntext
	) VALUES (
		10, 12, 4, 45.7, 21.2, 21.2, 48.3, 21.5, 21.0, 9.7, 9.7,
		'AveryL6009', 200, 1, 6, 7, 1, 1,
		'Header Text', 6, 0, 20, 'images/itdb.png', 5, 5,
		'A4', NULL, NULL, NULL
	);`,
	`INSERT INTO labelpapers (
		id, rows, cols, lwidth, lheight, vpitch, hpitch, tmargin, bmargin, lmargin, rmargin,
		name, border, padding, headerfontsize, idfontsize, wantheadertext, wantheaderimage,
		headertext, fontsize, wantbarcode, barcodesize, image, imagewidth, imageheight,
		papersize, qrtext, wantnotext, wantraligntext
	) VALUES (
		14, 12, 4, 45.7, 21.2, 21.2, 48.3, 21.5, 21.0, 11.0, 8.0,
		'AveryL6009-6110', 200, 1, 6, 7, 1, 1,
		'Header Text', 6, 0, 20, 'images/itdb.png', 5, 5,
		'A4', NULL, NULL, NULL
	);`,
}

func ensureDatabaseInitialized(cfg Config) error {
	dbPath := strings.TrimSpace(cfg.DBPath)
	if dbPath == "" || strings.EqualFold(dbPath, ":memory:") {
		return nil
	}

	if _, err := os.Stat(dbPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return err
	}

	if len(databaseBootstrapSQL) == 0 {
		return fmt.Errorf("database bootstrap sql is empty")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := setupSQLite(db); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for i, statement := range databaseBootstrapSQL {
		if _, err := tx.Exec(statement); err != nil {
			_ = tx.Rollback()
			preview := statement
			if len(preview) > 240 {
				preview = preview[:240] + "..."
			}
			return fmt.Errorf("execute bootstrap sql #%d failed: %w; statement=%s", i+1, err, preview)
		}
	}

	adminPass, err := hashPassword("admin123")
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("hash default admin password failed: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO users (username, userdesc, pass, usertype) VALUES (?, ?, ?, ?)`,
		"admin", "administrator", adminPass, 0); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("insert default admin failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("数据库初始化完成: db=%s, statements=%d", dbPath, len(databaseBootstrapSQL)+1)
	return nil
}
