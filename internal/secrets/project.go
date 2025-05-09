package secrets

import (
	"fmt"
	"path/filepath"
)

// GetProjectName returns the name of the current project (directory).
func GetProjectName() (string, error) {
	projectDirectory, err := FindProjectKanukaRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get project directory: %w", err)
	}
	projectName := filepath.Base(projectDirectory)
	return projectName, nil
}
