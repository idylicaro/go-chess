package render

import (
	"bytes"
	"testing"
)

func env(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

func TestDetectStyleUnicode(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want bool
	}{
		{"utf-8 LANG", map[string]string{"LANG": "en_US.UTF-8"}, true},
		{"utf8 spelling", map[string]string{"LANG": "pt_BR.utf8"}, true},
		{"LC_ALL wins over LANG", map[string]string{"LC_ALL": "C", "LANG": "en_US.UTF-8"}, false},
		{"LC_CTYPE wins over LANG", map[string]string{"LC_CTYPE": "en_US.UTF-8", "LANG": "C"}, true},
		{"C locale", map[string]string{"LANG": "C"}, false},
		{"no locale", map[string]string{}, false},
		{"dumb terminal", map[string]string{"LANG": "en_US.UTF-8", "TERM": "dumb"}, false},
		{"linux console", map[string]string{"LANG": "en_US.UTF-8", "TERM": "linux"}, false},
	}
	for _, tt := range tests {
		if got := DetectStyle(&bytes.Buffer{}, env(tt.vars)).Unicode; got != tt.want {
			t.Errorf("%s: Unicode = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestDetectStyleColor(t *testing.T) {
	// A buffer is not a terminal, so color needs CLICOLOR_FORCE here.
	if DetectStyle(&bytes.Buffer{}, env(nil)).Color {
		t.Error("color on for a non-terminal")
	}
	if !DetectStyle(&bytes.Buffer{}, env(map[string]string{"CLICOLOR_FORCE": "1"})).Color {
		t.Error("CLICOLOR_FORCE should turn color on")
	}
	if DetectStyle(&bytes.Buffer{}, env(map[string]string{"CLICOLOR_FORCE": "0"})).Color {
		t.Error("CLICOLOR_FORCE=0 should not turn color on")
	}
	if DetectStyle(&bytes.Buffer{}, env(map[string]string{"CLICOLOR_FORCE": "1", "NO_COLOR": "1"})).Color {
		t.Error("NO_COLOR must win over CLICOLOR_FORCE")
	}
	if DetectStyle(&bytes.Buffer{}, env(map[string]string{"CLICOLOR_FORCE": "1", "TERM": "dumb"})).Color {
		t.Error("a dumb terminal has no color")
	}
}
