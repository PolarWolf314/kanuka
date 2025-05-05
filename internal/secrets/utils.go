package secrets

import (
	"os/user"
	"strings"
)

// GetUsername returns the current username
func GetUsername() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

// FormatPaths formats a slice of paths into a readable string
func FormatPaths(paths []string) string {
	var b strings.Builder
	b.WriteString("\n")
	for _, path := range paths {
		b.WriteString("    - ")
		b.WriteString(path)
		b.WriteString("\n")
	}
	return b.String()
}
