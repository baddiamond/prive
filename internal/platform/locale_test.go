package platform

import "testing"

func TestNormalizeLocale(t *testing.T) {
	cases := map[string]string{
		"":              "",
		"C":             "",
		"POSIX":         "",
		"C.UTF-8":       "",
		"es_ES.UTF-8":   "es-ES",
		"pt_BR@euro":    "pt-BR",
		"en-us":         "en-US",
		"EN":            "en",
		"zh-hans-cn":    "zh-Hans-CN",
		`"es-419"`:      "es-419",
		"  fr_CA  ":     "fr-CA",
		"1234":          "",
		"toolonglocale": "",
	}
	for in, want := range cases {
		if got := NormalizeLocale(in); got != want {
			t.Errorf("NormalizeLocale(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLocaleFromEnv(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"none", map[string]string{}, ""},
		{"lang", map[string]string{"LANG": "es_ES.UTF-8"}, "es-ES"},
		{"lc_all wins", map[string]string{"LC_ALL": "fr_FR.UTF-8", "LANG": "es_ES.UTF-8"}, "fr-FR"},
		{"skips neutral", map[string]string{"LC_ALL": "C", "LANG": "de_DE.UTF-8"}, "de-DE"},
		{"language list", map[string]string{"LANG": "C", "LANGUAGE": "es:en"}, "es"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(k string) string { return tc.env[k] }
			if got := localeFromEnv(getenv); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseAppleLanguages(t *testing.T) {
	out := "(\n    \"es-ES\",\n    \"en-US\"\n)\n"
	if got := parseAppleLanguages(out); got != "es-ES" {
		t.Errorf("got %q, want es-ES", got)
	}
	if got := parseAppleLanguages("(\n)\n"); got != "" {
		t.Errorf("empty array: got %q, want empty", got)
	}
}
