package grove

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// NixSearchResult represents a single package from nix search output
type NixSearchResult struct {
	PackageName string `json:"pname"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// ValidatePackageExists checks if a package exists in nixpkgs using nix search
func ValidatePackageExists(packageName string) (bool, *NixSearchResult, error) {
	// Use nix search to find the package
	cmd := exec.Command("nix", "search", "nixpkgs", packageName, "--json")
	output, err := cmd.Output()
	if err != nil {
		// Check if nix command is available
		if strings.Contains(err.Error(), "executable file not found") {
			return false, nil, fmt.Errorf("nix command not found - please install Nix package manager")
		}
		// If nix search fails, it might mean no packages found
		return false, nil, nil
	}

	// Parse the JSON output
	var searchResults map[string]NixSearchResult
	if err := json.Unmarshal(output, &searchResults); err != nil {
		return false, nil, fmt.Errorf("failed to parse nix search output: %w", err)
	}

	// Look for exact match or close match
	exactMatch := "nixpkgs#" + packageName
	if result, exists := searchResults[exactMatch]; exists {
		return true, &result, nil
	}

	// Look for any match that contains the package name
	for key, result := range searchResults {
		if strings.Contains(key, packageName) {
			return true, &result, nil
		}
	}

	return false, nil, nil
}

// SearchPackages searches for packages in nixpkgs and returns multiple results
func SearchPackages(searchTerm string) (map[string]NixSearchResult, error) {
	cmd := exec.Command("nix", "search", "nixpkgs", searchTerm, "--json")
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			return nil, fmt.Errorf("nix command not found - please install Nix package manager")
		}
		return nil, fmt.Errorf("nix search failed: %w", err)
	}

	var searchResults map[string]NixSearchResult
	if err := json.Unmarshal(output, &searchResults); err != nil {
		return nil, fmt.Errorf("failed to parse nix search output: %w", err)
	}

	return searchResults, nil
}

// GetPackageNixName extracts the proper nix package name from search results
func GetPackageNixName(packageName string) (string, error) {
	exists, _, err := ValidatePackageExists(packageName)
	if err != nil {
		return "", err
	}
	
	if !exists {
		return "", fmt.Errorf("package '%s' not found in nixpkgs", packageName)
	}

	// For most packages, the nix name is just pkgs.packageName
	// But we could enhance this to handle special cases
	return "pkgs." + packageName, nil
}