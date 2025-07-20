package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var groveStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show development environment status",
	Long: `Display comprehensive status information about your development environment.
Shows project details, managed packages and languages, environment health, and helpful diagnostics.

Examples:
  kanuka grove status                  # Show full environment status
  kanuka grove status --compact        # Show compact status summary`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GroveLogger.Infof("Starting grove status command")
		spinner, cleanup := startGroveSpinner("Gathering environment status...", groveVerbose)
		defer cleanup()

		compact, _ := cmd.Flags().GetBool("compact")

		// Build status information
		status, err := gatherEnvironmentStatus()
		if err != nil {
			return GroveLogger.ErrorfAndReturn("Failed to gather status: %v", err)
		}

		// Format and display status
		var finalMessage strings.Builder

		if compact {
			finalMessage.WriteString(formatCompactStatus(status))
		} else {
			finalMessage.WriteString(formatDetailedStatus(status))
		}

		spinner.FinalMSG = finalMessage.String()
		return nil
	},
}

// EnvironmentStatus holds all status information.
type EnvironmentStatus struct {
	// Project information
	IsGroveProject bool
	ProjectName    string
	ProjectID      string
	ProjectPath    string

	// File status
	HasKanukaToml bool
	HasDevenvNix  bool
	HasDotenv     bool

	// Managed items
	ManagedPackages  []string
	ManagedLanguages []string

	// Environment health
	DevenvInstalled bool
	DevenvVersion   string
	NixInstalled    bool

	// AWS SSO status
	AWSSSoConfigured bool
	AWSSSoProfile    string
	AWSAuthenticated bool

	// Errors
	Errors []string
}

// gatherEnvironmentStatus collects all environment status information.
func gatherEnvironmentStatus() (*EnvironmentStatus, error) {
	status := &EnvironmentStatus{
		Errors: make([]string, 0),
	}

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		status.Errors = append(status.Errors, fmt.Sprintf("Failed to get current directory: %v", err))
		return status, nil
	}
	status.ProjectPath = currentDir
	status.ProjectName = filepath.Base(currentDir)

	// Check if this is a grove project
	kanukaExists, err := grove.DoesKanukaTomlExist()
	if err != nil {
		status.Errors = append(status.Errors, fmt.Sprintf("Failed to check kanuka.toml: %v", err))
	} else {
		status.HasKanukaToml = kanukaExists
		status.IsGroveProject = kanukaExists
	}

	// Check devenv.nix
	devenvExists, err := grove.DoesDevenvNixExist()
	if err != nil {
		status.Errors = append(status.Errors, fmt.Sprintf("Failed to check devenv.nix: %v", err))
	} else {
		status.HasDevenvNix = devenvExists
	}

	// Check .env file
	dotenvPath := filepath.Join(currentDir, ".env")
	if _, err := os.Stat(dotenvPath); err == nil {
		status.HasDotenv = true
	}

	// Get project ID if available
	if status.HasKanukaToml {
		if projectID, err := getProjectIDFromToml(); err == nil {
			status.ProjectID = projectID
		}
	}

	// Get managed packages and languages (only if in grove project)
	if status.IsGroveProject && status.HasDevenvNix {
		if packages, err := grove.GetKanukaManagedPackages(); err == nil {
			status.ManagedPackages = packages
		} else {
			status.Errors = append(status.Errors, fmt.Sprintf("Failed to get managed packages: %v", err))
		}

		if languages, err := grove.GetKanukaManagedLanguages(); err == nil {
			status.ManagedLanguages = languages
		} else {
			status.Errors = append(status.Errors, fmt.Sprintf("Failed to get managed languages: %v", err))
		}
	}

	// Check devenv installation
	if devenvPath, err := exec.LookPath("devenv"); err == nil {
		status.DevenvInstalled = true
		// Try to get devenv version
		if out, err := exec.Command(devenvPath, "--version").Output(); err == nil {
			status.DevenvVersion = strings.TrimSpace(string(out))
		}
	}

	// Check nix installation
	if _, err := exec.LookPath("nix"); err == nil {
		status.NixInstalled = true
	}

	// Check AWS SSO status (using official AWS Go SDK, no external dependencies)
	if ssoConfig, err := findAWSSSoConfigForStatus(); err == nil {
		status.AWSSSoConfigured = true
		status.AWSSSoProfile = ssoConfig.ProfileName

		// Check if authenticated using official AWS Go SDK
		status.AWSAuthenticated = isAWSSSoAuthenticatedForStatus(ssoConfig)
	}

	return status, nil
}

// getProjectIDFromToml extracts project ID from kanuka.toml.
func getProjectIDFromToml() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	tomlPath := filepath.Join(currentDir, "kanuka.toml")
	content, err := os.ReadFile(tomlPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "id = ") {
			// Extract ID from 'id = "value"'
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	return "", fmt.Errorf("project ID not found")
}

