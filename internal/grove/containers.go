package grove

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContainerProfile represents a container configuration profile.
type ContainerProfile struct {
	Name            string   `toml:"name"`
	IncludeDevTools bool     `toml:"include_dev_tools"`
	ExcludePackages []string `toml:"exclude_packages,omitempty"`
	ExposePorts     []string `toml:"expose_ports,omitempty"`
}

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

// AddContainerConfigToDevenvNix adds container configuration to devenv.nix
// With the simplified approach, container support is enabled by just having the name field
// and the nix2container input in devenv.yaml.
func AddContainerConfigToDevenvNix() error {
	// Container support is automatically enabled when:
	// 1. devenv.nix has a name field (already added in CreateDevenvNix)
	// 2. devenv.yaml has nix2container input (added by AddNix2ContainerInput)
	// No additional configuration needed in devenv.nix
	return nil
}

// AddContainerProfilesToKanukaToml adds container profiles to kanuka.toml.
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

// GetContainerProfile retrieves a specific container profile from kanuka.toml.
func GetContainerProfile(profileName string) (*ContainerProfile, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	kanukaTomlPath := filepath.Join(currentDir, "kanuka.toml")
	content, err := os.ReadFile(kanukaTomlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kanuka.toml: %w", err)
	}

	// Parse the TOML to find the requested profile
	lines := strings.Split(string(content), "\n")
	var profile *ContainerProfile
	var currentSection string
	var inTargetProfile bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for profile section
		if strings.HasPrefix(line, "[grove.containers.profiles.") {
			// Extract profile name
			start := strings.Index(line, "profiles.") + 9
			end := strings.Index(line[start:], "]")
			if end > 0 {
				currentSection = line[start : start+end]
				if currentSection == profileName {
					inTargetProfile = true
					profile = &ContainerProfile{}
				} else {
					inTargetProfile = false
				}
			}
		} else if inTargetProfile && profile != nil {
			// Parse profile properties
			if strings.HasPrefix(line, "name = ") {
				parts := strings.Split(line, "\"")
				if len(parts) >= 2 {
					profile.Name = parts[1]
				}
			} else if strings.HasPrefix(line, "include_dev_tools = ") {
				profile.IncludeDevTools = strings.Contains(line, "true")
			} else if strings.HasPrefix(line, "exclude_packages = [") {
				// Parse array of excluded packages
				arrayContent := strings.TrimPrefix(line, "exclude_packages = [")
				arrayContent = strings.TrimSuffix(arrayContent, "]")
				if arrayContent != "" {
					packages := strings.Split(arrayContent, ",")
					for _, pkg := range packages {
						pkg = strings.Trim(strings.TrimSpace(pkg), "\"")
						if pkg != "" {
							profile.ExcludePackages = append(profile.ExcludePackages, pkg)
						}
					}
				}
			} else if strings.HasPrefix(line, "expose_ports = [") {
				// Parse array of exposed ports
				arrayContent := strings.TrimPrefix(line, "expose_ports = [")
				arrayContent = strings.TrimSuffix(arrayContent, "]")
				if arrayContent != "" {
					ports := strings.Split(arrayContent, ",")
					for _, port := range ports {
						port = strings.Trim(strings.TrimSpace(port), "\"")
						if port != "" {
							profile.ExposePorts = append(profile.ExposePorts, port)
						}
					}
				}
			}
		}
	}

	if profile == nil {
		return nil, fmt.Errorf("container profile '%s' not found", profileName)
	}

	return profile, nil
}

// ApplyContainerProfile temporarily modifies devenv.nix to apply profile settings.
func ApplyContainerProfile(profile *ContainerProfile) (func(), error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvNixPath := filepath.Join(currentDir, "devenv.nix")

	// Read original devenv.nix
	originalContent, err := os.ReadFile(devenvNixPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read devenv.nix: %w", err)
	}

	// Apply profile modifications
	modifiedContent, err := applyProfileToDevenvNix(string(originalContent), profile)
	if err != nil {
		return nil, fmt.Errorf("failed to apply profile modifications: %w", err)
	}

	// Write modified content
	if err := os.WriteFile(devenvNixPath, []byte(modifiedContent), 0600); err != nil {
		return nil, fmt.Errorf("failed to write modified devenv.nix: %w", err)
	}

	// Return cleanup function to restore original content
	cleanup := func() {
		_ = os.WriteFile(devenvNixPath, originalContent, 0600)
	}

	return cleanup, nil
}

// applyProfileToDevenvNix applies profile settings to devenv.nix content.
func applyProfileToDevenvNix(content string, profile *ContainerProfile) (string, error) {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// If this profile excludes dev tools, filter out common dev packages
		if !profile.IncludeDevTools && isDevToolPackage(line) {
			// Skip this line (exclude dev tools)
			continue
		}

		// If this package is in the exclude list, skip it
		if isExcludedPackage(line, profile.ExcludePackages) {
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n"), nil
}

// isDevToolPackage checks if a line contains a development tool package.
func isDevToolPackage(line string) bool {
	devTools := []string{"git", "vim", "curl", "wget", "htop", "tree", "jq"}
	line = strings.TrimSpace(line)

	for _, tool := range devTools {
		if strings.Contains(line, "pkgs."+tool) && !strings.HasPrefix(line, "#") {
			return true
		}
	}

	return false
}

// isExcludedPackage checks if a line contains a package that should be excluded.
func isExcludedPackage(line string, excludeList []string) bool {
	line = strings.TrimSpace(line)

	for _, excluded := range excludeList {
		if strings.Contains(line, "pkgs."+excluded) && !strings.HasPrefix(line, "#") {
			return true
		}
	}

	return false
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

// Note: ApplyContainerProfileAndName was removed because devenv containers
// use the 'name' field directly from devenv.nix, not separate container configurations.
