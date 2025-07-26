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
	NixName     string // Nix package name (e.g., pkgs.nodejs_18 or inputs.nixpkgs-stable.legacyPackages.${system}.nodejs_18)
	DisplayName string // Display name for user feedback.
	Version     string // Version if specified.
	Channel     string // Channel used (unstable, stable)
}

// ParsePackageName parses a package name with optional version and validates it exists in nixpkgs.
// Examples: "nodejs", "nodejs_18", "typescript".
func ParsePackageName(packageName string) (*Package, error) {
	return ParsePackageNameWithChannel(packageName, "unstable")
}

// ParsePackageNameWithChannel parses a package name with optional version and channel.
func ParsePackageNameWithChannel(packageName, channel string) (*Package, error) {
	if packageName == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}

	// Resolve and validate channel
	resolvedChannel, nixName, err := resolveChannelAndNixName(packageName, channel)
	if err != nil {
		return nil, err
	}

	// Get channel validation info to determine how to validate
	channelInfo := GetChannelValidationInfo(resolvedChannel)
	
	var result *NixSearchResult
	if channelInfo.IsOfficial {
		// Validate against official nixpkgs using the appropriate channel
		exists, searchResult, err := ValidatePackageExistsInChannel(packageName, channelInfo.SearchChannel)
		if err != nil {
			return nil, fmt.Errorf("failed to validate package: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("package '%s' not found in %s channel", packageName, channelInfo.Name)
		}
		result = searchResult
	} else {
		// Custom channel - skip validation but create a basic result
		result = &NixSearchResult{
			PackageName: packageName,
			Description: fmt.Sprintf("Package from custom channel '%s'", channelInfo.Name),
		}
	}

	// Create package with validated information.
	pkg := &Package{
		Name:        packageName,
		NixName:     nixName,
		DisplayName: packageName,
		Version:     "",
		Channel:     resolvedChannel,
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

	// Ensure the let block has the necessary channel imports
	updatedContent, err := ensureChannelImportsInLetBlock(contentStr, pkg.Channel)
	if err != nil {
		return fmt.Errorf("failed to update let block: %w", err)
	}

	// Find the Kanuka-managed section and add the package.
	lines := strings.Split(updatedContent, "\n")
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
		// If devenv.nix doesn't exist, return empty list (no packages managed)
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to open devenv.nix: %w", err)
	}
	defer file.Close()

	var packages []string
	scanner := bufio.NewScanner(file)
	inKanukaSection := false

	// Regex to match package lines like "    pkgs.nodejs_18" or "    pkgs-stable.python3"
	packageRegex := regexp.MustCompile(`^\s+(pkgs(?:-\w+)?\.(\w+))`)

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
				packages = append(packages, matches[1]) // Return full nix name (e.g., "pkgs-stable.python3")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading devenv.nix: %w", err)
	}

	return packages, nil
}

// Common devenv supported languages.
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

// GetKanukaManagedLanguages returns a list of languages that are managed by Kanuka.
func GetKanukaManagedLanguages() ([]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvPath := filepath.Join(currentDir, "devenv.nix")
	content, err := os.ReadFile(devenvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read devenv.nix: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var languages []string
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

		if inKanukaSection && strings.Contains(trimmed, "languages.") && strings.Contains(trimmed, ".enable = true;") {
			// Extract language name from "languages.LANGUAGE.enable = true;"
			parts := strings.Split(trimmed, ".")
			if len(parts) >= 2 {
				languageName := parts[1]
				languages = append(languages, languageName)
			}
		}
	}

	return languages, nil
}

// ChannelValidationInfo contains information about how to validate a channel
type ChannelValidationInfo struct {
	Name           string
	IsOfficial     bool
	SearchChannel  string // For nix-search-cli ("unstable" or version like "24.05")
}

// GetChannelValidationInfo determines how to validate packages for a given channel
func GetChannelValidationInfo(resolvedChannelName string) ChannelValidationInfo {
	switch resolvedChannelName {
	case "nixpkgs":
		return ChannelValidationInfo{
			Name:          "nixpkgs",
			IsOfficial:    true,
			SearchChannel: "unstable",
		}
	case "nixpkgs-stable":
		// Extract version from the latest stable channel (e.g., "nixos-24.05" -> "24.05")
		latestStable := GetLatestStableChannel()
		stableVersion := strings.TrimPrefix(latestStable, "nixos-")
		return ChannelValidationInfo{
			Name:          "nixpkgs-stable",
			IsOfficial:    true,
			SearchChannel: stableVersion,
		}
	default:
		return ChannelValidationInfo{
			Name:          resolvedChannelName,
			IsOfficial:    false,
			SearchChannel: "",
		}
	}
}

