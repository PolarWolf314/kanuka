package cmd

import (
	"fmt"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var skipValidation bool

var groveAddCmd = &cobra.Command{
	Use:   "add <package>[@version]",
	Short: "Add a package to the development environment",
	Long: `Add a package to your development environment by modifying devenv.nix.
Packages are added to the Kanuka-managed section and can optionally include version specifications.

Examples:
  kanuka grove add nodejs          # Add latest nodejs
  kanuka grove add nodejs_18       # Add nodejs version 18
  kanuka grove add typescript      # Add typescript package
  kanuka grove add awscli2         # Add AWS CLI v2`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageName := args[0]

		GroveLogger.Infof("Starting grove add command for package: %s", packageName)
		spinner, cleanup := startGroveSpinner("Searching and validating package...", groveVerbose)
		defer cleanup()

		// Check if we're in a grove project
		GroveLogger.Debugf("Checking if kanuka.toml exists")
		exists, err := grove.DoesKanukaTomlExist()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check project status: %v", err)
		}
		if !exists {
			finalMessage := color.RedString("✗") + " Not in a grove project\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka grove init") + " first"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Check if devenv.nix exists
		GroveLogger.Debugf("Checking if devenv.nix exists")
		devenvExists, err := grove.DoesDevenvNixExist()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check devenv.nix: %v", err)
		}
		if !devenvExists {
			finalMessage := color.RedString("✗") + " devenv.nix not found\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka grove init") + " to create it"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Check if this is a language or package
		GroveLogger.Debugf("Checking if '%s' is a language or package", packageName)
		isLanguage := grove.IsLanguage(packageName)

		if isLanguage {
			// Handle language addition
			GroveLogger.Debugf("Handling language addition for: %s", packageName)
			return handleLanguageAddition(packageName, spinner)
		} else {
			// Handle package addition
			GroveLogger.Debugf("Handling package addition for: %s", packageName)
			return handlePackageAddition(packageName, spinner, skipValidation)
		}
	},
}

// getPackageSuggestions tries to find similar package names for better error messages.
func getPackageSuggestions(packageName string) []string {
	// Try a general search to find similar packages
	results, err := grove.SearchPackagesGeneral(packageName, 3)
	if err != nil {
		return nil
	}

	var suggestions []string
	for _, result := range results {
		if result.AttrName != packageName && len(suggestions) < 3 {
			suggestions = append(suggestions, result.AttrName)
		}
	}

	return suggestions
}

// isLikelyProgramName checks if a package name looks like it could be a program/binary name.
func isLikelyProgramName(name string) bool {
	// Simple heuristics: short names, common program patterns
	if len(name) <= 10 && !strings.Contains(name, "_") && !strings.Contains(name, "-") {
		return true
	}

	// Common program name patterns
	commonPrograms := []string{"go", "node", "python", "java", "rust", "gcc", "git", "vim", "curl", "wget"}
	for _, prog := range commonPrograms {
		if strings.Contains(name, prog) {
			return true
		}
	}

	return false
}

