package cmd

import (
	"sort"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var groveListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all Kanuka-managed packages and languages",
	Long: `Display all packages and languages currently managed by Kanuka in your development environment.
This shows only items that were added through Kanuka commands and can be removed using 'kanuka grove remove'.

Examples:
  kanuka grove list                    # Show all managed items
  kanuka grove list --packages-only    # Show only packages
  kanuka grove list --languages-only   # Show only languages`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GroveLogger.Infof("Starting grove list command")
		spinner, cleanup := startGroveSpinner("Scanning development environment...", groveVerbose)
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

		// Get managed packages
		GroveLogger.Debugf("Getting Kanuka-managed packages")
		packages, err := grove.GetKanukaManagedPackages()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to get managed packages: %v", err)
		}

		// Get managed languages
		GroveLogger.Debugf("Getting Kanuka-managed languages")
		languages, err := grove.GetKanukaManagedLanguages()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to get managed languages: %v", err)
		}

		// Check flags for filtering
		packagesOnly, _ := cmd.Flags().GetBool("packages-only")
		languagesOnly, _ := cmd.Flags().GetBool("languages-only")

		// Build the output message
		var finalMessage strings.Builder

		if len(packages) == 0 && len(languages) == 0 {
			finalMessage.WriteString(color.YellowString("!") + " No Kanuka-managed packages or languages found\n")
			finalMessage.WriteString(color.CyanString("→") + " Use " + color.YellowString("kanuka grove add <package>") + " to add packages\n")
			finalMessage.WriteString(color.CyanString("→") + " Use " + color.YellowString("kanuka grove add <language>") + " to add languages")
		} else {
			// Show packages if not filtered out
			if !languagesOnly && len(packages) > 0 {
				finalMessage.WriteString(color.GreenString("✓") + " Kanuka-managed packages:\n")
				
				// Sort packages for consistent output
				sort.Strings(packages)
				for _, pkg := range packages {
					// Remove "pkgs." prefix for cleaner display
					displayName := strings.TrimPrefix(pkg, "pkgs.")
					finalMessage.WriteString(color.CyanString("  • ") + displayName + "\n")
				}
				
				if !packagesOnly && len(languages) > 0 {
					finalMessage.WriteString("\n")
				}
			}

			// Show languages if not filtered out
			if !packagesOnly && len(languages) > 0 {
				finalMessage.WriteString(color.GreenString("✓") + " Kanuka-managed languages:\n")
				
				// Sort languages for consistent output
				sort.Strings(languages)
				for _, lang := range languages {
					finalMessage.WriteString(color.CyanString("  • ") + lang + "\n")
				}
			}

			// Add helpful next steps
			finalMessage.WriteString("\n" + color.CyanString("→") + " Run " + color.YellowString("kanuka grove enter") + " to use this environment\n")
			finalMessage.WriteString(color.CyanString("→") + " Use " + color.YellowString("kanuka grove remove <item>") + " to remove items")
		}

		spinner.FinalMSG = finalMessage.String()
		return nil
	},
}

func init() {
	groveListCmd.Flags().Bool("packages-only", false, "show only packages")
	groveListCmd.Flags().Bool("languages-only", false, "show only languages")
}