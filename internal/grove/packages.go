package grove

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Package represents a parsed package with version information
type Package struct {
	Name        string // Original name as provided by user
	NixName     string // Nix package name (e.g., pkgs.nodejs_18)
	DisplayName string // Display name for user feedback
	Version     string // Version if specified
}

// ParsePackageName parses a package name with optional version
// Examples: "nodejs", "nodejs_18", "typescript"
func ParsePackageName(packageName string) (*Package, error) {
	if packageName == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}

	// For now, we'll keep it simple and assume the package name is valid
	// In the future, we could add validation against nixpkgs
	pkg := &Package{
		Name:        packageName,
		NixName:     "pkgs." + packageName,
		DisplayName: packageName,
		Version:     "",
	}

	return pkg, nil
}

// DoesPackageExistInDevenv checks if a package already exists in devenv.nix
// Returns: exists, isKanukaManaged, error
func DoesPackageExistInDevenv(nixName string) (bool, bool, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return false, false, fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvPath := filepath.Join(currentDir, "devenv.nix")
	content, err := os.ReadFile(devenvPath)
	if err != nil {
		return false, false, fmt.Errorf("failed to read devenv.nix: %w", err)
	}

	contentStr := string(content)
	
	// Check if package exists anywhere in the file
	packageExists := strings.Contains(contentStr, nixName)
	
	// Check if it's in the Kanuka-managed section
	kanukaManaged := false
	if packageExists {
		kanukaManaged = isInKanukaManagedSection(contentStr, nixName)
	}

	return packageExists, kanukaManaged, nil
}

// isInKanukaManagedSection checks if a package is in the Kanuka-managed section
func isInKanukaManagedSection(content, nixName string) bool {
	lines := strings.Split(content, "\n")
	inKanukaSection := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		if strings.Contains(trimmed, "# Kanuka-managed packages - DO NOT EDIT MANUALLY") {
			inKanukaSection = true
			continue
		}
		
		if strings.Contains(trimmed, "# End Kanuka-managed packages") {
			inKanukaSection = false
			continue
		}
		
		if inKanukaSection && strings.Contains(line, nixName) {
			return true
		}
	}
	
	return false
}

// AddPackageToDevenv adds a package to the Kanuka-managed section of devenv.nix
func AddPackageToDevenv(pkg *Package) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvPath := filepath.Join(currentDir, "devenv.nix")
	content, err := os.ReadFile(devenvPath)
	if err != nil {
		return fmt.Errorf("failed to read devenv.nix: %w", err)
	}

	contentStr := string(content)
	
	// Find the Kanuka-managed section and add the package
	lines := strings.Split(contentStr, "\n")
	var newLines []string
	
	for _, line := range lines {
		// Look for the end of Kanuka-managed packages section
		if strings.Contains(strings.TrimSpace(line), "# End Kanuka-managed packages") {
			// Insert the package before this line
			newLines = append(newLines, "    "+pkg.NixName)
			newLines = append(newLines, line)
		} else {
			newLines = append(newLines, line)
		}
	}

	// Write the updated content back
	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(devenvPath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write devenv.nix: %w", err)
	}

	return nil
}

// RemovePackageFromDevenv removes a package from the Kanuka-managed section
func RemovePackageFromDevenv(nixName string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvPath := filepath.Join(currentDir, "devenv.nix")
	content, err := os.ReadFile(devenvPath)
	if err != nil {
		return fmt.Errorf("failed to read devenv.nix: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	inKanukaSection := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		if strings.Contains(trimmed, "# Kanuka-managed packages - DO NOT EDIT MANUALLY") {
			inKanukaSection = true
			newLines = append(newLines, line)
			continue
		}
		
		if strings.Contains(trimmed, "# End Kanuka-managed packages") {
			inKanukaSection = false
			newLines = append(newLines, line)
			continue
		}
		
		// Skip the line if it's in Kanuka section and contains our package
		if inKanukaSection && strings.Contains(line, nixName) {
			continue
		}
		
		newLines = append(newLines, line)
	}

	// Write the updated content back
	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(devenvPath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write devenv.nix: %w", err)
	}

	return nil
}

// GetKanukaManagedPackages returns a list of packages managed by Kanuka
func GetKanukaManagedPackages() ([]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvPath := filepath.Join(currentDir, "devenv.nix")
	file, err := os.Open(devenvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open devenv.nix: %w", err)
	}
	defer file.Close()

	var packages []string
	scanner := bufio.NewScanner(file)
	inKanukaSection := false
	
	// Regex to match package lines like "    pkgs.nodejs_18"
	packageRegex := regexp.MustCompile(`^\s+pkgs\.(\w+)`)
	
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		
		if strings.Contains(trimmed, "# Kanuka-managed packages - DO NOT EDIT MANUALLY") {
			inKanukaSection = true
			continue
		}
		
		if strings.Contains(trimmed, "# End Kanuka-managed packages") {
			inKanukaSection = false
			continue
		}
		
		if inKanukaSection {
			matches := packageRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				packages = append(packages, matches[1])
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading devenv.nix: %w", err)
	}

	return packages, nil
}