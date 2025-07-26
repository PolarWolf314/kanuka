package cmd

import (
	"github.com/spf13/cobra"
)

var groveChannelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage nixpkgs channels for Grove environment",
	Long: `Manage nixpkgs channels including listing, adding, removing, pinning, and updating channels.

Channels allow you to use different versions of nixpkgs packages in your Grove environment.
By default, Grove provides 'unstable' and 'stable' channels, but you can add custom channels
for specific package versions or custom nixpkgs forks.

Available commands:
  list   - Show all configured channels
  add    - Add new nixpkgs channel (coming soon)
  remove - Remove channel (coming soon)
  show   - Show channel details (coming soon)
  pin    - Pin channel to specific commit (coming soon)
  update - Update channel to latest (coming soon)

Examples:
  kanuka grove channel list                    # Show all channels
  kanuka grove channel add custom-stable github:MyOrg/nixpkgs/stable
  kanuka grove channel show nixpkgs-stable    # Show stable channel details`,
}

func init() {
	groveChannelCmd.AddCommand(groveChannelListCmd)
	groveChannelCmd.AddCommand(groveChannelAddCmd)
	// Future commands will be added here:
	// groveChannelCmd.AddCommand(groveChannelRemoveCmd) // Now implemented in grove_channel_remove.go
	// groveChannelCmd.AddCommand(groveChannelShowCmd)
	// groveChannelCmd.AddCommand(groveChannelPinCmd)
	// groveChannelCmd.AddCommand(groveChannelUpdateCmd)
}
