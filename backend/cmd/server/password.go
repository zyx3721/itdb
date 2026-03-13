package server

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func hashPassword(plain string) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return "", nil
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func isBcryptHash(stored string) bool {
	s := strings.TrimSpace(stored)
	return strings.HasPrefix(s, "$2a$") || strings.HasPrefix(s, "$2b$") || strings.HasPrefix(s, "$2y$")
}

func verifyPassword(stored, plain string) (ok bool, legacyPlaintext bool) {
	if strings.TrimSpace(stored) == "" {
		return false, false
	}
	if isBcryptHash(stored) {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(plain)) == nil, false
	}
	return stored == plain, true
}

func (a *App) upgradeLegacyUserPassword(ctx context.Context, userID int64, plain string) {
	if userID <= 0 || strings.TrimSpace(plain) == "" {
		return
	}
	hashed, err := hashPassword(plain)
	if err != nil || hashed == "" {
		return
	}
	_, _ = a.db.ExecContext(ctx, `UPDATE users SET pass = ? WHERE id = ?`, hashed, userID)
}
