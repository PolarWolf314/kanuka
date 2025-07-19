package grove

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Package represents a parsed package with version information.
type Package struct {
	Name        string // Original name as provided by user.
	NixName     string // Nix package name (e.g., pkgs.nodejs_18)
	DisplayName string // Display name for user feedback.
	Version     string // Version if specified.
}

// ParsePackageName parses a package name with optional version and validates it exists in nixpkgs.
// Examples: "nodejs", "nodejs_18", "typescript".
func ParsePackageName(packageName string) (*Package, error) {
	if packageName == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}

	// Validate package exists in nixpkgs.
	exists, result, err := ValidatePackageExists(packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to validate package: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("package '%s' not found in nixpkgs", packageName)
	}

	// Create package with validated information.
	pkg := &Package{
		Name:        packageName,
		NixName:     "pkgs." + packageName,
		DisplayName: packageName,
		Version:     "",
	}

	// If we have result information, we could use it for better display.
	if result != nil && result.Description != "" {
		pkg.DisplayName = packageName + " (" + result.Description + ")"
	}

	return pkg, nil
}

// ParsePackageNameWithoutValidation parses a package name without nixpkgs validation (for testing).
func ParsePackageNameWithoutValidation(packageName string) (*Package, error) {
	if packageName == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}

	pkg := &Package{
		Name:        packageName,
		NixName:     "pkgs." + packageName,
		DisplayName: packageName,
		Version:     "",
	}

	return pkg, nil
}

// DoesPackageExistInDevenv checks if a package already exists in devenv.nix
// Returns: exists, isKanukaManaged, error.
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

	// Check if package exists anywhere in the file.
	packageExists := strings.Contains(contentStr, nixName)

	// Check if it's in the Kanuka-managed section.
	kanukaManaged := false
	if packageExists {
		kanukaManaged = isInKanukaManagedSection(contentStr, nixName)
	}

	return packageExists, kanukaManaged, nil
}

// isInKanukaManagedSection checks if a package is in the Kanuka-managed section.
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

// AddPackageToDevenv adds a package to the Kanuka-managed section of devenv.nix.
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

	// Find the Kanuka-managed section and add the package.
	lines := strings.Split(contentStr, "\n")
	var newLines []string

	for _, line := range lines {
		// Look for the end of Kanuka-managed packages section.
		if strings.Contains(strings.TrimSpace(line), "# End Kanuka-managed packages") {
			// Insert the package before this line.
			newLines = append(newLines, "    "+pkg.NixName)
			newLines = append(newLines, line)
		} else {
			newLines = append(newLines, line)
		}
	}

	// Write the updated content back.
	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(devenvPath, []byte(newContent), 0600)
	if err != nil {
		return fmt.Errorf("failed to write devenv.nix: %w", err)
	}

	return nil
}

// RemovePackageFromDevenv removes a package from the Kanuka-managed section.
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

		// Skip the line if it's in Kanuka section and contains our package.
		if inKanukaSection && strings.Contains(line, nixName) {
			continue
		}

		newLines = append(newLines, line)
	}

	// Write the updated content back.
	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(devenvPath, []byte(newContent), 0600)
	if err != nil {
		return fmt.Errorf("failed to write devenv.nix: %w", err)
	}

	return nil
}

// GetKanukaManagedPackages returns a list of packages managed by Kanuka.
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

// Common devenv supported languages
var supportedLanguages = map[string]bool{
	"ansible":        true,
	"c":              true,
	"clojure":        true,
	"cplusplus":      true,
	"crystal":        true,
	"cue":            true,
	"dart":           true,
	"deno":           true,
	"dotnet":         true,
	"elixir":         true,
	"elm":            true,
	"erlang":         true,
	"fortran":        true,
	"gawk":           true,
	"gleam":          true,
	"go":             true,
	"haskell":        true,
	"idris":          true,
	"java":           true,
	"javascript":     true,
	"jsonnet":        true,
	"julia":          true,
	"kotlin":         true,
	"lean4":          true,
	"lua":            true,
	"nim":            true,
	"nix":            true,
	"ocaml":          true,
	"odin":           true,
	"opentofu":       true,
	"pascal":         true,
	"perl":           true,
	"php":            true,
	"purescript":     true,
	"python":         true,
	"r":              true,
	"racket":         true,
	"raku":           true,
	"robotframework": true,
	"ruby":           true,
	"rust":           true,
	"scala":          true,
	"shell":          true,
	"solidity":       true,
	"standardml":     true,
	"swift":          true,
	"terraform":      true,
	"texlive":        true,
	"typescript":     true,
	"typst":          true,
	"unison":         true,
	"v":              true,
	"vala":           true,
	"zig":            true,
}

