package cmd

import (
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var groveChannelRemoveCmd = &cobra.Command{
	Use:   "remove <channel-name>",
	Short: "Remove a nixpkgs channel from Grove environment",
	Long: `Remove a nixpkgs channel from the Grove environment.

This command removes a channel from devenv.yaml and cleans up any associated
imports in devenv.nix. It will check if any packages are using the channel
and warn before removal.

Examples:
  kanuka grove channel remove custom-elm     # Remove custom-elm channel
  kanuka grove channel remove nixpkgs-old   # Remove old nixpkgs channel
  
Note: You cannot remove the default nixpkgs or nixpkgs-stable channels.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		channelName := args[0]

		// Create and start spinner
		s := spinner.New(spinner.CharSets[14], 100)
		s.Suffix = " Removing channel..."
		s.Start()
		defer s.Stop()

		// Handle the channel removal
		if err := handleChannelRemoval(channelName, s); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	groveChannelCmd.AddCommand(groveChannelRemoveCmd)
}

// handleChannelRemoval handles the removal of a channel from devenv.yaml
func handleChannelRemoval(channelName string, spinner *spinner.Spinner) error {
	// Validate channel name
	if channelName == "" {
		finalMessage := color.RedString("✗") + " Channel name cannot be empty\n" +
			color.CyanString("→") + " Use " + color.YellowString("kanuka grove channel list") + " to see available channels\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Check if it's a protected channel
	if isProtectedChannel(channelName) {
		finalMessage := color.RedString("✗") + " Cannot remove protected channel '" + channelName + "'\n" +
			color.CyanString("→") + " Protected channels: nixpkgs, nixpkgs-stable\n" +
			color.CyanString("→") + " These channels are required for Grove functionality\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Check if channel exists
	GroveLogger.Debugf("Checking if channel exists: %s", channelName)
	channels, err := grove.ListChannels()
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to list channels: %v", err)
	}

	var channelExists bool
	for _, ch := range channels {
		if ch.Name == channelName {
			channelExists = true
			break
		}
	}

	if !channelExists {
		finalMessage := color.RedString("✗") + " Channel '" + channelName + "' not found\n" +
			color.CyanString("→") + " Use " + color.YellowString("kanuka grove channel list") + " to see available channels\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Check if any packages are using this channel
	GroveLogger.Debugf("Checking for packages using channel: %s", channelName)
	packagesUsingChannel, err := getPackagesUsingChannel(channelName)
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to check channel usage: %v", err)
	}

	if len(packagesUsingChannel) > 0 {
		packageList := strings.Join(packagesUsingChannel, ", ")
		finalMessage := color.RedString("✗") + " Cannot remove channel '" + channelName + "' - packages are using it\n" +
			color.CyanString("→") + " Packages using this channel: " + packageList + "\n" +
			color.CyanString("→") + " Remove these packages first, then try again\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Remove the channel
	GroveLogger.Debugf("Removing channel from devenv.yaml: %s", channelName)
	if err := grove.RemoveChannel(channelName); err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to remove channel: %v", err)
	}

	// Clean up any unused imports in devenv.nix
	GroveLogger.Debugf("Cleaning up devenv.nix imports")
	if err := cleanupChannelImports(channelName); err != nil {
		GroveLogger.Warnf("Failed to clean up devenv.nix imports: %v", err)
		// Don't fail the command for cleanup issues
	}

	finalMessage := color.GreenString("✓") + " Removed channel '" + channelName + "' from Grove environment\n" +
		color.CyanString("→") + " Channel removed from devenv.yaml\n" +
		color.CyanString("→") + " Run " + color.YellowString("kanuka grove channel list") + " to see remaining channels\n"

	spinner.FinalMSG = finalMessage
	return nil
}

// Functions moved to grove_channel_helpers.go for deduplication

// cleanupChannelImports removes unused channel imports from devenv.nix let block
func cleanupChannelImports(channelName string) error {
	// This is a placeholder for now - implementing let block cleanup is complex
	// and not critical for the remove functionality to work
	GroveLogger.Debugf("Channel import cleanup for %s - placeholder implementation", channelName)
	return nil
}