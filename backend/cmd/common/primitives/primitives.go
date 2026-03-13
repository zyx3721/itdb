package primitives

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var invalidFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func ParseDateInput(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}

	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return n, nil
	}

	if len(raw) == 4 {
		year, err := strconv.Atoi(raw)
		if err == nil {
			return time.Date(year, 1, 1, 0, 0, 0, 0, time.Local).Unix(), nil
		}
	}

	normalized := strings.NewReplacer(
		"年", "-",
		"月", "-",
		"日", "",
	).Replace(raw)
	normalized = strings.TrimSpace(normalized)
	if m := regexp.MustCompile(`^(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})$`).FindStringSubmatch(normalized); len(m) == 4 {
		year, errYear := strconv.Atoi(m[1])
		month, errMonth := strconv.Atoi(m[2])
		day, errDay := strconv.Atoi(m[3])
		if errYear == nil && errMonth == nil && errDay == nil {
			t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
			if t.Year() == year && int(t.Month()) == month && t.Day() == day {
				return t.Unix(), nil
			}
		}
	}

	layouts := []string{
		"2006-01-02",
		"2006-1-2",
		time.RFC3339,
		"2006/01/02",
		"2006/1/2",
		"02.01.2006",
		"2006.01.02",
		"2006.1.2",
		"02/01/2006",
		"01/02/2006",
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t.Unix(), nil
		}
		if normalized != raw {
			if t, err := time.ParseInLocation(layout, normalized, time.Local); err == nil {
				return t.Unix(), nil
			}
		}
	}

	for _, layout := range []string{"2006-01", "2006-1", "2006/01", "2006/1", "2006.01", "2006.1"} {
		if t, err := time.ParseInLocation(layout, normalized, time.Local); err == nil {
			return t.Unix(), nil
		}
	}

	return 0, fmt.Errorf("unsupported date format: %s", raw)
}

func IntParam(v string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
}

func IntParamDefault(v string, d int64) int64 {
	if strings.TrimSpace(v) == "" {
		return d
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil || n < 0 {
		return d
	}
	return n
}

func ListLimitParam(v string, d int64) int64 {
	s := strings.TrimSpace(v)
	if s == "" {
		return d
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return d
	}
	if n == -1 {
		return -1
	}
	if n < 0 {
		return d
	}
	return n
}

func NullableInt(v *int64) interface{} {
	if v == nil || *v == 0 {
		return nil
	}
	return *v
}

func NullableInt64Value(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

func EqualNullInt64(dbValue sql.NullInt64, req *int64) bool {
	if !dbValue.Valid && (req == nil || *req == 0) {
		return true
	}
	if !dbValue.Valid || req == nil {
		return false
	}
	return dbValue.Int64 == *req
}

func SameDay(ts1, ts2 int64) bool {
	if ts1 == 0 || ts2 == 0 {
		return false
	}
	t1 := time.Unix(ts1, 0)
	t2 := time.Unix(ts2, 0)
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func ParseIDCSV(raw string) []int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []int64{}
	}
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseInt(p, 10, 64)
		if err == nil && v > 0 {
			ids = append(ids, v)
		}
	}
	return ids
}

func SanitizeFilename(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = invalidFilenameChars.ReplaceAllString(s, "")
	if s == "" {
		s = "file"
	}
	return s
}

func ShortUUID() string {
	hex := fmt.Sprintf("%x", time.Now().UnixNano())
	if len(hex) <= 4 {
		return hex
	}
	return hex[len(hex)-4:]
}

func AsString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func AsInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		n, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprintf("%v", v)), 10, 64)
		return n
	}
}
