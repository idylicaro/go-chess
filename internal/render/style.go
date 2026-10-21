// Package render draws chess positions as text: the board with colors and
// Unicode pieces when the terminal supports them and plain ASCII when it
// does not. It returns strings, so the CLI prints them and the TUI embeds
// them in its layout.
package render

import (
	"io"
	"os"
	"strings"
)

// Style says what the terminal can show.
type Style struct {
	Color   bool // ANSI colors for the squares and pieces
	Unicode bool // chess symbols (♞) instead of letters (n)
}

// DetectStyle looks at w and the environment. Colors need a terminal and
// are off for NO_COLOR and TERM=dumb (CLICOLOR_FORCE turns them on without a
// terminal, unless NO_COLOR is set). Unicode needs a UTF-8 locale and a
// terminal that is not the Linux console or dumb.
//
//	style := render.DetectStyle(os.Stdout, os.Getenv)
func DetectStyle(w io.Writer, getenv func(string) string) Style {
	term := getenv("TERM")
	forced := getenv("CLICOLOR_FORCE") != "" && getenv("CLICOLOR_FORCE") != "0"
	return Style{
		Color:   getenv("NO_COLOR") == "" && term != "dumb" && (isTerminal(w) || forced),
		Unicode: term != "dumb" && term != "linux" && utf8Locale(getenv),
	}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// utf8Locale reports whether the first locale variable that is set names UTF-8.
func utf8Locale(getenv func(string) string) bool {
	for _, name := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if value := getenv(name); value != "" {
			value = strings.ToLower(value)
			return strings.Contains(value, "utf-8") || strings.Contains(value, "utf8")
		}
	}
	return false
}
