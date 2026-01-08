package utils

import (
	"os"
	"os/user"
	"regexp"
	"strconv"
	"strings"
)

// GetUsername returns the current username.
func GetUsername() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

// GetHostname returns the system hostname.
func GetHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return hostname, nil
}

// SanitizeDeviceName sanitizes a device name by removing special characters and converting spaces to hyphens.
func SanitizeDeviceName(name string) string {
	// Trim whitespace.
	name = strings.TrimSpace(name)

	// Convert to lowercase.
	name = strings.ToLower(name)

	// Replace spaces with hyphens.
	name = strings.ReplaceAll(name, " ", "-")

	// Remove any characters that are not alphanumeric, hyphens, or underscores.
	re := regexp.MustCompile(`[^a-z0-9\-_]`)
	name = re.ReplaceAllString(name, "")

	// Remove consecutive hyphens.
	re = regexp.MustCompile(`-+`)
	name = re.ReplaceAllString(name, "-")

	// Trim leading and trailing hyphens.
	name = strings.Trim(name, "-")

	// If empty after sanitization, use a default.
	if name == "" {
		name = "device"
	}

	return name
}

// GenerateDeviceName generates a device name based on the system hostname.
// It sanitizes the hostname and checks for conflicts with existing device names.
// If a conflict is found, it appends a number suffix (-2, -3, etc.).
func GenerateDeviceName(existingDeviceNames []string) (string, error) {
	hostname, err := GetHostname()
	if err != nil {
		// Fallback to username if hostname is unavailable.
		username, userErr := GetUsername()
		if userErr != nil {
			hostname = "device"
		} else {
			hostname = username
		}
	}

	baseName := SanitizeDeviceName(hostname)
	deviceName := baseName

	// Check for conflicts and append suffix if needed.
	existingSet := make(map[string]bool)
	for _, name := range existingDeviceNames {
		existingSet[strings.ToLower(name)] = true
	}

	suffix := 2
	for existingSet[strings.ToLower(deviceName)] {
		deviceName = baseName + "-" + strconv.Itoa(suffix)
		suffix++
	}

	return deviceName, nil
}