// formatDetailedStatus formats comprehensive status information.
func formatDetailedStatus(status *EnvironmentStatus) string {
	var output strings.Builder

	// Header
	output.WriteString(color.CyanString("═══ Kanuka Grove Status ═══\n\n"))

	// Project Information
	output.WriteString(color.YellowString("Project Information\n"))
	if status.IsGroveProject {
		output.WriteString(fmt.Sprintf("   %s Grove project: %s\n", color.GreenString("✓"), color.WhiteString(status.ProjectName)))
		if status.ProjectID != "" {
			output.WriteString(fmt.Sprintf("   %s Project ID: %s\n", color.CyanString("→"), color.WhiteString(status.ProjectID[:8]+"...")))
		}
		output.WriteString(fmt.Sprintf("   %s Location: %s\n", color.CyanString("→"), color.WhiteString(status.ProjectPath)))
	} else {
		output.WriteString(fmt.Sprintf("   %s Not a Grove project\n", color.RedString("✗")))
		output.WriteString(fmt.Sprintf("   %s Run %s to initialize\n", color.CyanString("→"), color.YellowString("kanuka grove init")))
	}

	// File Status
	output.WriteString(fmt.Sprintf("\n%s Configuration Files\n", color.YellowString("Configuration Files")))
	if status.HasKanukaToml {
		output.WriteString(fmt.Sprintf("   %s kanuka.toml\n", color.GreenString("✓")))
	} else {
		output.WriteString(fmt.Sprintf("   %s kanuka.toml (missing)\n", color.RedString("✗")))
	}

	if status.HasDevenvNix {
		output.WriteString(fmt.Sprintf("   %s devenv.nix\n", color.GreenString("✓")))
	} else {
		output.WriteString(fmt.Sprintf("   %s devenv.nix (missing)\n", color.RedString("✗")))
	}

	if status.HasDotenv {
		output.WriteString(fmt.Sprintf("   %s .env (dotenv integration enabled)\n", color.GreenString("✓")))
	} else {
		output.WriteString(fmt.Sprintf("   %s .env (not found)\n", color.YellowString("!")))
	}

	// Managed Items
	if status.IsGroveProject {
		output.WriteString(fmt.Sprintf("\n%s Managed Items\n", color.YellowString("Managed Items")))

		if len(status.ManagedPackages) > 0 {
			output.WriteString(fmt.Sprintf("   %s Packages (%d):\n", color.GreenString("✓"), len(status.ManagedPackages)))
			sort.Strings(status.ManagedPackages)
			for _, pkg := range status.ManagedPackages {
				displayName := strings.TrimPrefix(pkg, "pkgs.")
				output.WriteString(fmt.Sprintf("     %s %s\n", color.CyanString("•"), displayName))
			}
		} else {
			output.WriteString(fmt.Sprintf("   %s No managed packages\n", color.YellowString("!")))
		}

		if len(status.ManagedLanguages) > 0 {
			output.WriteString(fmt.Sprintf("   %s Languages (%d):\n", color.GreenString("✓"), len(status.ManagedLanguages)))
			sort.Strings(status.ManagedLanguages)
			for _, lang := range status.ManagedLanguages {
				output.WriteString(fmt.Sprintf("     %s %s\n", color.CyanString("•"), lang))
			}
		} else {
			output.WriteString(fmt.Sprintf("   %s No managed languages\n", color.YellowString("!")))
		}
	}

	// Environment Health
	output.WriteString(fmt.Sprintf("\n%s Environment Health\n", color.YellowString("Environment Health")))

	if status.NixInstalled {
		output.WriteString(fmt.Sprintf("   %s Nix package manager\n", color.GreenString("✓")))
	} else {
		output.WriteString(fmt.Sprintf("   %s Nix package manager (not installed)\n", color.RedString("✗")))
	}

	if status.DevenvInstalled {
		output.WriteString(fmt.Sprintf("   %s devenv", color.GreenString("✓")))
		if status.DevenvVersion != "" {
			output.WriteString(fmt.Sprintf(" (%s)", color.WhiteString(status.DevenvVersion)))
		}
		output.WriteString("\n")
	} else {
		output.WriteString(fmt.Sprintf("   %s devenv (not installed)\n", color.RedString("✗")))
		output.WriteString(fmt.Sprintf("   %s Install: %s\n", color.CyanString("→"), color.YellowString("nix profile install nixpkgs#devenv")))
	}

	// AWS SSO Status
	if status.AWSSSoConfigured {
		if status.AWSAuthenticated {
			output.WriteString(fmt.Sprintf("   %s AWS SSO (%s) - authenticated\n", color.GreenString("✓"), color.WhiteString(status.AWSSSoProfile)))
		} else {
			output.WriteString(fmt.Sprintf("   %s AWS SSO (%s) - not authenticated\n", color.YellowString("!"), color.WhiteString(status.AWSSSoProfile)))
			output.WriteString(fmt.Sprintf("   %s Authenticate: %s\n", color.CyanString("→"), color.YellowString("kanuka grove enter --auth")))
		}
	} else {
		output.WriteString(fmt.Sprintf("   %s AWS SSO (not configured)\n", color.YellowString("!")))
		output.WriteString(fmt.Sprintf("   %s Configure: %s\n", color.CyanString("→"), color.YellowString("Configure AWS SSO in ~/.aws/config")))
	}

	// Errors
	if len(status.Errors) > 0 {
		output.WriteString(fmt.Sprintf("\n%s Issues\n", color.RedString("Issues")))
		for _, err := range status.Errors {
			output.WriteString(fmt.Sprintf("   %s %s\n", color.RedString("✗"), err))
		}
	}

	// Next Steps
	if status.IsGroveProject {
		output.WriteString(fmt.Sprintf("\n%s Next Steps\n", color.YellowString("Next Steps")))
		if len(status.ManagedPackages) == 0 && len(status.ManagedLanguages) == 0 {
			output.WriteString(fmt.Sprintf("   %s Add packages: %s\n", color.CyanString("→"), color.YellowString("kanuka grove add <package>")))
			output.WriteString(fmt.Sprintf("   %s Add languages: %s\n", color.CyanString("→"), color.YellowString("kanuka grove add <language>")))
		}
		if status.DevenvInstalled {
			output.WriteString(fmt.Sprintf("   %s Enter environment: %s\n", color.CyanString("→"), color.YellowString("kanuka grove enter")))
		}
		output.WriteString(fmt.Sprintf("   %s View managed items: %s\n", color.CyanString("→"), color.YellowString("kanuka grove list")))
	}

	return output.String()
}

