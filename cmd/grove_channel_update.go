package cmd

import (
	"fmt"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	updateAll    bool
	updatePinned bool
	dryRun       bool
	interactive  bool
)

var groveChannelUpdateCmd = &cobra.Command{
	Use:   "update <channel-name> [commit-hash]",
	Short: "Update channels to their latest versions",
	Long: `Update nixpkgs channels to their latest versions.

This command updates channels based on their type:
- Official channels: Updates to latest stable releases
- Pinned channels: Updates to latest commit (or specified commit)
- Custom channels: Cannot be auto-updated (skipped with message)

Examples:
  kanuka grove channel update nixpkgs-stable              # Update to latest stable release
  kanuka grove channel update nixpkgs-pinned-abc123      # Update pinned channel to latest
  kanuka grove channel update nixpkgs-pinned-abc123 def456  # Update pinned channel to specific commit
  kanuka grove channel update --all                      # Update all updatable channels
  kanuka grove channel update --pinned-only              # Update only pinned channels
  kanuka grove channel update nixpkgs-stable --dry-run   # Preview changes without applying`,
	Args: func(cmd *cobra.Command, args []string) error {
		if updateAll || updatePinned {
			if len(args) > 0 {
				return fmt.Errorf("cannot specify channel name with --all or --pinned-only flags")
			}
			return nil
		}
		if len(args) < 1 {
			return fmt.Errorf("channel name is required (or use --all/--pinned-only)")
		}
		if len(args) > 2 {
			return fmt.Errorf("too many arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create and start spinner
		s := spinner.New(spinner.CharSets[14], 100)
		s.Suffix = " Updating channels..."
		s.Start()
		defer s.Stop()

		// Handle different update modes
		if updateAll {
			return handleUpdateAll(s)
		} else if updatePinned {
			return handleUpdatePinnedOnly(s)
		} else {
			channelName := args[0]
			var commitHash string
			if len(args) > 1 {
				commitHash = args[1]
			}
			return handleUpdateSingle(channelName, commitHash, s)
		}
	},
}

func init() {
	groveChannelUpdateCmd.Flags().BoolVar(&updateAll, "all", false, "Update all updatable channels")
	groveChannelUpdateCmd.Flags().BoolVar(&updatePinned, "pinned-only", false, "Update only pinned channels")
	groveChannelUpdateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying them")
	groveChannelUpdateCmd.Flags().BoolVar(&interactive, "interactive", false, "Prompt for each channel update")
}

// UpdateBehavior defines how a channel should be updated.
type UpdateBehavior struct {
	ChannelType   string // "official", "pinned", "custom"
	UpdateMethod  string // "branch", "commit", "manual"
	CanAutoUpdate bool
}

// getUpdateBehavior determines how a channel should be updated.
func getUpdateBehavior(channel grove.ChannelConfig) UpdateBehavior {
	if isPinnedChannel(channel.Name) {
		return UpdateBehavior{
			ChannelType:   "pinned",
			UpdateMethod:  "commit",
			CanAutoUpdate: true,
		}
	}

	if isOfficialNixpkgsChannel(channel.URL) {
		return UpdateBehavior{
			ChannelType:   "official",
			UpdateMethod:  "branch",
			CanAutoUpdate: true,
		}
	}

	return UpdateBehavior{
		ChannelType:   "custom",
		UpdateMethod:  "manual",
		CanAutoUpdate: false,
	}
}

// handleUpdateSingle updates a single channel.
func handleUpdateSingle(channelName, commitHash string, spinner *spinner.Spinner) error {
	// Get channel details
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

	// Determine update behavior
	behavior := getUpdateBehavior(*targetChannel)

	if !behavior.CanAutoUpdate {
		finalMessage := color.YellowString("ℹ") + " Cannot auto-update custom channel '" + channelName + "'\n" +
			color.CyanString("→") + " Custom channels must be updated manually\n" +
			color.CyanString("→") + " Use " + color.YellowString("kanuka grove channel remove") + " and " + color.YellowString("kanuka grove channel add") + " to update\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Perform the update
	return performChannelUpdate(*targetChannel, commitHash, behavior, spinner)
}

// handleUpdateAll updates all updatable channels.
func handleUpdateAll(spinner *spinner.Spinner) error {
	channels, err := grove.ListChannels()
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to list channels: %v", err)
	}

	var results []string
	var updatedCount, skippedCount int

	for _, channel := range channels {
		behavior := getUpdateBehavior(channel)

		if !behavior.CanAutoUpdate {
			results = append(results, color.YellowString("⚠️ ")+channel.Name+": skipped (custom channel)")
			skippedCount++
			continue
		}

		// Check if update is needed
		updateNeeded, newURL, err := checkUpdateNeeded(channel, behavior)
		if err != nil {
			results = append(results, color.RedString("✗ ")+channel.Name+": error checking for updates")
			continue
		}

		if !updateNeeded {
			results = append(results, color.GreenString("✓ ")+channel.Name+": already latest")
			continue
		}

		if dryRun {
			results = append(results, color.CyanString("→ ")+channel.Name+": would update to "+extractVersionFromURL(newURL))
			continue
		}

		// Perform update
		if err := grove.UpdateChannelURL(channel.Name, newURL); err != nil {
			results = append(results, color.RedString("✗ ")+channel.Name+": update failed")
		} else {
			results = append(results, color.GreenString("✓ ")+channel.Name+": updated to "+extractVersionFromURL(newURL))
			updatedCount++
		}
	}

	// Build final message
	var output strings.Builder
	if dryRun {
		output.WriteString(color.BlueString("📋 Update Preview (--dry-run)\n\n"))
	} else {
		output.WriteString(color.BlueString("📋 Channel Update Results\n\n"))
	}

	for _, result := range results {
		output.WriteString(result + "\n")
	}

	output.WriteString("\n")
	if dryRun {
		output.WriteString(color.CyanString("→ Run without --dry-run to apply changes\n"))
	} else {
		output.WriteString(fmt.Sprintf("Updated: %d, Skipped: %d\n", updatedCount, skippedCount))
		if updatedCount > 0 {
			output.WriteString(color.CyanString("→ Run ") + color.YellowString("kanuka grove enter") + color.CyanString(" to use updated packages\n"))
		}
	}

	spinner.FinalMSG = output.String()
	return nil
}

// handleUpdatePinnedOnly updates only pinned channels.
func handleUpdatePinnedOnly(spinner *spinner.Spinner) error {
	channels, err := grove.ListChannels()
	if err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to list channels: %v", err)
	}

	var pinnedChannels []grove.ChannelConfig
	for _, channel := range channels {
		if isPinnedChannel(channel.Name) {
			pinnedChannels = append(pinnedChannels, channel)
		}
	}

	if len(pinnedChannels) == 0 {
		finalMessage := color.YellowString("ℹ") + " No pinned channels found\n" +
			color.CyanString("→") + " Use " + color.YellowString("kanuka grove channel pin") + " to create pinned channels\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	var results []string
	var updatedCount int

	for _, channel := range pinnedChannels {
		behavior := getUpdateBehavior(channel)

		// Check if update is needed
		updateNeeded, newURL, err := checkUpdateNeeded(channel, behavior)
		if err != nil {
			results = append(results, color.RedString("✗ ")+channel.Name+": error checking for updates")
			continue
		}

		if !updateNeeded {
			results = append(results, color.GreenString("✓ ")+channel.Name+": already latest")
			continue
		}

		if dryRun {
			results = append(results, color.CyanString("→ ")+channel.Name+": would update to latest commit")
			continue
		}

		// Perform update
		if err := grove.UpdateChannelURL(channel.Name, newURL); err != nil {
			results = append(results, color.RedString("✗ ")+channel.Name+": update failed")
		} else {
			results = append(results, color.GreenString("✓ ")+channel.Name+": updated to latest commit")
			updatedCount++
		}
	}

	// Build final message
	var output strings.Builder
	if dryRun {
		output.WriteString(color.BlueString("📋 Pinned Channel Update Preview\n\n"))
	} else {
		output.WriteString(color.BlueString("📋 Pinned Channel Update Results\n\n"))
	}

	for _, result := range results {
		output.WriteString(result + "\n")
	}

	output.WriteString("\n")
	if dryRun {
		output.WriteString(color.CyanString("→ Run without --dry-run to apply changes\n"))
	} else {
		output.WriteString(fmt.Sprintf("Updated: %d pinned channels\n", updatedCount))
		if updatedCount > 0 {
			output.WriteString(color.CyanString("→ Run ") + color.YellowString("kanuka grove enter") + color.CyanString(" to use updated packages\n"))
		}
	}

	spinner.FinalMSG = output.String()
	return nil
}

// performChannelUpdate performs the actual update for a single channel.
func performChannelUpdate(channel grove.ChannelConfig, commitHash string, behavior UpdateBehavior, spinner *spinner.Spinner) error {
	// Check if update is needed
	updateNeeded, newURL, err := checkUpdateNeeded(channel, behavior)
	if err != nil {
		finalMessage := color.RedString("✗") + " Failed to check for updates: " + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// If specific commit hash provided for pinned channel, use that
	if commitHash != "" && behavior.ChannelType == "pinned" {
		normalizedCommit, err := validateAndNormalizeCommit(commitHash)
		if err != nil {
			finalMessage := color.RedString("✗") + " Invalid commit hash: " + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return nil
		}
		newURL = "github:NixOS/nixpkgs/" + normalizedCommit
		updateNeeded = true // Force update with specific commit
	}

	if !updateNeeded {
		finalMessage := color.GreenString("✓") + " Channel '" + channel.Name + "' is already up to date\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if dryRun {
		var output strings.Builder
		output.WriteString(color.BlueString("📋 Update Preview for ") + color.HiWhiteString(channel.Name) + "\n\n")
		output.WriteString(color.CyanString("From: ") + channel.URL + "\n")
		output.WriteString(color.CyanString("To:   ") + newURL + "\n\n")

		// Show affected packages
		packagesUsing, err := getPackagesUsingChannel(channel.Name)
		if err == nil && len(packagesUsing) > 0 {
			output.WriteString(color.BlueString("📦 Affected packages (") + fmt.Sprintf("%d", len(packagesUsing)) + "):\n")
			for _, pkg := range packagesUsing {
				output.WriteString(fmt.Sprintf("  - %s\n", pkg))
			}
			output.WriteString("\n")
		}

		output.WriteString(color.CyanString("→ Run without --dry-run to apply changes\n"))
		spinner.FinalMSG = output.String()
		return nil
	}

	// Perform the actual update
	GroveLogger.Debugf("Updating channel %s from %s to %s", channel.Name, channel.URL, newURL)
	if err := grove.UpdateChannelURL(channel.Name, newURL); err != nil {
		return GroveLogger.ErrorfAndReturn("Failed to update channel: %v", err)
	}

	// Build success message
	var output strings.Builder
	output.WriteString(color.GreenString("✓") + " Updated channel '" + channel.Name + "'\n")

	if behavior.ChannelType == "pinned" {
		output.WriteString(color.CyanString("→") + " Pinned channel now points to latest commit\n")
	} else {
		oldVersion := extractVersionFromURL(channel.URL)
		newVersion := extractVersionFromURL(newURL)
		if oldVersion != newVersion {
			output.WriteString(color.CyanString("→") + " Updated from " + oldVersion + " to " + newVersion + "\n")
		}
	}

	// Show affected packages
	packagesUsing, err := getPackagesUsingChannel(channel.Name)
	if err == nil && len(packagesUsing) > 0 {
		output.WriteString(color.CyanString("→") + fmt.Sprintf(" %d packages using this channel: %s\n",
			len(packagesUsing), strings.Join(packagesUsing, ", ")))
	}

	output.WriteString(color.CyanString("→") + " Run " + color.YellowString("kanuka grove enter") + " to use updated packages\n")

	spinner.FinalMSG = output.String()
	return nil
}
