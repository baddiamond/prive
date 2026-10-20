package platform

import "strings"

// SystemLocale returns the user's preferred language as a BCP 47 tag
// (for example "es-ES" or "en"), or an empty string if it cannot be determined.
// The frontend maps it to one of the languages Privé ships.
func SystemLocale() string {
	return NormalizeLocale(systemLocale())
}

// NormalizeLocale converts OS-specific locale spellings into a BCP 47 tag.
// It accepts forms such as "es_ES.UTF-8", "pt_BR@euro", "en-US" or "zh-Hans-CN"
// and returns "" for empty or neutral locales ("C", "POSIX").
func NormalizeLocale(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, `"`)
	// Drop the encoding (".UTF-8") and the modifier ("@euro").
	if i := strings.IndexAny(s, ".@"); i >= 0 {
		s = s[:i]
	}
	if s == "" || strings.EqualFold(s, "C") || strings.EqualFold(s, "POSIX") {
		return ""
	}

	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' })
	if len(parts) == 0 || !isAlpha(parts[0]) || len(parts[0]) < 2 || len(parts[0]) > 3 {
		return ""
	}
	parts[0] = strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		p := parts[i]
		switch {
		case len(p) == 2 && isAlpha(p):
			parts[i] = strings.ToUpper(p) // region: "ES"
		case len(p) == 4 && isAlpha(p):
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:]) // script: "Hans"
		}
	}
	return strings.Join(parts, "-")
}

// localeFromEnv reads the POSIX locale variables in priority order.
func localeFromEnv(getenv func(string) string) string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := NormalizeLocale(getenv(key)); v != "" {
			return v
		}
	}
	// LANGUAGE is a colon-separated preference list, e.g. "es:en".
	if first, _, _ := strings.Cut(getenv("LANGUAGE"), ":"); first != "" {
		return NormalizeLocale(first)
	}
	return ""
}

// parseAppleLanguages extracts the first language from the plist array printed by
// `defaults read -g AppleLanguages`, e.g. `(\n    "es-ES",\n    "en-US"\n)`.
func parseAppleLanguages(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.Trim(strings.TrimSpace(line), `",`)
		if line != "" && line != "(" && line != ")" {
			return line
		}
	}
	return ""
}

func isAlpha(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return s != ""
}
