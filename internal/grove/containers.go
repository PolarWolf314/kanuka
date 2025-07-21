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

// GetContainerProfile retrieves a specific container profile from kanuka.toml
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

// ApplyContainerProfile temporarily modifies devenv.nix to apply profile settings
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
		os.WriteFile(devenvNixPath, originalContent, 0600)
	}

	return cleanup, nil
}

// applyProfileToDevenvNix applies profile settings to devenv.nix content
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

// isDevToolPackage checks if a line contains a development tool package
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

// isExcludedPackage checks if a line contains a package that should be excluded
func isExcludedPackage(line string, excludeList []string) bool {
	line = strings.TrimSpace(line)
	
	for _, excluded := range excludeList {
		if strings.Contains(line, "pkgs."+excluded) && !strings.HasPrefix(line, "#") {
			return true
		}
	}
	
	return false
}

// GetContainerNameFromDevenvNix extracts the container name from devenv.nix
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

	// Look for container configuration like: containers.myproject = {
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "containers.") && strings.Contains(line, "= {") {
			// Extract container name from "containers.name = {"
			start := strings.Index(line, "containers.") + 11
			end := strings.Index(line[start:], " =")
			if end > 0 {
				containerName := line[start : start+end]
				return containerName, nil
			}
		}
	}

	return "", fmt.Errorf("no container configuration found in devenv.nix")
}

// ApplyContainerProfileAndName temporarily modifies devenv.nix to apply profile settings and custom name
func ApplyContainerProfileAndName(profile *ContainerProfile, containerName string) (func(), error) {
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

	// Apply profile modifications and update container name
	modifiedContent, err := applyProfileAndNameToDevenvNix(string(originalContent), profile, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to apply profile and name modifications: %w", err)
	}

	// Write modified content
	if err := os.WriteFile(devenvNixPath, []byte(modifiedContent), 0600); err != nil {
		return nil, fmt.Errorf("failed to write modified devenv.nix: %w", err)
	}

	// Return cleanup function to restore original content
	cleanup := func() {
		os.WriteFile(devenvNixPath, originalContent, 0600)
	}

	return cleanup, nil
}

// applyProfileAndNameToDevenvNix applies profile settings and updates container name in devenv.nix content
func applyProfileAndNameToDevenvNix(content string, profile *ContainerProfile, containerName string) (string, error) {
	lines := strings.Split(content, "\n")
	var result []string
	
	for _, line := range lines {
		// Update container name if this is the container configuration line
		if strings.Contains(line, "containers.") && strings.Contains(line, "= {") {
			// Replace the container name with the custom one
			// From: containers.oldname = {
			// To:   containers.newname = {
			if containerName != "" {
				line = fmt.Sprintf("  containers.%s = {", containerName)
			}
		}
		
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