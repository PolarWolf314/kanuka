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

Examples:
  kanuka grove channel list                    # Show all channels
  kanuka grove channel add custom-stable github:MyOrg/nixpkgs/stable
  kanuka grove channel show nixpkgs-stable    # Show stable channel details`,
}

func init() {
	groveChannelCmd.AddCommand(groveChannelListCmd)
	groveChannelCmd.AddCommand(groveChannelAddCmd)
	groveChannelCmd.AddCommand(groveChannelRemoveCmd)
	groveChannelCmd.AddCommand(groveChannelShowCmd)
	groveChannelCmd.AddCommand(groveChannelPinCmd)
	groveChannelCmd.AddCommand(groveChannelUpdateCmd)
}
