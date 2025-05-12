package utils

import (
	"fmt"
	"path/filepath"
)

// GetProjectName returns the name of the current project (directory).
func GetProjectName() (string, error) {
	projectRoot, err := FindProjectKanukaRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get project directory: %w", err)
	}
	if projectRoot == "" {
		return "", fmt.Errorf("failed to find project root because it doesn't exist")
	}
	projectName := filepath.Base(projectRoot)
	return projectName, nil
}
