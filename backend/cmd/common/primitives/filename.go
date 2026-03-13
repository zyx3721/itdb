package primitives

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mozillazg/go-pinyin"
)

var (
	englishFilenameSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	chineseFilenameSegmentPattern = regexp.MustCompile(`^\p{Han}+$`)
)

func BuildFilenameSegment(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "file"
	}

	compact := strings.Join(strings.Fields(trimmed), "")
	if compact == "" {
		return "file"
	}

	if englishFilenameSegmentPattern.MatchString(compact) {
		return SanitizeFilename(compact)
	}

	if chineseFilenameSegmentPattern.MatchString(compact) {
		args := pinyin.NewArgs()
		args.Style = pinyin.FirstLetter
		parts := pinyin.Pinyin(compact, args)
		var builder strings.Builder
		for _, part := range parts {
			if len(part) == 0 {
				continue
			}
			builder.WriteString(strings.ToLower(part[0]))
		}
		segment := SanitizeFilename(builder.String())
		if segment != "" {
			return segment
		}
	}

	return "file"
}

func ContentDispositionFallbackName(displayName string) string {
	trimmed := strings.TrimSpace(displayName)
	if trimmed == "" {
		return "file"
	}

	ext := strings.ToLower(filepath.Ext(trimmed))
	base := strings.TrimSpace(strings.TrimSuffix(trimmed, ext))
	base = SanitizeFilename(base)
	if base == "" {
		base = "file"
	}
	return base + ext
}
