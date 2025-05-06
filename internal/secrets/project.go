package secrets

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetProjectName returns the name of the current project (directory).
func GetProjectName() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	projectName := filepath.Base(workingDirectory)
	return projectName, nil
}
