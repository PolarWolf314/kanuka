package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var groveChannelAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add a new nixpkgs channel",
	Long: `Add a new nixpkgs channel to your Grove environment.

Channels allow you to use different versions of nixpkgs packages. You can add
custom channels pointing to specific nixpkgs branches, forks, or commits.

The channel URL should follow the format:
  github:owner/repo/branch-or-commit

Examples:
  kanuka grove channel add stable-custom github:MyOrg/nixpkgs/stable
  kanuka grove channel add nixos-23-11 github:NixOS/nixpkgs/nixos-23.11
  kanuka grove channel add my-fork github:username/nixpkgs/my-branch
  kanuka grove channel add pinned github:NixOS/nixpkgs/abc123def456

After adding a channel, use it with:
  kanuka grove add <package> --channel <name>`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		channelName := args[0]
		channelURL := args[1]

		GroveLogger.Infof("Starting grove channel add command for channel '%s'", channelName)
		spinner, cleanup := startGroveSpinner("Adding channel...", groveVerbose)
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

		// Validate channel name
		GroveLogger.Debugf("Validating channel name: %s", channelName)
		if err := validateChannelName(channelName); err != nil {
			finalMessage := color.RedString("✗") + " Invalid channel name: " + err.Error() + "\n" +
				color.CyanString("→") + " Channel names must be alphanumeric with hyphens only"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Validate channel URL
		GroveLogger.Debugf("Validating channel URL: %s", channelURL)
		if err := validateChannelURL(channelURL); err != nil {
			finalMessage := color.RedString("✗") + " Invalid channel URL: " + err.Error() + "\n" +
				color.CyanString("→") + " Use format: github:owner/repo/branch-or-commit"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Check if channel already exists
		GroveLogger.Debugf("Checking if channel already exists")
		channels, err := grove.ListChannels()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to read existing channels: %v", err)
		}

		for _, channel := range channels {
			if channel.Name == channelName {
				finalMessage := color.RedString("✗") + " Channel '" + channelName + "' already exists\n" +
					color.CyanString("→") + " Current URL: " + channel.URL + "\n" +
					color.CyanString("→") + " Use " + color.YellowString("kanuka grove channel remove "+channelName) + " first\n" +
					color.CyanString("→") + " Or use " + color.YellowString("kanuka grove channel show "+channelName) + " to see details"
				spinner.FinalMSG = finalMessage
				return nil
			}
		}

		// Add the channel
		GroveLogger.Debugf("Adding channel to devenv.yaml")
		if err := grove.AddChannel(channelName, channelURL); err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to add channel: %v", err)
		}

		// Build success message
		finalMessage := color.GreenString("✓") + " Added channel '" + color.YellowString(channelName) + "'\n" +
			color.CyanString("→") + " Channel: " + channelURL + "\n" +
			color.CyanString("→") + " Use: " + color.YellowString("kanuka grove add <package> --channel "+channelName) + "\n" +
			color.CyanString("→") + " View: " + color.YellowString("kanuka grove channel list")

		spinner.FinalMSG = finalMessage
		return nil
	},
}

// validateChannelName validates that a channel name follows the required format.
func validateChannelName(name string) error {
	if name == "" {
		return fmt.Errorf("channel name cannot be empty")
	}

	// Check for reserved names
	reservedNames := []string{"nixpkgs", "nixpkgs-stable"}
	for _, reserved := range reservedNames {
		if name == reserved {
			return fmt.Errorf("'%s' is a reserved channel name", name)
		}
	}

	// Validate format: alphanumeric and hyphens only
	validName := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("channel name must contain only letters, numbers, and hyphens")
	}

	// Must not start or end with hyphen
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("channel name cannot start or end with a hyphen")
	}

	// Reasonable length limits
	if len(name) < 2 {
		return fmt.Errorf("channel name must be at least 2 characters long")
	}
	if len(name) > 50 {
		return fmt.Errorf("channel name must be 50 characters or less")
	}

	return nil
}

// validateChannelURL validates that a channel URL follows the required format.
func validateChannelURL(url string) error {
	if url == "" {
		return fmt.Errorf("channel URL cannot be empty")
	}

	// Must start with github:
	if !strings.HasPrefix(url, "github:") {
		return fmt.Errorf("URL must start with 'github:'")
	}

	// Validate github URL format: github:owner/repo/branch-or-commit
	githubPattern := regexp.MustCompile(`^github:[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)
	if !githubPattern.MatchString(url) {
		return fmt.Errorf("URL must follow format: github:owner/repo/branch-or-commit")
	}

	// Extract parts for additional validation
	parts := strings.Split(strings.TrimPrefix(url, "github:"), "/")
	if len(parts) != 3 {
		return fmt.Errorf("URL must have exactly 3 parts: owner/repo/branch-or-commit")
	}

	owner, repo, ref := parts[0], parts[1], parts[2]

	// Validate owner
	if len(owner) == 0 {
		return fmt.Errorf("owner cannot be empty")
	}

	// Validate repo
	if len(repo) == 0 {
		return fmt.Errorf("repository name cannot be empty")
	}

	// Validate reference (branch or commit)
	if len(ref) == 0 {
		return fmt.Errorf("branch or commit reference cannot be empty")
	}

	return nil
}

func init() {
	// No flags needed for basic add functionality
}