// handlePackageAddition handles the addition of a package to devenv.nix.
func handlePackageAddition(packageName string, spinner *spinner.Spinner, skipValidation bool) error {
	// Parse package name and version
	GroveLogger.Debugf("Parsing package name: %s", packageName)
	var parsedPackage *grove.Package
	var err error
	if skipValidation {
		parsedPackage, err = grove.ParsePackageNameWithoutValidation(packageName)
	} else {
		parsedPackage, err = grove.ParsePackageName(packageName)
	}
	if err != nil {
		// Handle validation errors with proper spinner cleanup and enhanced suggestions
		var finalMessage string
		if strings.Contains(err.Error(), "failed to create search client") {
			finalMessage = color.RedString("✗") + " Failed to connect to package search service\n" +
				color.CyanString("→") + " Check your internet connection and try again\n" +
				color.CyanString("→") + " Or use " + color.YellowString("--skip-validation") + " flag for testing"
		} else if strings.Contains(err.Error(), "not found in nixpkgs") {
			// Try to provide helpful suggestions using the new search capabilities
			suggestions := getPackageSuggestions(packageName)
			finalMessage = color.RedString("✗") + " Package '" + packageName + "' not found in nixpkgs\n" +
				color.CyanString("→") + " Try " + color.YellowString("kanuka grove search "+packageName) + " to find similar packages"

			if len(suggestions) > 0 {
				finalMessage += "\n" + color.CyanString("→") + " Similar packages: " + color.YellowString(strings.Join(suggestions, ", "))
			}

			// Suggest program-based search if the package name looks like a binary
			if isLikelyProgramName(packageName) {
				finalMessage += "\n" + color.CyanString("→") + " Or search by program: " + color.YellowString("kanuka grove search --program "+packageName)
			}
		} else {
			finalMessage = color.RedString("✗") + " Failed to validate package: " + err.Error()
		}
		spinner.FinalMSG = finalMessage
		return nil
	}
	GroveLogger.Infof("Parsed package: %s", parsedPackage.NixName)

	// Check if package already exists
	GroveLogger.Debugf("Checking if package already exists in devenv.nix")
	exists, isKanukaManaged, err := grove.DoesPackageExistInDevenv(parsedPackage.NixName)
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to check existing packages: %v", err)
	}

	if exists {
		if isKanukaManaged {
			finalMessage := color.YellowString("!") + " Package '" + parsedPackage.NixName + "' already managed by Kanuka\n" +
				color.CyanString("→") + " Use " + color.YellowString("kanuka grove remove "+packageName) + " first to replace it"
			spinner.FinalMSG = finalMessage
			return nil
		} else {
			// Package exists but not managed by Kanuka - ask for confirmation
			spinner.Stop()
			GroveLogger.WarnfUser("Package '%s' already exists in devenv.nix (not managed by Kanuka)", parsedPackage.NixName)
			GroveLogger.WarnfUser("Replace existing package? (y/N)")

			var response string
			_, err := fmt.Scanln(&response)
			if err != nil {
				return err
			}

			if response != "y" && response != "Y" {
				finalMessage := color.YellowString("!") + " Package addition cancelled"
				spinner.FinalMSG = finalMessage
				spinner.Restart()
				return nil
			}
			spinner.Restart()
		}
	}

	// Update spinner message for the actual addition step
	spinner.Suffix = " Adding package to devenv.nix..."

	// Add package to devenv.nix
	GroveLogger.Debugf("Adding package to devenv.nix")
	if err := grove.AddPackageToDevenv(parsedPackage); err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to add package: %v", err)
	}
	GroveLogger.Infof("Package added successfully")

	finalMessage := color.GreenString("✓") + " Added " + parsedPackage.NixName + " to devenv.nix\n" +
		color.CyanString("→") + " Run " + color.YellowString("kanuka grove enter") + " to start using " + parsedPackage.DisplayName

	spinner.FinalMSG = finalMessage
	return nil
}

// handleLanguageAddition handles the addition of a language to devenv.nix.
func handleLanguageAddition(languageName string, spinner *spinner.Spinner) error {
	// Check if language already exists
	GroveLogger.Debugf("Checking if language already exists in devenv.nix")
	exists, isKanukaManaged, err := grove.DoesLanguageExistInDevenv(languageName)
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to check existing languages: %v", err)
	}

	if exists {
		if isKanukaManaged {
			finalMessage := color.YellowString("!") + " Language '" + languageName + "' already managed by Kanuka\n" +
				color.CyanString("→") + " Use " + color.YellowString("kanuka grove remove "+languageName) + " first to replace it"
			spinner.FinalMSG = finalMessage
			return nil
		} else {
			// Language exists but not managed by Kanuka - ask for confirmation
			spinner.Stop()
			GroveLogger.WarnfUser("Language '%s' already exists in devenv.nix (not managed by Kanuka)", languageName)
			GroveLogger.WarnfUser("Replace existing language? (y/N)")

			var response string
			_, err := fmt.Scanln(&response)
			if err != nil {
				return err
			}

			if response != "y" && response != "Y" {
				finalMessage := color.YellowString("!") + " Language addition cancelled"
				spinner.FinalMSG = finalMessage
				spinner.Restart()
				return nil
			}
			spinner.Restart()
		}
	}

	// Update spinner message for the actual addition step
	spinner.Suffix = " Adding language to devenv.nix..."

	// Add language to devenv.nix
	GroveLogger.Debugf("Adding language to devenv.nix")
	if err := grove.AddLanguageToDevenv(languageName); err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to add language: %v", err)
	}
	GroveLogger.Infof("Language added successfully")

	finalMessage := color.GreenString("✓") + " Added " + languageName + " language to devenv.nix\n" +
		color.CyanString("→") + " Run " + color.YellowString("kanuka grove enter") + " to start using " + languageName

	spinner.FinalMSG = finalMessage
	return nil
}

func init() {
	groveAddCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "skip nixpkgs validation (for testing)")
}
