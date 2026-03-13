package statustypes

import "strings"

const (
	statusTypeInUse    = "使用中"
	statusTypeInStock  = "库存"
	statusTypeFaulty   = "有故障"
	statusTypeScrapped = "报废"
)

var fixedStatusTypeColors = map[string]string{
	statusTypeInUse:    "#2f7fba",
	statusTypeInStock:  "#16a34a",
	statusTypeFaulty:   "#dc2626",
	statusTypeScrapped: "#9ca3af",
}

func fixedStatusTypeColor(desc string) (string, bool) {
	color, ok := fixedStatusTypeColors[strings.TrimSpace(desc)]
	return color, ok
}

func isProtectedStatusType(id int64, desc string) bool {
	if _, ok := fixedStatusTypeColor(desc); ok {
		return true
	}
	// 兼容旧库里内置状态通常为前四个编号（0-3）。
	return id >= 0 && id <= 3
}

func normalizeHexColor(raw string) string {
	color := strings.TrimSpace(raw)
	if color == "" {
		return ""
	}
	if len(color) != 7 || color[0] != '#' {
		return ""
	}
	for _, r := range color[1:] {
		if !isHexDigit(r) {
			return ""
		}
	}
	return strings.ToLower(color)
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func FixedStatusTypeColors() map[string]string {
	out := make(map[string]string, len(fixedStatusTypeColors))
	for k, v := range fixedStatusTypeColors {
		out[k] = v
	}
	return out
}

func FixedStatusTypeColor(desc string) (string, bool) {
	return fixedStatusTypeColor(desc)
}

func IsProtectedStatusType(id int64, desc string) bool {
	return isProtectedStatusType(id, desc)
}

func NormalizeHexColor(raw string) string {
	return normalizeHexColor(raw)
}