// formatCompactStatus formats a brief status summary.
func formatCompactStatus(status *EnvironmentStatus) string {
	var output strings.Builder

	if status.IsGroveProject {
		output.WriteString(fmt.Sprintf("%s %s", color.GreenString("✓"), color.WhiteString(status.ProjectName)))

		itemCount := len(status.ManagedPackages) + len(status.ManagedLanguages)
		if itemCount > 0 {
			output.WriteString(fmt.Sprintf(" (%d items)", itemCount))
		}

		if !status.DevenvInstalled {
			output.WriteString(fmt.Sprintf(" %s", color.RedString("[devenv missing]")))
		}
	} else {
		output.WriteString(fmt.Sprintf("%s Not a Grove project", color.RedString("✗")))
	}

	if len(status.Errors) > 0 {
		output.WriteString(fmt.Sprintf(" %s", color.RedString("[errors]")))
	}

	output.WriteString("\n")
	return output.String()
}

// findAWSSSoConfigForStatus is a simplified version for status checking.
func findAWSSSoConfigForStatus() (*AWSSSoConfigForStatus, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".aws", "config")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var currentProfile string
	var ssoConfig *AWSSSoConfigForStatus

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
			currentProfile = strings.TrimSuffix(strings.TrimPrefix(line, "[profile "), "]")
			continue
		}

		if line == "[default]" {
			currentProfile = "default"
			continue
		}

		if currentProfile != "" && strings.Contains(line, "sso_start_url") {
			if ssoConfig == nil {
				ssoConfig = &AWSSSoConfigForStatus{ProfileName: currentProfile}
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				ssoConfig.SSOStartURL = strings.TrimSpace(parts[1])
			}
		}

		if currentProfile != "" && strings.Contains(line, "sso_region") {
			if ssoConfig != nil {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					ssoConfig.SSORegion = strings.TrimSpace(parts[1])
				}
			}
		}

		if currentProfile != "" && strings.Contains(line, "region") && !strings.Contains(line, "sso_region") {
			if ssoConfig != nil {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					ssoConfig.Region = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	if ssoConfig == nil {
		return nil, fmt.Errorf("no AWS SSO configuration found")
	}

	return ssoConfig, nil
}

// AWSSSoConfigForStatus holds minimal AWS SSO config for status checking.
type AWSSSoConfigForStatus struct {
	ProfileName string
	SSOStartURL string
	SSORegion   string
	Region      string
}

// isAWSSSoAuthenticatedForStatus checks authentication status using official AWS Go SDK.
func isAWSSSoAuthenticatedForStatus(config *AWSSSoConfigForStatus) bool {
	// Use the same authentication check as the main enter command
	mainConfig := &AWSSSoConfig{
		ProfileName: config.ProfileName,
		SSOStartURL: config.SSOStartURL,
		SSORegion:   config.SSORegion,
		Region:      config.Region,
	}

	return isAWSSSoAuthenticated(mainConfig)
}

func init() {
	groveStatusCmd.Flags().Bool("compact", false, "show compact status summary")
}
