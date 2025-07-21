package grove

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContainerProfile represents a container configuration profile
type ContainerProfile struct {
	Name            string   `toml:"name"`
	IncludeDevTools bool     `toml:"include_dev_tools"`
	ExcludePackages []string `toml:"exclude_packages,omitempty"`
	ExposePorts     []string `toml:"expose_ports,omitempty"`
}

// DoesContainerConfigExist checks if container configuration already exists in devenv.nix
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

	// Check if container configuration already exists
	return strings.Contains(string(content), "containers."), nil
}

// AddContainerConfigToDevenvNix adds container configuration to devenv.nix
func AddContainerConfigToDevenvNix() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvNixPath := filepath.Join(currentDir, "devenv.nix")
	content, err := os.ReadFile(devenvNixPath)
	if err != nil {
		return fmt.Errorf("failed to read devenv.nix: %w", err)
	}

	// Get project name from directory
	projectName := filepath.Base(currentDir)

	// Container configuration to add
	containerConfig := fmt.Sprintf(`
  # Kanuka container configuration
  containers.%s = {
    name = "grove-%s";
    startupCommand = "bash";
    isRootContainer = false;
  };`, projectName, projectName)

	// Find the closing brace and insert container config before it
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")
	
	// Find the last closing brace (should be the end of the main configuration)
	var insertIndex int
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "}" {
			insertIndex = i
			break
		}
	}

	// Insert container configuration before the closing brace
	newLines := make([]string, 0, len(lines)+len(strings.Split(containerConfig, "\n")))
	newLines = append(newLines, lines[:insertIndex]...)
	newLines = append(newLines, strings.Split(containerConfig, "\n")...)
	newLines = append(newLines, lines[insertIndex:]...)

	// Write the modified content back
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(devenvNixPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write modified devenv.nix: %w", err)
	}

	return nil
}

// AddContainerProfilesToKanukaToml adds container profiles to kanuka.toml
func AddContainerProfilesToKanukaToml() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	kanukaTomlPath := filepath.Join(currentDir, "kanuka.toml")
	content, err := os.ReadFile(kanukaTomlPath)
	if err != nil {
		return fmt.Errorf("failed to read kanuka.toml: %w", err)
	}

	// Check if container configuration already exists
	if strings.Contains(string(content), "[grove.containers]") {
		return nil // Already exists, nothing to do
	}

	// Container profiles configuration to add
	containerProfiles := `
[grove.containers]
default_profile = "default"

[grove.containers.profiles.default]
name = "development"
include_dev_tools = true
expose_ports = ["3000", "8080"]

[grove.containers.profiles.minimal]
name = "production"
include_dev_tools = false
exclude_packages = ["git", "vim", "curl"]
`

	// Append container configuration to the file
	newContent := string(content) + containerProfiles
	if err := os.WriteFile(kanukaTomlPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write modified kanuka.toml: %w", err)
	}

	return nil
}