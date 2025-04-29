package secrets

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

func EnsureUserSettings() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	userKanukaDirectory := filepath.Join(currentUser.HomeDir, ".kanuka", "keys")

	if err := os.MkdirAll(userKanukaDirectory, 0700); err != nil {
		return fmt.Errorf("failed to create %s: %w", userKanukaDirectory, err)
	}

	return nil
}

func GetProjectName() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	projectName := filepath.Base(workingDirectory)
	return projectName, nil
}

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

func CopyUserPublicKeyToProject() error {
	username, err := GetUsername()
	if err != nil {
		return fmt.Errorf("failed to get username: %w", err)
	}

	projectName, err := GetProjectName()
	if err != nil {
		return fmt.Errorf("failed to get project name: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Source path: ~/.kanuka/keys/{project_name}.pub
	sourceKeyPath := filepath.Join(homeDir, ".kanuka", "keys", projectName+".pub")

	// Check if source key exists
	if _, err := os.Stat(sourceKeyPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("public key for project %s not found at %s", projectName, sourceKeyPath)
		}
		return fmt.Errorf("failed to check for source key: %w", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Destination directory: {project_path}/.kanuka/public_keys/{username}.pub
	destKeyPath := filepath.Join(workingDir, ".kanuka", "public_keys", username+".pub")

	keyData, err := os.ReadFile(sourceKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read source key file: %w", err)
	}

	// Write to destination file
	if err := os.WriteFile(destKeyPath, keyData, 0600); err != nil {
		return fmt.Errorf("failed to write key to project: %w", err)
	}

	return nil
}

func GetUsername() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

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