// resolveChannelAndNixName resolves a channel name to actual channel and generates the appropriate nix name
func resolveChannelAndNixName(packageName, channel string) (string, string, error) {
	// Get available channels from devenv.yaml
	availableChannels, err := ListChannels()
	if err != nil {
		return "", "", fmt.Errorf("failed to read available channels: %w", err)
	}

	// Create a map of available channel names
	channelMap := make(map[string]string)
	for _, ch := range availableChannels {
		channelMap[ch.Name] = ch.Name
	}

	// Handle user-friendly aliases
	var resolvedChannelName string
	switch channel {
	case "unstable":
		// Map "unstable" to "nixpkgs" if it exists, otherwise use first unstable-like channel
		if _, exists := channelMap["nixpkgs"]; exists {
			resolvedChannelName = "nixpkgs"
		} else {
			// Find first channel with "unstable" in URL
			for _, ch := range availableChannels {
				if strings.Contains(ch.URL, "nixpkgs-unstable") {
					resolvedChannelName = ch.Name
					break
				}
			}
			if resolvedChannelName == "" {
				return "", "", fmt.Errorf("no unstable channel found in devenv.yaml")
			}
		}
	case "stable":
		// Map "stable" to "nixpkgs-stable" if it exists, otherwise use first stable-like channel
		if _, exists := channelMap["nixpkgs-stable"]; exists {
			resolvedChannelName = "nixpkgs-stable"
		} else {
			// Find first channel with stable version in URL
			for _, ch := range availableChannels {
				if strings.Contains(ch.URL, "nixos-") && !strings.Contains(ch.URL, "unstable") {
					resolvedChannelName = ch.Name
					break
				}
			}
			if resolvedChannelName == "" {
				return "", "", fmt.Errorf("no stable channel found in devenv.yaml")
			}
		}
	default:
		// Direct channel name - validate it exists
		if _, exists := channelMap[channel]; !exists {
			// Build helpful error message with available channels
			var availableNames []string
			for _, ch := range availableChannels {
				availableNames = append(availableNames, ch.Name)
			}
			return "", "", fmt.Errorf("channel '%s' not found in devenv.yaml. Available channels: %s", 
				channel, strings.Join(availableNames, ", "))
		}
		resolvedChannelName = channel
	}

	// Generate appropriate nix name based on resolved channel using the correct devenv pattern
	var nixName string
	if resolvedChannelName == "nixpkgs" {
		// Default nixpkgs uses pkgs.
		nixName = "pkgs." + packageName
	} else if resolvedChannelName == "nixpkgs-stable" {
		// nixpkgs-stable uses the imported pkgs-stable from the let block
		nixName = "pkgs-stable." + packageName
	} else {
		// Custom channels use pkgs-<channel-name> pattern
		// We need to ensure the let block imports them properly
		channelVarName := "pkgs-" + strings.ReplaceAll(resolvedChannelName, "-", "_")
		nixName = channelVarName + "." + packageName
	}

	return resolvedChannelName, nixName, nil
}

// ensureChannelImportsInLetBlock ensures that the let block contains the necessary channel imports
func ensureChannelImportsInLetBlock(content, channelName string) (string, error) {
	// If it's the default nixpkgs channel, no import needed
	if channelName == "nixpkgs" {
		return content, nil
	}

	// Generate the import variable name
	var importVarName string
	if channelName == "nixpkgs-stable" {
		importVarName = "pkgs-stable"
	} else {
		importVarName = "pkgs-" + strings.ReplaceAll(channelName, "-", "_")
	}

	// Check if the import already exists in the let block
	importLine := fmt.Sprintf("%s = import inputs.%s { system = pkgs.stdenv.system; };", importVarName, channelName)
	if strings.Contains(content, importVarName+" = import inputs."+channelName) {
		return content, nil // Already exists
	}

	// Find the let block and add the import
	lines := strings.Split(content, "\n")
	var newLines []string
	inLetBlock := false
	letBlockFound := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Detect start of let block
		if strings.HasPrefix(trimmed, "let") {
			inLetBlock = true
			letBlockFound = true
			newLines = append(newLines, line)
			continue
		}
		
		// Detect end of let block (line starting with "in")
		if inLetBlock && strings.HasPrefix(trimmed, "in") {
			// Insert the new import before the "in" line
			newLines = append(newLines, "  "+importLine)
			newLines = append(newLines, line)
			inLetBlock = false
			continue
		}
		
		// If we're in the let block and this is an import line, just add it
		if inLetBlock && strings.Contains(line, "= import inputs.") {
			newLines = append(newLines, line)
			continue
		}
		
		newLines = append(newLines, line)
	}

	// If no let block was found, we need to create one
	if !letBlockFound {
		return addLetBlockToDevenvNix(content, channelName)
	}

	return strings.Join(newLines, "\n"), nil
}

// addLetBlockToDevenvNix adds a let block to devenv.nix if it doesn't exist
func addLetBlockToDevenvNix(content, channelName string) (string, error) {
	// Generate the import variable name and line
	var importVarName string
	if channelName == "nixpkgs-stable" {
		importVarName = "pkgs-stable"
	} else {
		importVarName = "pkgs-" + strings.ReplaceAll(channelName, "-", "_")
	}
	
	importLine := fmt.Sprintf("  %s = import inputs.%s { system = pkgs.stdenv.system; };", importVarName, channelName)
	
	lines := strings.Split(content, "\n")
	var newLines []string
	
	for _, line := range lines {
		// Look for the function signature line
		if strings.Contains(line, "{ pkgs, inputs, ... }:") {
			newLines = append(newLines, line)
			// Add the let block after the function signature
			newLines = append(newLines, "let")
			newLines = append(newLines, "  # Import additional nixpkgs channels for multi-channel support")
			if channelName != "nixpkgs-stable" {
				// Also add pkgs-stable if it's not already there and we're adding a custom channel
				newLines = append(newLines, "  pkgs-stable = import inputs.nixpkgs-stable { system = pkgs.stdenv.system; };")
			}
			newLines = append(newLines, importLine)
			newLines = append(newLines, "in")
			continue
		}
		
		newLines = append(newLines, line)
	}
	
	return strings.Join(newLines, "\n"), nil
}
