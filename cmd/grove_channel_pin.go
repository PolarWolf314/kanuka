package cmd

import (
	"fmt"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var groveChannelPinCmd = &cobra.Command{
	Use:   "pin <channel-name> <commit-hash>",
	Short: "Pin a channel to a specific commit",
	Long: `Pin a nixpkgs channel to a specific commit hash.

This command creates a new pinned channel that points to a specific commit,
while leaving the original channel unchanged. This allows you to use both
the latest version and a pinned version simultaneously.

The pinned channel will be named using the pattern: <channel>-pinned-<short-hash>

Examples:
  kanuka grove channel pin nixpkgs abc123def456           # Creates nixpkgs-pinned-abc123
  kanuka grove channel pin nixpkgs-stable def456abc123   # Creates nixpkgs-stable-pinned-def456
  
Note: You can find commit hashes at https://github.com/NixOS/nixpkgs/commits/`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		channelName := args[0]
		commitHash := args[1]

		// Create and start spinner
		s := spinner.New(spinner.CharSets[14], 100)
		s.Suffix = " Creating pinned channel..."
		s.Start()
		defer s.Stop()

		// Handle the channel pinning
		if err := handleChannelPin(channelName, commitHash, s); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	groveChannelCmd.AddCommand(groveChannelPinCmd)
}

// handleChannelPin creates a new pinned channel from an existing channel
func handleChannelPin(channelName, commitHash string, spinner *spinner.Spinner) error {
	// Validate inputs
	if channelName == "" {
		finalMessage := color.RedString("✗") + " Channel name cannot be empty\n" +
			color.CyanString("→") + " Use " + color.YellowString("kanuka grove channel list") + " to see available channels\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if commitHash == "" {
		finalMessage := color.RedString("✗") + " Commit hash cannot be empty\n" +
			color.CyanString("→") + " Find commit hashes at https://github.com/NixOS/nixpkgs/commits/\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Validate and normalize commit hash
	normalizedCommit, err := validateAndNormalizeCommit(commitHash)
	if err != nil {
		finalMessage := color.RedString("✗") + " Invalid commit hash: " + err.Error() + "\n" +
			color.CyanString("→") + " Commit hash should be a valid Git SHA (8-40 characters)\n" +
			color.CyanString("→") + " Find valid hashes at https://github.com/NixOS/nixpkgs/commits/\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Check if source channel exists
	GroveLogger.Debugf("Checking if source channel exists: %s", channelName)
	channels, err := grove.ListChannels()
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to list channels: %v", err)
	}

	var sourceChannel *grove.ChannelConfig
	for _, ch := range channels {
		if ch.Name == channelName {
			sourceChannel = &ch
			break
		}
	}

	if sourceChannel == nil {
		finalMessage := color.RedString("✗") + " Source channel '" + channelName + "' not found\n" +
			color.CyanString("→") + " Use " + color.YellowString("kanuka grove channel list") + " to see available channels\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Validate that this is a nixpkgs-based channel
	if !isNixpkgsChannel(sourceChannel.URL) {
		finalMessage := color.RedString("✗") + " Can only pin nixpkgs-based channels\n" +
			color.CyanString("→") + " Channel '" + channelName + "' does not appear to be a nixpkgs channel\n" +
			color.CyanString("→") + " Pinning is only supported for github:NixOS/nixpkgs/* channels\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Generate pinned channel name
	shortHash := normalizedCommit[:8]
	pinnedChannelName := channelName + "-pinned-" + shortHash

	// Check if pinned channel already exists
	existingChannels, err := grove.ListChannels()
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to list existing channels: %v", err)
	}

	for _, ch := range existingChannels {
		if ch.Name == pinnedChannelName {
			finalMessage := color.YellowString("ℹ") + " Pinned channel '" + pinnedChannelName + "' already exists\n" +
				color.CyanString("→") + " Use " + color.YellowString("kanuka grove channel show "+pinnedChannelName) + " to see details\n"
			spinner.FinalMSG = finalMessage
			return nil
		}
	}

	// Verify commit exists (optional - skip for now to avoid hanging)
	GroveLogger.Debugf("Skipping commit verification to avoid API timeout")
	commitInfo := ""

	// Create pinned channel URL
	pinnedURL := "github:NixOS/nixpkgs/" + normalizedCommit

	// Add the pinned channel
	GroveLogger.Debugf("Creating pinned channel: %s -> %s", pinnedChannelName, pinnedURL)
	if err := grove.AddChannel(pinnedChannelName, pinnedURL); err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to create pinned channel: %v", err)
	}

	// Build success message
	var output strings.Builder
	output.WriteString(color.GreenString("✓") + " Created pinned channel '" + pinnedChannelName + "'\n")
	output.WriteString(color.CyanString("→") + " Pinned to commit: " + shortHash)

	if commitInfo != "" {
		output.WriteString(" (" + commitInfo + ")")
	}
	output.WriteString("\n")

	output.WriteString(color.CyanString("→") + " Original channel '" + channelName + "' remains unchanged\n")
	output.WriteString("\n")
	output.WriteString(color.BlueString("💡 Usage:\n"))
	output.WriteString(color.CyanString("   kanuka grove add <package> --channel ") + pinnedChannelName + "\n")
	output.WriteString(color.CyanString("   kanuka grove channel show ") + pinnedChannelName + "\n")

	spinner.FinalMSG = output.String()
	return nil
}

// validateAndNormalizeCommit validates a commit hash and returns the normalized version
func validateAndNormalizeCommit(commitHash string) (string, error) {
	// Remove any whitespace
	commitHash = strings.TrimSpace(commitHash)

	// Check length (Git SHA can be 8-40 characters)
	if len(commitHash) < 8 || len(commitHash) > 40 {
		return "", fmt.Errorf("commit hash must be 8-40 characters long")
	}

	// Check if it's a valid hex string
	for _, char := range commitHash {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return "", fmt.Errorf("commit hash must contain only hexadecimal characters")
		}
	}

	// Convert to lowercase for consistency
	return strings.ToLower(commitHash), nil
}

// isNixpkgsChannel checks if a channel URL is a nixpkgs-based channel
func isNixpkgsChannel(url string) bool {
	return strings.Contains(url, "github:NixOS/nixpkgs") || strings.Contains(url, "github.com/NixOS/nixpkgs")
}

// verifyCommitExists checks if a commit exists in the NixOS/nixpkgs repository
func verifyCommitExists(commitHash string) (bool, string) {
	// Use the existing GitHub API functionality from helpers
	commitInfo, _ := fetchGitHubCommitInfo("NixOS", "nixpkgs", commitHash)

	if commitInfo != "" {
		return true, commitInfo
	}

	return false, ""
}
