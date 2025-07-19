package cmd

import (
	"fmt"
	"strings"
	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	searchByName        string
	searchByProgram     string
	searchByVersion     string
	searchMaxResults    int
	searchShowDetails   bool
	searchOutputJSON    bool
)

var groveSearchCmd = &cobra.Command{
	Use:   "search <term>",
	Short: "Search nixpkgs for packages",
	Long: `Search nixpkgs for packages using multiple search modes.
Supports searching by package name, program/binary name, version, or general search.

Examples:
  kanuka grove search nodejs              # General search for nodejs
  kanuka grove search --name nodejs       # Search by exact package name
  kanuka grove search --program node      # Search for packages providing 'node' binary
  kanuka grove search --name nodejs --details  # Show detailed package information
  kanuka grove search python --max-results 10  # Limit results to 10 packages`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow no args if specific search flags are provided
		if len(args) == 0 && searchByName == "" && searchByProgram == "" && searchByVersion == "" {
			return fmt.Errorf("requires at least one search term or search flag")
		}
		if len(args) > 1 {
			return fmt.Errorf("accepts at most 1 search term")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var searchTerm string
		if len(args) > 0 {
			searchTerm = args[0]
		}

		GroveLogger.Infof("Starting grove search command")
		spinner, cleanup := startGroveSpinner("Searching packages...", groveVerbose)
		defer cleanup()

		// Determine search type and execute appropriate search
		var results []PackageSearchResult
		var err error

		if searchByName != "" {
			GroveLogger.Debugf("Searching by name: %s", searchByName)
			results, err = performNameSearch(searchByName)
		} else if searchByProgram != "" {
			GroveLogger.Debugf("Searching by program: %s", searchByProgram)
			results, err = performProgramSearch(searchByProgram)
		} else if searchTerm != "" {
			GroveLogger.Debugf("Performing general search: %s", searchTerm)
			results, err = performGeneralSearch(searchTerm)
		} else {
			spinner.FinalMSG = color.RedString("✗") + " No search term provided"
			return nil
		}

		if err != nil {
			finalMessage := color.RedString("✗") + " Search failed: " + err.Error()
			if strings.Contains(err.Error(), "failed to create search client") {
				finalMessage += "\n" + color.CyanString("→") + " Check your internet connection and try again"
			}
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Format and display results
		if len(results) == 0 {
			finalMessage := color.YellowString("!") + " No packages found"
			if searchTerm != "" {
				finalMessage += "\n" + color.CyanString("→") + " Try a broader search term or different search mode"
				finalMessage += "\n" + color.CyanString("→") + " Use " + color.YellowString("--name") + " for exact name matching"
				finalMessage += "\n" + color.CyanString("→") + " Use " + color.YellowString("--program") + " to search by binary name"
			}
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Stop spinner and display results
		spinner.Stop()
		
		if searchOutputJSON {
			displayResultsJSON(results)
		} else {
			displayResultsFormatted(results, searchTerm)
		}

		return nil
	},
}

// PackageSearchResult represents a search result for display
type PackageSearchResult struct {
	AttrName    string
	Name        string
	Version     string
	Description string
	Programs    []string
	Homepage    []string
}

// performNameSearch searches for packages by exact name
func performNameSearch(name string) ([]PackageSearchResult, error) {
	packages, err := grove.SearchPackagesByName(name)
	if err != nil {
		return nil, err
	}
	
	return convertToSearchResults(packages), nil
}

// performProgramSearch searches for packages by program/binary name
func performProgramSearch(program string) ([]PackageSearchResult, error) {
	packages, err := grove.SearchPackagesByProgram(program)
	if err != nil {
		return nil, err
	}
	
	return convertToSearchResults(packages), nil
}

// performGeneralSearch performs a general search across all package fields
func performGeneralSearch(term string) ([]PackageSearchResult, error) {
	maxResults := searchMaxResults
	if maxResults <= 0 {
		maxResults = 25 // Default
	}
	
	packages, err := grove.SearchPackagesGeneral(term, maxResults)
	if err != nil {
		return nil, err
	}
	
	return convertToSearchResults(packages), nil
}

// convertToSearchResults converts grove.NixSearchPackage to our display format
func convertToSearchResults(packages []grove.NixSearchPackage) []PackageSearchResult {
	results := make([]PackageSearchResult, len(packages))
	for i, pkg := range packages {
		results[i] = PackageSearchResult{
			AttrName:    pkg.AttrName,
			Name:        pkg.Name,
			Version:     pkg.Version,
			Description: pkg.Description,
			Programs:    pkg.Programs,
			Homepage:    pkg.Homepage,
		}
	}
	return results
}

// displayResultsFormatted displays search results in a user-friendly format
func displayResultsFormatted(results []PackageSearchResult, searchTerm string) {
	// Header
	if searchTerm != "" {
		fmt.Printf("%s Search results for %s:\n\n", 
			color.GreenString("✓"), 
			color.YellowString("'"+searchTerm+"'"))
	} else {
		fmt.Printf("%s Search results:\n\n", color.GreenString("✓"))
	}

	// Display results
	for i, result := range results {
		if i >= searchMaxResults && searchMaxResults > 0 {
			break
		}

		// Package name and version
		fmt.Printf("%s @ %s\n", 
			color.CyanString(result.AttrName), 
			color.GreenString(result.Version))

		// Description
		if result.Description != "" {
			fmt.Printf("  %s\n", result.Description)
		}

		// Programs/binaries provided
		if len(result.Programs) > 0 && searchShowDetails {
			fmt.Printf("  %s %s\n", 
				color.YellowString("Programs:"), 
				strings.Join(result.Programs, ", "))
		}

		// Homepage
		if len(result.Homepage) > 0 && searchShowDetails {
			fmt.Printf("  %s %s\n", 
				color.YellowString("Homepage:"), 
				result.Homepage[0])
		}

		fmt.Println() // Empty line between results
	}

	// Footer with usage hints
	fmt.Printf("%s Found %d packages", 
		color.CyanString("→"), 
		len(results))
	
	if !searchShowDetails {
		fmt.Printf(" (use %s for more details)", color.YellowString("--details"))
	}
	
	fmt.Println()
	fmt.Printf("%s Add a package: %s\n", 
		color.CyanString("→"), 
		color.YellowString("kanuka grove add <package>"))
}

// displayResultsJSON displays search results in JSON format
func displayResultsJSON(results []PackageSearchResult) {
	// Simple JSON output for now - could be enhanced with proper JSON marshaling
	fmt.Println("[")
	for i, result := range results {
		if i >= searchMaxResults && searchMaxResults > 0 {
			break
		}
		
		fmt.Printf(`  {
    "attr_name": "%s",
    "name": "%s", 
    "version": "%s",
    "description": "%s"`,
			result.AttrName, result.Name, result.Version, result.Description)
		
		if len(result.Programs) > 0 {
			fmt.Printf(`,
    "programs": ["%s"]`, strings.Join(result.Programs, `", "`))
		}
		
		if len(result.Homepage) > 0 {
			fmt.Printf(`,
    "homepage": ["%s"]`, strings.Join(result.Homepage, `", "`))
		}
		
		fmt.Printf("\n  }")
		if i < len(results)-1 && (searchMaxResults <= 0 || i < searchMaxResults-1) {
			fmt.Printf(",")
		}
		fmt.Println()
	}
	fmt.Println("]")
}

func init() {
	groveSearchCmd.Flags().StringVar(&searchByName, "name", "", "search by exact package name")
	groveSearchCmd.Flags().StringVar(&searchByProgram, "program", "", "search by program/binary name")
	groveSearchCmd.Flags().StringVar(&searchByVersion, "version", "", "search by version (future feature)")
	groveSearchCmd.Flags().IntVarP(&searchMaxResults, "max-results", "m", 25, "maximum number of results to show")
	groveSearchCmd.Flags().BoolVarP(&searchShowDetails, "details", "d", false, "show detailed package information")
	groveSearchCmd.Flags().BoolVarP(&searchOutputJSON, "json", "j", false, "output results in JSON format")
}