// IsLanguage checks if the given name is a supported devenv language.
func IsLanguage(name string) bool {
	return supportedLanguages[name]
}

// DoesLanguageExistInDevenv checks if a language is already enabled in devenv.nix
// Returns: exists, isKanukaManaged, error.
func DoesLanguageExistInDevenv(languageName string) (bool, bool, error) {
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

	// Check if language exists anywhere in the file.
	languagePattern := fmt.Sprintf("languages.%s", languageName)
	languageExists := strings.Contains(contentStr, languagePattern)

	// Check if it's in the Kanuka-managed section.
	kanukaManaged := false
	if languageExists {
		kanukaManaged = isLanguageInKanukaManagedSection(contentStr, languageName)
	}

	return languageExists, kanukaManaged, nil
}

// isLanguageInKanukaManagedSection checks if a language is in the Kanuka-managed section.
func isLanguageInKanukaManagedSection(content, languageName string) bool {
	lines := strings.Split(content, "\n")
	inKanukaSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "# Kanuka-managed languages - DO NOT EDIT MANUALLY") {
			inKanukaSection = true
			continue
		}

		if strings.Contains(trimmed, "# End Kanuka-managed languages") {
			inKanukaSection = false
			continue
		}

		if inKanukaSection && strings.Contains(line, fmt.Sprintf("languages.%s", languageName)) {
			return true
		}
	}

	return false
}

// AddLanguageToDevenv adds a language to the Kanuka-managed section of devenv.nix.
func AddLanguageToDevenv(languageName string) error {
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

	// Check if Kanuka-managed languages section exists, if not create it
	if !strings.Contains(contentStr, "# Kanuka-managed languages - DO NOT EDIT MANUALLY") {
		// Add the languages section before the enterShell section
		lines := strings.Split(contentStr, "\n")
		var newLines []string

		for i, line := range lines {
			if strings.Contains(strings.TrimSpace(line), "enterShell") {
				// Insert languages section before enterShell
				newLines = append(newLines, "")
				newLines = append(newLines, "  # Kanuka-managed languages - DO NOT EDIT MANUALLY")
				newLines = append(newLines, fmt.Sprintf("  languages.%s.enable = true;", languageName))
				newLines = append(newLines, "  # End Kanuka-managed languages")
				newLines = append(newLines, "")
				newLines = append(newLines, line)
				// Add remaining lines
				newLines = append(newLines, lines[i+1:]...)
				break
			} else {
				newLines = append(newLines, line)
			}
		}

		// Write the updated content back
		newContent := strings.Join(newLines, "\n")
		err = os.WriteFile(devenvPath, []byte(newContent), 0600)
		if err != nil {
			return fmt.Errorf("failed to write devenv.nix: %w", err)
		}
		return nil
	}

	// Find the Kanuka-managed languages section and add the language
	lines := strings.Split(contentStr, "\n")
	var newLines []string

	for _, line := range lines {
		// Look for the end of Kanuka-managed languages section
		if strings.Contains(strings.TrimSpace(line), "# End Kanuka-managed languages") {
			// Insert the language before this line
			newLines = append(newLines, fmt.Sprintf("  languages.%s.enable = true;", languageName))
			newLines = append(newLines, line)
		} else {
			newLines = append(newLines, line)
		}
	}

	// Write the updated content back
	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(devenvPath, []byte(newContent), 0600)
	if err != nil {
		return fmt.Errorf("failed to write devenv.nix: %w", err)
	}

	return nil
}

// RemoveLanguageFromDevenv removes a language from the Kanuka-managed section.
func RemoveLanguageFromDevenv(languageName string) error {
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

		if strings.Contains(trimmed, "# Kanuka-managed languages - DO NOT EDIT MANUALLY") {
			inKanukaSection = true
			newLines = append(newLines, line)
			continue
		}

		if strings.Contains(trimmed, "# End Kanuka-managed languages") {
			inKanukaSection = false
			newLines = append(newLines, line)
			continue
		}

		// Skip the line if it's in Kanuka section and contains our language
		if inKanukaSection && strings.Contains(line, fmt.Sprintf("languages.%s", languageName)) {
			continue
		}

		newLines = append(newLines, line)
	}

	// Write the updated content back
	newContent := strings.Join(newLines, "\n")
	err = os.WriteFile(devenvPath, []byte(newContent), 0600)
	if err != nil {
		return fmt.Errorf("failed to write devenv.nix: %w", err)
	}

	return nil
}
