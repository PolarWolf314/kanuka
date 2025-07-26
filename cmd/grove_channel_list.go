package cmd

import (
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var groveChannelListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all configured nixpkgs channels",
	Long: `Display all nixpkgs channels currently configured in your Grove environment.

Shows channel names, URLs, and descriptions. Channels are read from the inputs
section of your devenv.yaml file. Only nixpkgs-related inputs are displayed.

Examples:
  kanuka grove channel list                    # Show all channels
  kanuka grove channel list --compact          # Show compact format`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GroveLogger.Infof("Starting grove channel list command")
		spinner, cleanup := startGroveSpinner("Reading channel configuration...", groveVerbose)
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

		// Check if devenv.yaml exists
		GroveLogger.Debugf("Checking if devenv.yaml exists")
		devenvYamlExists, err := grove.DoesDevenvYamlExist()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to check devenv.yaml: %v", err)
		}
		if !devenvYamlExists {
			finalMessage := color.RedString("✗") + " devenv.yaml not found\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka grove init") + " to create it"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Get channels from devenv.yaml
		GroveLogger.Debugf("Reading channels from devenv.yaml")
		channels, err := grove.ListChannels()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to read channels: %v", err)
		}

		// Check for compact flag
		compact, _ := cmd.Flags().GetBool("compact")

		// Build the output message
		var finalMessage strings.Builder

		if len(channels) == 0 {
			finalMessage.WriteString(color.YellowString("!") + " No nixpkgs channels found in devenv.yaml\n")
			finalMessage.WriteString(color.CyanString("→") + " Run " + color.YellowString("kanuka grove init") + " to set up default channels\n")
			finalMessage.WriteString(color.CyanString("→") + " Or add channels manually to devenv.yaml")
		} else {
			if compact {
				// Compact format: just names
				finalMessage.WriteString(color.GreenString("✓") + " Configured channels:\n")
				for _, channel := range channels {
					finalMessage.WriteString(color.CyanString("  • ") + channel.Name + "\n")
				}
			} else {
				// Full format: names, URLs, and descriptions
				finalMessage.WriteString(color.GreenString("✓") + " Configured nixpkgs channels:\n\n")
				
				for i, channel := range channels {
					// Channel name (highlighted)
					finalMessage.WriteString(color.CyanString("  ") + color.YellowString(channel.Name) + "\n")
					
					// Channel URL
					finalMessage.WriteString(color.CyanString("    URL: ") + channel.URL + "\n")
					
					// Channel description with potential warning
					description := channel.Description
					
					// Check if this is an old pinned channel and add warning
					if shouldWarn, ageInfo := shouldWarnAboutPinnedChannel(channel.Name, channel.URL); shouldWarn {
						description = channel.Description + " " + color.RedString("⚠️  "+ageInfo)
					}
					
					finalMessage.WriteString(color.CyanString("    Description: ") + description + "\n")
					
					// Add spacing between channels (except for the last one)
					if i < len(channels)-1 {
						finalMessage.WriteString("\n")
					}
				}
			}

			// Add helpful next steps
			finalMessage.WriteString("\n" + color.CyanString("→") + " Use channels with: " + color.YellowString("kanuka grove add <package> --channel <name>") + "\n")
			finalMessage.WriteString(color.CyanString("→") + " Default channel: " + color.YellowString("unstable") + " (nixpkgs)\n")
			finalMessage.WriteString(color.CyanString("→") + " Stable channel: " + color.YellowString("stable") + " (nixpkgs-stable)")
		}

		spinner.FinalMSG = finalMessage.String()
		return nil
	},
}

func init() {
	groveChannelListCmd.Flags().Bool("compact", false, "show compact format with just channel names")
}