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
	// If the project doesn't exist, then the project name doesn't exist either
	// but don't throw an error because it will cause a crash when a non-init command
	// is run on a repo that hasn't been intialised
	if projectRoot == "" {
		return "", nil
	}
	projectName := filepath.Base(projectRoot)
	return projectName, nil
}
