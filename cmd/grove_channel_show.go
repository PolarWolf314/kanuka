package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var groveChannelShowCmd = &cobra.Command{
	Use:   "show <channel-name>",
	Short: "Show detailed information about a specific channel",
	Long: `Show detailed information about a specific nixpkgs channel.

This command displays comprehensive information about a channel including its URL,
packages using it, and additional metadata for official channels.

Examples:
  kanuka grove channel show nixpkgs-stable     # Show stable channel details
  kanuka grove channel show custom-elm        # Show custom channel details
  kanuka grove channel show nixpkgs           # Show unstable channel details`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		channelName := args[0]

		// Create and start spinner
		s := spinner.New(spinner.CharSets[14], 100)
		s.Suffix = " Gathering channel information..."
		s.Start()
		defer s.Stop()

		// Handle the channel show
		if err := handleChannelShow(channelName, s); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	groveChannelCmd.AddCommand(groveChannelShowCmd)
}

// handleChannelShow displays detailed information about a specific channel
func handleChannelShow(channelName string, spinner *spinner.Spinner) error {
	// Validate channel name
	if channelName == "" {
		finalMessage := color.RedString("✗") + " Channel name cannot be empty\n" +
			color.CyanString("→") + " Use " + color.YellowString("kanuka grove channel list") + " to see available channels\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Check if channel exists
	GroveLogger.Debugf("Checking if channel exists: %s", channelName)
	channels, err := grove.ListChannels()
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to list channels: %v", err)
	}

	var targetChannel *grove.ChannelConfig
	for _, ch := range channels {
		if ch.Name == channelName {
			targetChannel = &ch
			break
		}
	}

	if targetChannel == nil {
		finalMessage := color.RedString("✗") + " Channel '" + channelName + "' not found\n" +
			color.CyanString("→") + " Use " + color.YellowString("kanuka grove channel list") + " to see available channels\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Get packages using this channel
	GroveLogger.Debugf("Getting packages using channel: %s", channelName)
	packagesUsingChannel, err := getPackagesUsingChannelForShow(channelName)
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to check channel usage: %v", err)
	}

	// Get channel validation info to determine if it's official
	channelInfo := grove.GetChannelValidationInfo(channelName)

	// Build the detailed output
	var output strings.Builder
	
	// Header
	output.WriteString(color.BlueString("📦 Channel Details: ") + color.HiWhiteString(channelName) + "\n\n")
	
	// Basic information (always available)
	output.WriteString(color.CyanString("URL:              ") + targetChannel.URL + "\n")
	
	// Enhanced information for official channels
	if channelInfo.IsOfficial {
		output.WriteString(color.CyanString("Type:             ") + "Official nixpkgs channel\n")
		
		// Try to get additional metadata for official channels
		commitInfo, lastUpdated, status := getOfficialChannelMetadata(targetChannel.URL)
		if commitInfo != "" {
			output.WriteString(color.CyanString("Current Commit:   ") + commitInfo + "\n")
		}
		if lastUpdated != "" {
			output.WriteString(color.CyanString("Last Updated:     ") + lastUpdated + "\n")
		}
		if status != "" {
			output.WriteString(color.CyanString("Status:           ") + status + "\n")
		}
		output.WriteString(color.CyanString("Description:      ") + targetChannel.Description + "\n")
	} else {
		// Simplified information for custom channels
		urlStatus := checkURLAccessibility(targetChannel.URL)
		output.WriteString(color.CyanString("Status:           ") + urlStatus + "\n")
		output.WriteString(color.YellowString("Note:             ") + "Limited metadata available for custom channels\n")
	}
	
	// Package usage information (always shown)
	output.WriteString("\n")
	if len(packagesUsingChannel) > 0 {
		output.WriteString(color.BlueString("📋 Packages using this channel ") + 
			color.HiWhiteString(fmt.Sprintf("(%d)", len(packagesUsingChannel))) + ":\n")
		
		for _, pkg := range packagesUsingChannel {
			// Generate the nix name for display
			var nixName string
			if channelName == "nixpkgs" {
				nixName = "pkgs." + pkg
			} else {
				nixName = "pkgs-" + strings.ReplaceAll(channelName, "-", "_") + "." + pkg
			}
			output.WriteString(fmt.Sprintf("  - %-15s (%s)\n", pkg, color.HiBlackString(nixName)))
		}
		
		output.WriteString("\n" + color.YellowString("💡 Remove these packages before removing the channel:\n"))
		packageList := strings.Join(packagesUsingChannel, " ")
		output.WriteString(color.CyanString("   kanuka grove remove ") + packageList + "\n")
	} else {
		output.WriteString(color.GreenString("✓ No packages currently using this channel\n"))
		// Only show removal message for non-protected channels
		if !isProtectedChannelForShow(channelName) {
			output.WriteString(color.CyanString("→ Channel can be safely removed if no longer needed\n"))
		} else {
			output.WriteString(color.YellowString("→ This is a protected channel required for Grove functionality\n"))
		}
	}
	
	// Usage examples
	output.WriteString("\n" + color.BlueString("💡 Usage:\n"))
	output.WriteString(color.CyanString("   kanuka grove add <package> --channel ") + channelName + "\n")
	
	// For custom channels, provide link to source
	if !channelInfo.IsOfficial && strings.HasPrefix(targetChannel.URL, "github:") {
		// Convert github:owner/repo/branch to https://github.com/owner/repo
		parts := strings.Split(strings.TrimPrefix(targetChannel.URL, "github:"), "/")
		if len(parts) >= 2 {
			githubURL := fmt.Sprintf("https://github.com/%s/%s", parts[0], parts[1])
			output.WriteString(color.CyanString("   View source: ") + githubURL + "\n")
		}
	}

	spinner.FinalMSG = output.String()
	return nil
}

