package grove

import (
	"context"
	"fmt"
	"strings"

	"github.com/peterldowns/nix-search-cli/pkg/nixsearch"
)

// NixSearchPackage represents a package from nix-search-cli for external use.
type NixSearchPackage struct {
	AttrName    string
	Name        string
	Version     string
	Description string
	Programs    []string
	Homepage    []string
}

// NixSearchResult represents a single package from nix search output.
// This maintains compatibility with existing code while using nix-search-cli internally.
type NixSearchResult struct {
	PackageName string `json:"pname"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// convertPackageToResult converts nix-search-cli Package to our NixSearchResult.
func convertPackageToResult(pkg nixsearch.Package) *NixSearchResult {
	return &NixSearchResult{
		PackageName: pkg.Name,
		Version:     pkg.Version,
		Description: pkg.Description,
	}
}

// ValidatePackageExists checks if a package exists in nixpkgs using nix-search-cli.
// This function uses the unstable channel by default for backward compatibility.
func ValidatePackageExists(packageName string) (bool, *NixSearchResult, error) {
	return ValidatePackageExistsInChannel(packageName, "unstable")
}

// ValidatePackageExistsInChannel checks if a package exists in a specific nixpkgs channel using nix-search-cli.
func ValidatePackageExistsInChannel(packageName, channel string) (bool, *NixSearchResult, error) {
	client, err := nixsearch.NewElasticSearchClient()
	if err != nil {
		return false, nil, fmt.Errorf("failed to create search client: %w", err)
	}

	// Try exact name match first.
	query := nixsearch.Query{
		MaxResults: 1,
		Channel:    channel,
		Name:       &nixsearch.MatchName{Name: packageName},
	}

	results, err := client.Search(context.Background(), query)
	if err != nil {
		return false, nil, fmt.Errorf("search failed: %w", err)
	}

	if len(results) > 0 {
		return true, convertPackageToResult(results[0]), nil
	}

	// If no exact match, try a general search to see if package exists with similar name.
	generalQuery := nixsearch.Query{
		MaxResults: 5,
		Channel:    channel,
		Search:     &nixsearch.MatchSearch{Search: packageName},
	}

	generalResults, err := client.Search(context.Background(), generalQuery)
	if err != nil {
		return false, nil, fmt.Errorf("general search failed: %w", err)
	}

	// Look for close matches in attribute names.
	for _, result := range generalResults {
		if result.AttrName == packageName || strings.Contains(result.AttrName, packageName) {
			return true, convertPackageToResult(result), nil
		}
	}

	return false, nil, nil
}

// SearchPackages searches for packages in nixpkgs and returns multiple results.
func SearchPackages(searchTerm string) (map[string]NixSearchResult, error) {
	client, err := nixsearch.NewElasticSearchClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create search client: %w", err)
	}

	query := nixsearch.Query{
		MaxResults: 50, // Reasonable default for search results.
		Channel:    "unstable",
		Search:     &nixsearch.MatchSearch{Search: searchTerm},
	}

	results, err := client.Search(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert to map format for compatibility with existing code.
	searchResults := make(map[string]NixSearchResult)
	for _, result := range results {
		key := "nixpkgs#" + result.AttrName
		searchResults[key] = NixSearchResult{
			PackageName: result.Name,
			Version:     result.Version,
			Description: result.Description,
		}
	}

	return searchResults, nil
}

// convertNixSearchPackage converts nixsearch.Package to our NixSearchPackage type.
func convertNixSearchPackage(pkg nixsearch.Package) NixSearchPackage {
	return NixSearchPackage{
		AttrName:    pkg.AttrName,
		Name:        pkg.Name,
		Version:     pkg.Version,
		Description: pkg.Description,
		Programs:    pkg.Programs,
		Homepage:    pkg.Homepage,
	}
}

// SearchPackagesByName searches for packages by exact name match.
func SearchPackagesByName(packageName string) ([]NixSearchPackage, error) {
	client, err := nixsearch.NewElasticSearchClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create search client: %w", err)
	}

	query := nixsearch.Query{
		MaxResults: 20,
		Channel:    "unstable",
		Name:       &nixsearch.MatchName{Name: packageName},
	}

	results, err := client.Search(context.Background(), query)
	if err != nil {
		return nil, err
	}

	packages := make([]NixSearchPackage, len(results))
	for i, result := range results {
		packages[i] = convertNixSearchPackage(result)
	}

	return packages, nil
}

// SearchPackagesByProgram searches for packages that provide a specific program/binary.
func SearchPackagesByProgram(programName string) ([]NixSearchPackage, error) {
	client, err := nixsearch.NewElasticSearchClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create search client: %w", err)
	}

	query := nixsearch.Query{
		MaxResults: 20,
		Channel:    "unstable",
		Program:    &nixsearch.MatchProgram{Program: programName},
	}

	results, err := client.Search(context.Background(), query)
	if err != nil {
		return nil, err
	}

	packages := make([]NixSearchPackage, len(results))
	for i, result := range results {
		packages[i] = convertNixSearchPackage(result)
	}

	return packages, nil
}

// SearchPackagesGeneral performs a general search across all package fields.
func SearchPackagesGeneral(searchTerm string, maxResults int) ([]NixSearchPackage, error) {
	client, err := nixsearch.NewElasticSearchClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create search client: %w", err)
	}

	if maxResults <= 0 {
		maxResults = 25 // Default.
	}

	query := nixsearch.Query{
		MaxResults: maxResults,
		Channel:    "unstable",
		Search:     &nixsearch.MatchSearch{Search: searchTerm},
	}

	results, err := client.Search(context.Background(), query)
	if err != nil {
		return nil, err
	}

	packages := make([]NixSearchPackage, len(results))
	for i, result := range results {
		packages[i] = convertNixSearchPackage(result)
	}

	return packages, nil
}

// GetPackageNixName extracts the proper nix package name from search results.
func GetPackageNixName(packageName string) (string, error) {
	exists, _, err := ValidatePackageExists(packageName)
	if err != nil {
		return "", err
	}

	if !exists {
		return "", fmt.Errorf("package '%s' not found in nixpkgs", packageName)
	}

	// For most packages, the nix name is just pkgs.packageName
	// But we could enhance this to handle special cases.
	return "pkgs." + packageName, nil
}
