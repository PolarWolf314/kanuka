package utils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// FindProjectKanukaRoot traverses up directories to find the project's Kanuka root.
// Returns the path to the project root if found, empty string otherwise.
// Stops searching when it reaches the user's home directory.
func FindProjectKanukaRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	for {
		// Stop searching at one level above home directory
		if currentDir == path.Join(homeDir, "..") {
			return "", nil
		}

		kanukaDir := filepath.Join(currentDir, ".kanuka")
		fileInfo, err := os.Stat(kanukaDir)
		// No error means the path exists
		if err == nil {
			if fileInfo.IsDir() {
				return currentDir, nil
			}
		} else if !os.IsNotExist(err) {
			// Return any error that's not "file not found" (like permission issues)
			return "", fmt.Errorf("error checking for .kanuka directory at %s: %w", currentDir, err)
		}

		parentDir := filepath.Dir(currentDir)

		// If we've reached the filesystem root and haven't found .kanuka
		if parentDir == currentDir {
			return "", nil
		}
		currentDir = parentDir
	}
}
