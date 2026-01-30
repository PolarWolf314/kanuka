package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// Formatter applies semantic formatting to text.
type Formatter struct {
	color  *color.Color
	prefix string
	suffix string
}

// Sprint formats the arguments and returns the resulting string.
func (f Formatter) Sprint(a ...interface{}) string {
	text := fmt.Sprint(a...)
	if noColor() {
		return f.prefix + text + f.suffix
	}
	return f.color.Sprint(text)
}

// Sprintf formats according to a format specifier and returns the resulting string.
func (f Formatter) Sprintf(format string, a ...interface{}) string {
	text := fmt.Sprintf(format, a...)
	if noColor() {
		return f.prefix + text + f.suffix
	}
	return f.color.Sprint(text)
}

// EnsureNewline ensures the string ends with a newline character.
func EnsureNewline(s string) string {
	if len(s) == 0 || s[len(s)-1] != '\n' {
		return s + "\n"
	}
	return s
}

// noColor returns true if color output should be disabled.
func noColor() bool {
	// Check NO_COLOR environment variable (https://no-color.org/).
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		return true
	}
	// Also respect fatih/color's detection (terminal capability, TERM=dumb, etc.).
	return color.NoColor
}

// Semantic formatters for different types of CLI output.
var (
	// Code formats runnable commands or code snippets.
	// Yellow with color, `backticks` without.
	Code = Formatter{color.New(color.FgYellow), "`", "`"}

	// Path formats file or directory paths.
	// Yellow with color, no decoration without (paths are self-evident).
	Path = Formatter{color.New(color.FgYellow), "", ""}

	// Flag formats CLI flags like --verbose or --dry-run.
	// Yellow with color, no decoration without (-- prefix is sufficient).
	Flag = Formatter{color.New(color.FgYellow), "", ""}

	// Success formats success indicators and messages.
	// Green with color, unchanged without.
	Success = Formatter{color.New(color.FgGreen), "", ""}

	// Error formats error indicators and messages.
	// Red with color, unchanged without.
	Error = Formatter{color.New(color.FgRed), "", ""}

	// Warning formats warning indicators and messages.
	// Yellow with color, unchanged without.
	Warning = Formatter{color.New(color.FgYellow), "", ""}

	// Info formats informational hints, tips, and directional indicators.
	// Cyan with color, unchanged without.
	Info = Formatter{color.New(color.FgCyan), "", ""}

	// Highlight formats emphasized user values like emails, project names, device names.
	// Cyan with color, 'single quotes' without.
	Highlight = Formatter{color.New(color.FgCyan), "'", "'"}

	// Muted formats de-emphasized or secondary text.
	// Gray with color, (parentheses) without.
	Muted = Formatter{color.New(color.FgHiBlack), "(", ")"}
)
