package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadParsesActiveTheme verifies the parser against the real Omarchy
// colors.toml when present; otherwise it confirms the Default fallback.
func TestLoadParsesActiveTheme(t *testing.T) {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml")
	got := Load()

	if _, err := os.Stat(path); err != nil {
		if got != Default() {
			t.Fatalf("no theme file but Load() != Default(): %+v", got)
		}
		t.Skip("no active Omarchy theme on this machine; Default() used")
	}

	for _, c := range []string{got.Accent, got.Fg, got.Bg, got.Good, got.Bad} {
		if !strings.HasPrefix(c, "#") || len(c) != 7 {
			t.Errorf("bad colour %q", c)
		}
	}
	t.Logf("loaded theme: %+v", got)
}