// getOfficialChannelMetadata attempts to get additional metadata for official nixpkgs channels
func getOfficialChannelMetadata(url string) (commitInfo, lastUpdated, status string) {
	// For now, return basic status - this could be enhanced later with GitHub API calls
	status = checkURLAccessibility(url)
	
	// Placeholder for future enhancements
	// TODO: Implement GitHub API calls to get:
	// - Latest commit hash and message
	// - Last updated timestamp
	// - Branch/tag information
	
	return "", "", status
}

// checkURLAccessibility checks if a channel URL is accessible
func checkURLAccessibility(url string) string {
	// For now, just return a basic status
	// TODO: Implement actual URL/Git accessibility checking
	if strings.Contains(url, "github.com") || strings.Contains(url, "github:") {
		return color.GreenString("✓") + " URL format valid"
	}
	return color.YellowString("?") + " Custom URL (not validated)"
}

// isProtectedChannelForShow checks if a channel is protected from removal
func isProtectedChannelForShow(channelName string) bool {
	protectedChannels := map[string]bool{
		"nixpkgs":        true,
		"nixpkgs-stable": true,
	}
	return protectedChannels[channelName]
}

// getPackagesUsingChannelForShow returns a list of packages that are using the specified channel
func getPackagesUsingChannelForShow(channelName string) ([]string, error) {
	// Get the expected package prefix for this channel
	var packagePrefix string
	if channelName == "nixpkgs" {
		packagePrefix = "pkgs."
	} else {
		// Convert channel name to package prefix (e.g., "custom-elm" -> "pkgs-custom_elm.")
		packagePrefix = "pkgs-" + strings.ReplaceAll(channelName, "-", "_") + "."
	}

	// Get all Kanuka-managed packages (if devenv.nix exists)
	packages, err := grove.GetKanukaManagedPackages()
	if err != nil {
		// If devenv.nix doesn't exist, no packages are using any channels
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get managed packages: %w", err)
	}

	var usingChannel []string
	for _, pkg := range packages {
		if strings.HasPrefix(pkg, packagePrefix) {
			// Extract package name from full nix name (e.g., "pkgs-custom_elm.elm" -> "elm")
			parts := strings.Split(pkg, ".")
			if len(parts) >= 2 {
				packageName := parts[len(parts)-1]
				usingChannel = append(usingChannel, packageName)
			}
		}
	}

	return usingChannel, nil
}