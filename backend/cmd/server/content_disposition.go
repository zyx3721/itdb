package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"itdb-backend/cmd/common/primitives"
)

func setContentDispositionHeader(w http.ResponseWriter, disposition, displayName string) {
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = "file"
	}

	fallback := primitives.ContentDispositionFallbackName(name)
	escaped := url.PathEscape(name)
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`, disposition, fallback, escaped),
	)
}
