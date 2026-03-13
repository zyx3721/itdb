package server

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

const defaultLDAPLoginAttr = "sAMAccountName"

type ldapSettings struct {
	UseLDAP            int64
	LDAPServer         string
	LDAPDN             string
	LDAPBindDN         string
	LDAPBindPassword   string
	LDAPGetUsers       string
	LDAPGetUsersFilter string
}

func loadLDAPSettings(db *sql.DB, legacyKey string) (ldapSettings, error) {
	var cfg ldapSettings
	err := db.QueryRow(`SELECT useldap, ldap_server, ldap_dn, ldap_bind_dn, ldap_bind_password, ldap_getusers, ldap_getusers_filter FROM settings LIMIT 1`).Scan(
		&cfg.UseLDAP,
		&cfg.LDAPServer,
		&cfg.LDAPDN,
		&cfg.LDAPBindDN,
		&cfg.LDAPBindPassword,
		&cfg.LDAPGetUsers,
		&cfg.LDAPGetUsersFilter,
	)
	if err != nil {
		return ldapSettings{}, err
	}
	cfg.LDAPBindPassword, err = decryptSettingsSecret(cfg.LDAPBindPassword, legacyKey)
	if err != nil {
		return ldapSettings{}, err
	}
	return cfg, nil
}

func testLDAPSettings(req settingsPayload) error {
	cfg := ldapSettings{
		UseLDAP:            req.UseLDAP,
		LDAPServer:         req.LDAPServer,
		LDAPDN:             req.LDAPDN,
		LDAPBindDN:         req.LDAPBindDN,
		LDAPBindPassword:   req.LDAPBindPassword,
		LDAPGetUsers:       req.LDAPGetUsers,
		LDAPGetUsersFilter: req.LDAPGetUsersFilter,
	}
	conn, err := dialAndBindLDAP(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

func authenticateLDAPUser(db *sql.DB, legacyKey, username, password string) error {
	cfg, err := loadLDAPSettings(db, legacyKey)
	if err != nil {
		return err
	}
	if cfg.UseLDAP != 1 {
		return errors.New("LDAP 未启用")
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("password is required")
	}

	conn, err := dialAndBindLDAP(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	userDN, err := searchLDAPUserDN(conn, cfg, username)
	if err != nil {
		return err
	}

	if err := conn.Bind(userDN, password); err != nil {
		if isLDAPInvalidCredentialsError(err) {
			return errors.New("invalid username or password")
		}
		return err
	}
	return nil
}

func dialAndBindLDAP(cfg ldapSettings) (*ldap.Conn, error) {
	serverURL := normalizeLDAPServerURL(cfg.LDAPServer)
	if serverURL == "" {
		return nil, errors.New("ldap server is required")
	}
	if strings.TrimSpace(cfg.LDAPDN) == "" {
		return nil, errors.New("ldap base DN is required")
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := ldap.DialURL(serverURL, ldap.DialWithDialer(dialer))
	if err != nil {
		return nil, err
	}

	bindDN := strings.TrimSpace(cfg.LDAPBindDN)
	bindPassword := cfg.LDAPBindPassword
	if bindDN == "" {
		if strings.TrimSpace(bindPassword) != "" {
			conn.Close()
			return nil, errors.New("ldap bind DN is required")
		}
		return conn, nil
	}

	if err := conn.Bind(bindDN, bindPassword); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func searchLDAPUserDN(conn *ldap.Conn, cfg ldapSettings, username string) (string, error) {
	searchBase := strings.TrimSpace(cfg.LDAPDN)
	if searchBase == "" {
		return "", errors.New("ldap base DN is required")
	}

	filter, err := buildLDAPUserFilter(cfg.LDAPGetUsers, cfg.LDAPGetUsersFilter, username)
	if err != nil {
		return "", err
	}

	req := ldap.NewSearchRequest(
		searchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		2,
		5,
		false,
		filter,
		[]string{"dn"},
		nil,
	)

	result, err := conn.Search(req)
	if err != nil {
		return "", err
	}
	if len(result.Entries) == 0 {
		return "", errors.New("invalid username or password")
	}
	if len(result.Entries) > 1 {
		return "", errors.New("LDAP 用户查询结果不唯一")
	}
	return result.Entries[0].DN, nil
}

func buildLDAPUserFilter(queryTemplate, extraFilter, username string) (string, error) {
	escapedUser := ldap.EscapeFilter(strings.TrimSpace(username))
	loginAttr := defaultLDAPLoginAttr
	primaryTemplate := strings.TrimSpace(queryTemplate)

	switch {
	case primaryTemplate == "":
		primaryTemplate = fmt.Sprintf("(%s=%%{user})", loginAttr)
	case looksLikeLDAPAttributeName(primaryTemplate):
		loginAttr = primaryTemplate
		primaryTemplate = fmt.Sprintf("(%s=%%{user})", loginAttr)
	}

	primary, err := expandLDAPFilter(primaryTemplate, loginAttr, escapedUser)
	if err != nil {
		return "", err
	}

	extra := strings.TrimSpace(extraFilter)
	if extra == "" {
		return primary, nil
	}

	extraExpanded, err := expandLDAPFilter(extra, loginAttr, escapedUser)
	if err != nil {
		return "", err
	}
	return "(&" + primary + extraExpanded + ")", nil
}

func expandLDAPFilter(raw, loginAttr, escapedUser string) (string, error) {
	filter := strings.TrimSpace(raw)
	if filter == "" {
		return "", errors.New("ldap filter is required")
	}

	attrName := sanitizeLDAPAttributeName(loginAttr)
	if attrName == "" {
		attrName = defaultLDAPLoginAttr
	}

	filter = strings.ReplaceAll(filter, "%{attr}", attrName)
	filter = strings.ReplaceAll(filter, "%{user}", escapedUser)
	if !strings.HasPrefix(filter, "(") {
		filter = "(" + filter + ")"
	}
	return filter, nil
}

func looksLikeLDAPAttributeName(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false
	}
	if strings.ContainsAny(value, "()=%{}&|!<>~ ") {
		return false
	}
	return sanitizeLDAPAttributeName(value) == value
}

func sanitizeLDAPAttributeName(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeLDAPServerURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "ldap://") || strings.HasPrefix(lower, "ldaps://") {
		return value
	}
	return "ldap://" + value
}

func isLDAPInvalidCredentialsError(err error) bool {
	if err == nil {
		return false
	}

	var ldapErr *ldap.Error
	if errors.As(err, &ldapErr) && ldapErr.ResultCode == ldap.LDAPResultInvalidCredentials {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "invalid username or password") ||
		strings.Contains(message, "invalid credentials") ||
		strings.Contains(message, "ldap result code 49")
}
