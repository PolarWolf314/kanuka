package utils

import (
	"regexp"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/ui"
)

// emailRegex is a simple regex for validating email format.
// It checks for: local-part@domain.tld format.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// FormatPaths formats a slice of paths into a readable string.
func FormatPaths(paths []string) string {
	var b strings.Builder
	b.WriteString("\n")
	for _, path := range paths {
		b.WriteString("    - ")
		b.WriteString(ui.Path.Sprint(path))
		b.WriteString("\n")
	}
	return b.String()
}

// IsValidEmail checks if the given string is a valid email address format.
func IsValidEmail(email string) bool {
	if email == "" {
		return false
	}
	return emailRegex.MatchString(email)
}

// IsValidDeviceName checks if a device name is valid (alphanumeric, hyphens, underscores).
func IsValidDeviceName(name string) bool {
	if name == "" {
		return false
	}
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
	return validPattern.MatchString(name)
}
