package utils

import (
	"strings"

	"github.com/fatih/color"
)

// FormatPaths formats a slice of paths into a readable string.
func FormatPaths(paths []string) string {
	var b strings.Builder
	b.WriteString("\n")
	for _, path := range paths {
		b.WriteString("    - ")
		b.WriteString(color.YellowString(path))
		b.WriteString("\n")
	}
	return b.String()
}
