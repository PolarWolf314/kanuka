package cmd

import (
	"fmt"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var groveRemoveCmd = &cobra.Command{
	Use:   "remove <package>",
	Short: "Remove a package from the development environment",
	Long: `Remove a package from your development environment by modifying devenv.nix.
Only packages managed by Kanuka can be removed using this command.

Examples:
  kanuka grove remove nodejs_18       # Remove nodejs version 18
  kanuka grove remove typescript      # Remove typescript package
  kanuka grove remove awscli2         # Remove AWS CLI v2
  # Note: AWS SSO authentication uses integrated AWS SDK - no external dependencies!`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageName := args[0]

		GroveLogger.Infof("Starting grove remove command for package: %s", packageName)
		spinner, cleanup := startGroveSpinner("Checking package status...", groveVerbose)
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
			// Handle language removal
			GroveLogger.Debugf("Handling language removal for: %s", packageName)
			return handleLanguageRemoval(packageName, spinner)
		} else {
			// Handle package removal
			GroveLogger.Debugf("Handling package removal for: %s", packageName)
			return handlePackageRemoval(packageName, spinner)
		}
	},
}

// handlePackageRemoval handles the removal of a package from devenv.nix.
func handlePackageRemoval(packageName string, spinner *spinner.Spinner) error {
	// Parse package name to get the proper nix name format
	GroveLogger.Debugf("Parsing package name: %s", packageName)
	var nixName string
	if strings.HasPrefix(packageName, "pkgs.") {
		nixName = packageName
	} else {
		nixName = "pkgs." + packageName
	}

	// Check if package exists and is managed by Kanuka
	GroveLogger.Debugf("Checking if package exists in devenv.nix")
	exists, isKanukaManaged, err := grove.DoesPackageExistInDevenv(nixName)
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to check existing packages: %v", err)
	}

	if !exists {
		finalMessage := color.RedString("✗") + " Package '" + packageName + "' not found in devenv.nix\n" +
			color.CyanString("→") + " Use " + color.YellowString("kanuka grove search "+packageName) + " to find available packages"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if !isKanukaManaged {
		finalMessage := color.RedString("✗") + " Package '" + packageName + "' is not managed by Kanuka\n" +
			color.CyanString("→") + " Only Kanuka-managed packages can be removed with this command\n" +
			color.CyanString("→") + " Edit devenv.nix manually to remove packages added outside of Kanuka"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Ask for confirmation
	spinner.Stop()
	GroveLogger.WarnfUser("Remove package '%s' from devenv.nix? (y/N)", packageName)

	var response string
	_, err = fmt.Scanln(&response)
	if err != nil {
		return err
	}

	if response != "y" && response != "Y" {
		finalMessage := color.YellowString("!") + " Package removal cancelled"
		spinner.FinalMSG = finalMessage
		spinner.Restart()
		return nil
	}
	spinner.Restart()

	// Update spinner message for the actual removal step
	spinner.Suffix = " Removing package from devenv.nix..."

	// Remove package from devenv.nix
	GroveLogger.Debugf("Removing package from devenv.nix")
	if err := grove.RemovePackageFromDevenv(nixName); err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to remove package: %v", err)
	}
	GroveLogger.Infof("Package removed successfully")

	finalMessage := color.GreenString("✓") + " Removed " + packageName + " from devenv.nix\n" +
		color.CyanString("→") + " Run " + color.YellowString("kanuka grove enter") + " to use the updated environment"

	spinner.FinalMSG = finalMessage
	return nil
}

// handleLanguageRemoval handles the removal of a language from devenv.nix.
func handleLanguageRemoval(languageName string, spinner *spinner.Spinner) error {
	// Check if language exists and is managed by Kanuka
	GroveLogger.Debugf("Checking if language exists in devenv.nix")
	exists, isKanukaManaged, err := grove.DoesLanguageExistInDevenv(languageName)
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to check existing languages: %v", err)
	}

	if !exists {
		finalMessage := color.RedString("✗") + " Language '" + languageName + "' not found in devenv.nix\n" +
			color.CyanString("→") + " Use " + color.YellowString("kanuka grove add "+languageName) + " to add this language"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if !isKanukaManaged {
		finalMessage := color.RedString("✗") + " Language '" + languageName + "' is not managed by Kanuka\n" +
			color.CyanString("→") + " Only Kanuka-managed languages can be removed with this command\n" +
			color.CyanString("→") + " Edit devenv.nix manually to remove languages added outside of Kanuka"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Ask for confirmation
	spinner.Stop()
	GroveLogger.WarnfUser("Remove language '%s' from devenv.nix? (y/N)", languageName)

	var response string
	_, err = fmt.Scanln(&response)
	if err != nil {
		return err
	}

	if response != "y" && response != "Y" {
		finalMessage := color.YellowString("!") + " Language removal cancelled"
		spinner.FinalMSG = finalMessage
		spinner.Restart()
		return nil
	}
	spinner.Restart()

	// Update spinner message for the actual removal step
	spinner.Suffix = " Removing language from devenv.nix..."

	// Remove language from devenv.nix
	GroveLogger.Debugf("Removing language from devenv.nix")
	if err := grove.RemoveLanguageFromDevenv(languageName); err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to remove language: %v", err)
	}
	GroveLogger.Infof("Language removed successfully")

	finalMessage := color.GreenString("✓") + " Removed " + languageName + " language from devenv.nix\n" +
		color.CyanString("→") + " Run " + color.YellowString("kanuka grove enter") + " to use the updated environment"

	spinner.FinalMSG = finalMessage
	return nil
}
