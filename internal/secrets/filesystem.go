package secrets

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// EnsureUserSettings ensures that the user's Kanuka settings directory exists.
func EnsureUserSettings() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	userKanukaDirectory := filepath.Join(currentUser.HomeDir, ".config", ".kanuka", "keys")

	if err := os.MkdirAll(userKanukaDirectory, 0700); err != nil {
		return fmt.Errorf("failed to create %s: %w", userKanukaDirectory, err)
	}

	return nil
}

// DoesProjectKanukaSettingsExist checks if the project's Kanuka settings directory exists.
func DoesProjectKanukaSettingsExist() (bool, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("failed to get working directory: %w", err)
	}

	projectKanukaDirectory := filepath.Join(workingDirectory, ".kanuka")

	fileInfo, err := os.Stat(projectKanukaDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, but this isn't an error condition
			// for this function - it's an expected possible outcome
			return false, nil
		}
		// Some other error occurred (permissions, etc.)
		return false, fmt.Errorf("failed to check if project Kanuka directory exists: %w", err)
	}

	// Make sure it's a directory
	if !fileInfo.IsDir() {
		return false, fmt.Errorf(".kanuka exists but is not a directory")
	}

	// Directory exists
	return true, nil
}

// EnsureKanukaSettings ensures that the project's Kanuka settings directories exist.
func EnsureKanukaSettings() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	kanukaDir := filepath.Join(wd, ".kanuka")
	secretsDir := filepath.Join(kanukaDir, "secrets")
	publicKeysDir := filepath.Join(kanukaDir, "public_keys")

	if _, err := os.Stat(kanukaDir); os.IsNotExist(err) {
		if err := os.MkdirAll(secretsDir, 0755); err != nil {
			return fmt.Errorf("failed to create .kanuka/secrets: %w", err)
		}
		if err := os.MkdirAll(publicKeysDir, 0755); err != nil {
			return fmt.Errorf("failed to create .kanuka/public_keys: %w", err)
		}

	} else if err != nil {
		// Handle other potential errors from os.Stat
		return fmt.Errorf("failed to check if .kanuka directory exists: %w", err)
	}

	return nil
}

// FindEnvOrKanukaFiles finds .env or .kanuka files in the project directory.
func FindEnvOrKanukaFiles(rootDir string, ignoreDirs []string, isKanuka bool) ([]string, error) {
	var result []string

	ignoreMap := make(map[string]bool)
	for _, dir := range ignoreDirs {
		ignoreMap[dir] = true
	}

	// Always ignore searching for .env files in .kanuka/
	ignoreMap[".kanuka"] = true

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed while walking directory: %w", err)
		}

		// Skip ignored directories
		if d.IsDir() {
			if ignoreMap[filepath.Base(path)] {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip irregular files such as sockets, pipes, devices, etc
		if !d.Type().IsRegular() {
			return nil
		}

		if isKanuka {
			if strings.Contains(filepath.Base(path), ".env") && strings.Contains(path, ".kanuka") {
				result = append(result, path)
			}
		} else {
			// Check if the filename contains ".env"
			if strings.Contains(filepath.Base(path), ".env") && !strings.Contains(path, ".kanuka") {
				result = append(result, path)
			}
		}

		return nil
	})

	return result, err
}
