package grove

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DoesContainerConfigExist checks if container configuration already exists.
func DoesContainerConfigExist() (bool, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvNixPath := filepath.Join(currentDir, "devenv.nix")
	content, err := os.ReadFile(devenvNixPath)
	if err != nil {
		return false, fmt.Errorf("failed to read devenv.nix: %w", err)
	}

	devenvYamlPath := filepath.Join(currentDir, "devenv.yaml")
	yamlContent, err := os.ReadFile(devenvYamlPath)
	if err != nil {
		return false, fmt.Errorf("failed to read devenv.yaml: %w", err)
	}

	// Container support exists if:
	// 1. devenv.nix has a name field
	// 2. devenv.yaml has nix2container input
	hasName := strings.Contains(string(content), "name = ")
	hasNix2Container := strings.Contains(string(yamlContent), "nix2container:")

	return hasName && hasNix2Container, nil
}

// AddContainerConfigToDevenvNix adds container configuration to devenv.nix.
// Container support is automatically enabled when devenv.nix has a name field
// and devenv.yaml has nix2container input - no additional configuration needed.
func AddContainerConfigToDevenvNix() error {
	return nil
}

// GetContainerNameFromDevenvNix extracts the container name from devenv.nix.
func GetContainerNameFromDevenvNix() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvNixPath := filepath.Join(currentDir, "devenv.nix")
	content, err := os.ReadFile(devenvNixPath)
	if err != nil {
		return "", fmt.Errorf("failed to read devenv.nix: %w", err)
	}

	// Look for name field like: name = "project-name";
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name = ") && strings.Contains(line, "\"") {
			// Extract name from 'name = "project-name";'
			start := strings.Index(line, "\"") + 1
			end := strings.LastIndex(line, "\"")
			if end > start {
				containerName := line[start:end]
				return containerName, nil
			}
		}
	}

	return "", fmt.Errorf("no name field found in devenv.nix")
}
