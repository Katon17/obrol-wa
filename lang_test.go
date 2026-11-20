package main

import (
	"os"
	"path/filepath"
	"testing"
)

// clearLocaleEnv mengosongkan environment locale agar host test tidak bocor.
func clearLocaleEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"} {
		t.Setenv(key, "")
	}
}

func TestNormalizeLang(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"auto", "auto", true},
		{"id", "id", true},
		{"id_ID.UTF-8", "id", true},
		{"en-US", "en", true},
		{"pt_BR", "", false},
		{"C", "", false},
		{"fr_FR", "", false},
		{"", "", false},
		{"  EN_us  ", "en", true},
		{"en_US@modifier", "en", true},
		{"POSIX", "", false},
	}
	for _, tt := range tests {
		got, ok := normalizeLang(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("normalizeLang(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSystemLang(t *testing.T) {
	t.Run("LANGUAGE wins over others", func(t *testing.T) {
		clearLocaleEnv(t)
		t.Setenv("LANGUAGE", "fr_FR:en_US.UTF-8")
		t.Setenv("LC_ALL", "id_ID.UTF-8")
		t.Setenv("LANG", "id_ID.UTF-8")
		if got := systemLang(); got != "en" {
			t.Errorf("systemLang() = %q, want %q", got, "en")
		}
	})

	t.Run("LC_ALL before LANG", func(t *testing.T) {
		clearLocaleEnv(t)
		t.Setenv("LC_ALL", "id_ID.UTF-8")
		t.Setenv("LANG", "en_US.UTF-8")
		if got := systemLang(); got != "id" {
			t.Errorf("systemLang() = %q, want %q", got, "id")
		}
	})

	t.Run("LC_MESSAGES before LANG", func(t *testing.T) {
		clearLocaleEnv(t)
		t.Setenv("LC_MESSAGES", "id_ID.UTF-8")
		t.Setenv("LANG", "en_US.UTF-8")
		if got := systemLang(); got != "id" {
			t.Errorf("systemLang() = %q, want %q", got, "id")
		}
	})

	t.Run("unsupported locales fall back to en", func(t *testing.T) {
		clearLocaleEnv(t)
		t.Setenv("LANGUAGE", "fr_FR:de_DE")
		t.Setenv("LC_ALL", "fr_FR")
		t.Setenv("LC_MESSAGES", "de_DE")
		t.Setenv("LANG", "es_ES.UTF-8")
		if got := systemLang(); got != "en" {
			t.Errorf("systemLang() = %q, want %q", got, "en")
		}
	})

	t.Run("empty environment falls back to en", func(t *testing.T) {
		clearLocaleEnv(t)
		if got := systemLang(); got != "en" {
			t.Errorf("systemLang() = %q, want %q", got, "en")
		}
	})

	t.Run("C and POSIX fall back to en", func(t *testing.T) {
		clearLocaleEnv(t)
		t.Setenv("LANGUAGE", "C")
		t.Setenv("LC_ALL", "POSIX")
		t.Setenv("LC_MESSAGES", "C.UTF-8")
		t.Setenv("LANG", "POSIX")
		if got := systemLang(); got != "en" {
			t.Errorf("systemLang() = %q, want %q", got, "en")
		}
	})
}

func TestResolveLang(t *testing.T) {
	t.Run("explicit setting overrides system", func(t *testing.T) {
		clearLocaleEnv(t)
		t.Setenv("LANG", "id_ID.UTF-8")
		if got := resolveLang("en"); got != "en" {
			t.Errorf("resolveLang(%q) = %q, want %q", "en", got, "en")
		}
		if got := resolveLang("id"); got != "id" {
			t.Errorf("resolveLang(%q) = %q, want %q", "id", got, "id")
		}
	})

	t.Run("auto follows system", func(t *testing.T) {
		clearLocaleEnv(t)
		t.Setenv("LANG", "id_ID.UTF-8")
		if got := resolveLang("auto"); got != "id" {
			t.Errorf("resolveLang(%q) = %q, want %q", "auto", got, "id")
		}
	})

	t.Run("empty and unsupported follow system", func(t *testing.T) {
		clearLocaleEnv(t)
		t.Setenv("LANG", "en_US.UTF-8")
		for _, setting := range []string{"", "pt_BR", "fr_FR"} {
			if got := resolveLang(setting); got != "en" {
				t.Errorf("resolveLang(%q) = %q, want %q", setting, got, "en")
			}
		}
	})
}

func TestLoadConfigLang(t *testing.T) {
	t.Run("missing config defaults to auto", func(t *testing.T) {
		dir := t.TempDir()
		if got := loadConfig(dir).Lang; got != "auto" {
			t.Errorf("loadConfig().Lang = %q, want %q", got, "auto")
		}
	})

	t.Run("saved language is preserved", func(t *testing.T) {
		for _, want := range []string{"id", "en"} {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.json")
			if err := os.WriteFile(path, []byte(`{"lang":"`+want+`"}`), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := loadConfig(dir).Lang; got != want {
				t.Errorf("loadConfig().Lang = %q, want %q", got, want)
			}
		}
	})
}
