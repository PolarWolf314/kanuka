package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/briandowns/spinner"
	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/synfinatic/aws-sso-cli/sso"
)

var (
	enterAuth bool
	enterEnv  string
)

var groveEnterCmd = &cobra.Command{
	Use:   "enter",
	Short: "Enter the development shell environment",
	Long: `Enter the development shell environment using devenv with --clean flag.
This starts a clean, isolated shell with all your configured packages and languages available,
ensuring the environment doesn't depend on any system-specific configuration.

Authentication is handled entirely by synfinatic/aws-sso-cli - no AWS CLI dependency required.
When --auth is used, you will always be prompted to authenticate for this session only.

Examples:
  kanuka grove enter                   # Enter clean development shell
  kanuka grove enter --auth            # Always prompt for AWS SSO authentication (session-only)
  kanuka grove enter --env production  # Enter clean shell with production environment`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GroveLogger.Infof("Starting grove enter command")
		spinner, cleanup := startGroveSpinner("Preparing development environment...", groveVerbose)
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

		// Check if devenv is available
		GroveLogger.Debugf("Checking if devenv command is available")
		_, err = exec.LookPath("devenv")
		if err != nil {
			finalMessage := color.RedString("✗") + " devenv command not found\n" +
				color.CyanString("→") + " Install devenv: " + color.YellowString("nix profile install nixpkgs#devenv") + "\n" +
				color.CyanString("→") + " Or visit: " + color.BlueString("https://devenv.sh/getting-started/")
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Handle authentication if requested
		if enterAuth {
			GroveLogger.Debugf("Authentication requested")
			spinner.Suffix = " Setting up authentication..."

			err := handleAuthentication(spinner)
			if err != nil {
				// Handle authentication errors gracefully with spinner
				finalMessage := formatAuthenticationError(err)
				spinner.FinalMSG = finalMessage
				return nil
			}
		}

		// Handle named environment if specified
		if enterEnv != "" {
			GroveLogger.Debugf("Named environment requested: %s", enterEnv)
			spinner.Suffix = " Loading environment: " + enterEnv + "..."

			err := handleNamedEnvironment(enterEnv)
			if err != nil {
				return GroveLogger.ErrorfAndReturn("Failed to load environment '%s': %v", enterEnv, err)
			}
		}

		// Stop spinner before entering shell
		spinner.Stop()

		// Display Kanuka ASCII art using go-figure
		fmt.Println()
		myFigure := figure.NewColorFigure("Kanuka", "alligator2", "green", true)
		myFigure.Print()
		fmt.Println()

		// Show entry message
		fmt.Printf("%s Entering development environment...\n", color.GreenString("✓"))
		if enterAuth {
			fmt.Printf("%s Authentication enabled\n", color.CyanString("→"))
		}
		if enterEnv != "" {
			fmt.Printf("%s Environment: %s\n", color.CyanString("→"), color.YellowString(enterEnv))
		}
		fmt.Printf("%s Type %s to exit\n\n", color.CyanString("→"), color.YellowString("exit"))

		// Enter the devenv shell
		GroveLogger.Debugf("Executing devenv shell")
		return enterDevenvShell()
	},
}

// enterDevenvShell executes the devenv shell command with --clean flag.
func enterDevenvShell() error {
	// Find devenv executable
	devenvPath, err := exec.LookPath("devenv")
	if err != nil {
		return fmt.Errorf("devenv command not found: %w", err)
	}

	// Prepare the command with --clean flag for isolated environment
	args := []string{"devenv", "shell", "--clean"}

	// Execute devenv shell --clean, replacing the current process
	GroveLogger.Debugf("Executing: %s %v", devenvPath, args[1:])
	err = syscall.Exec(devenvPath, args, os.Environ())
	if err != nil {
		return fmt.Errorf("failed to execute devenv shell --clean: %w", err)
	}

	// This line should never be reached if syscall.Exec succeeds
	return nil
}

// handleAuthentication sets up AWS SSO authentication using synfinatic/aws-sso-cli.
func handleAuthentication(spinner *spinner.Spinner) error {
	GroveLogger.Debugf("Setting up AWS SSO authentication using synfinatic/aws-sso-cli")

	// Always prompt for authentication when --auth is used
	// First, try to get SSO config from ~/.aws/config if it exists
	ssoConfig, configErr := findAWSSSoConfig()

	// If no config found, prompt for SSO details interactively
	if configErr != nil {
		GroveLogger.Infof("No AWS SSO configuration found, prompting for details...")
		spinner.Stop() // Stop spinner for interactive input

		var err error
		ssoConfig, err = promptForSSOConfig()
		if err != nil {
			return &AuthenticationError{
				Type:       "sso_config_input_failed",
				Message:    "Failed to get AWS SSO configuration",
				Details:    err.Error(),
				Suggestion: "Ensure you provide valid AWS SSO details",
			}
		}
	} else {
		GroveLogger.Infof("Found AWS SSO configuration: %s", ssoConfig.ProfileName)
	}

	// Always perform authentication (no existing auth checks)
	GroveLogger.Infof("Initiating AWS SSO login for this session...")
	spinner.Stop() // Stop spinner for interactive login

	loginErr := performIntegratedAwsSsoLogin(ssoConfig)
	if loginErr != nil {
		return &AuthenticationError{
			Type:       "sso_login_failed",
			Message:    "AWS SSO login failed",
			Details:    loginErr.Error(),
			Suggestion: "Check your AWS SSO configuration and try again",
		}
	}

	GroveLogger.Infof("AWS SSO authentication successful for this session")

	// Set AWS environment variables for the shell session
	err := setAWSEnvironmentVariablesForSession(ssoConfig)
	if err != nil {
		return &AuthenticationError{
			Type:    "env_setup_failed",
			Message: "Failed to set AWS environment variables",
			Details: err.Error(),
		}
	}

	return nil
}

// AuthenticationError represents a structured authentication error.
type AuthenticationError struct {
	Type       string
	Message    string
	Details    string
	Suggestion string
}

func (e *AuthenticationError) Error() string {
	return e.Message
}

// formatAuthenticationError creates a user-friendly error message for authentication failures.
func formatAuthenticationError(err error) string {
	authErr, ok := err.(*AuthenticationError)
	if !ok {
		// Fallback for unexpected errors
		return color.RedString("✗") + " Authentication failed: " + err.Error()
	}

	var message strings.Builder

	switch authErr.Type {
	case "sso_config_missing":
		message.WriteString(color.RedString("✗") + " AWS SSO configuration not found\n")
		if authErr.Details != "" {
			message.WriteString(color.CyanString("→") + " " + authErr.Details + "\n")
		}
		message.WriteString(color.CyanString("→") + " " + color.YellowString(authErr.Suggestion) + "\n")
		message.WriteString(color.CyanString("→") + " Then run: " + color.YellowString("kanuka grove enter --auth"))

	case "sso_config_input_failed":
		message.WriteString(color.RedString("✗") + " Failed to get AWS SSO configuration\n")
		if authErr.Details != "" {
			message.WriteString(color.CyanString("→") + " " + authErr.Details + "\n")
		}
		message.WriteString(color.CyanString("→") + " " + color.YellowString(authErr.Suggestion) + "\n")
		message.WriteString(color.CyanString("→") + " Try running: " + color.YellowString("kanuka grove enter --auth") + " again")

	case "sso_login_failed":
		message.WriteString(color.RedString("✗") + " AWS SSO login failed\n")
		if authErr.Details != "" {
			message.WriteString(color.CyanString("→") + " " + authErr.Details + "\n")
		}
		message.WriteString(color.CyanString("→") + " " + color.YellowString(authErr.Suggestion) + "\n")
		message.WriteString(color.CyanString("→") + " Or run: " + color.YellowString("kanuka grove enter --auth") + " again")

	case "sso_verification_failed":
		message.WriteString(color.RedString("✗") + " AWS SSO authentication verification failed\n")
		message.WriteString(color.CyanString("→") + " " + authErr.Details + "\n")
		message.WriteString(color.CyanString("→") + " " + color.YellowString(authErr.Suggestion) + "\n")
		message.WriteString(color.CyanString("→") + " Or check your AWS SSO configuration")

	case "env_setup_failed":
		message.WriteString(color.RedString("✗") + " Failed to set up AWS environment\n")
		if authErr.Details != "" {
			message.WriteString(color.CyanString("→") + " " + authErr.Details + "\n")
		}
		message.WriteString(color.CyanString("→") + " Try running: " + color.YellowString("kanuka grove enter --auth") + " again")

	default:
		message.WriteString(color.RedString("✗") + " Authentication failed: " + authErr.Message)
		if authErr.Suggestion != "" {
			message.WriteString("\n" + color.CyanString("→") + " Try: " + color.YellowString(authErr.Suggestion))
		}
	}

	return message.String()
}

// AWSSSoConfig holds AWS SSO configuration details.
type AWSSSoConfig struct {
	ProfileName string
	SSOStartURL string
	SSORegion   string
	Region      string
}

// findAWSSSoConfig looks for AWS SSO configuration in ~/.aws/config.
func findAWSSSoConfig() (*AWSSSoConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".aws", "config")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read AWS config file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var currentProfile string
	var ssoConfig *AWSSSoConfig

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for profile section
		if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
			currentProfile = strings.TrimSuffix(strings.TrimPrefix(line, "[profile "), "]")
			continue
		}

		// Check for default profile
		if line == "[default]" {
			currentProfile = "default"
			continue
		}

		// Look for SSO configuration in current profile
		if currentProfile != "" && strings.Contains(line, "sso_start_url") {
			if ssoConfig == nil {
				ssoConfig = &AWSSSoConfig{ProfileName: currentProfile}
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
		return nil, fmt.Errorf("no AWS SSO configuration found in ~/.aws/config")
	}

	return ssoConfig, nil
}

// promptForSSOConfig interactively prompts the user for AWS SSO configuration.
func promptForSSOConfig() (*AWSSSoConfig, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s AWS SSO Configuration Required\n", color.YellowString("→"))
	fmt.Printf("%s Please provide your AWS SSO details:\n\n", color.CyanString("→"))

	// Prompt for SSO Start URL
	fmt.Printf("%s SSO Start URL: ", color.CyanString("→"))
	startURL, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read SSO start URL: %w", err)
	}
	startURL = strings.TrimSpace(startURL)
	if startURL == "" {
		return nil, fmt.Errorf("SSO start URL cannot be empty")
	}

	// Prompt for SSO Region
	fmt.Printf("%s SSO Region (e.g., us-east-1): ", color.CyanString("→"))
	ssoRegion, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read SSO region: %w", err)
	}
	ssoRegion = strings.TrimSpace(ssoRegion)
	if ssoRegion == "" {
		ssoRegion = "us-east-1" // Default
	}

	// Prompt for Default Region
	fmt.Printf("%s Default AWS Region (e.g., us-east-1): ", color.CyanString("→"))
	region, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read default region: %w", err)
	}
	region = strings.TrimSpace(region)
	if region == "" {
		region = ssoRegion // Use SSO region as default
	}

	// Prompt for Profile Name (optional)
	fmt.Printf("%s Profile Name (default: 'session'): ", color.CyanString("→"))
	profileName, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read profile name: %w", err)
	}
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		profileName = "session"
	}

	fmt.Printf("\n%s Configuration saved for this session\n", color.GreenString("✓"))

	return &AWSSSoConfig{
		ProfileName: profileName,
		SSOStartURL: startURL,
		SSORegion:   ssoRegion,
		Region:      region,
	}, nil
}

// isAWSSSoAuthenticated checks if the user is currently authenticated with AWS SSO.
func isAWSSSoAuthenticated(config *AWSSSoConfig) bool {
	// Create SSO config for synfinatic/aws-sso-cli
	ssoConfig := &sso.SSOConfig{
		SSORegion:     config.SSORegion,
		StartUrl:      config.SSOStartURL,
		DefaultRegion: config.Region,
		MaxBackoff:    30,
		MaxRetry:      3,
	}

	// Create AWSSSO instance with nil storage (we don't need persistent storage for auth check)
	awsSSO := sso.NewAWSSSO(ssoConfig, nil)

	// Try to check if we're already authenticated (non-interactive)
	// This will return nil if already authenticated, error if not
	err := awsSSO.Authenticate("", "")
	return err == nil
}

// performIntegratedAwsSsoLogin uses the synfinatic/aws-sso-cli library directly for authentication.
func performIntegratedAwsSsoLogin(config *AWSSSoConfig) error {
	GroveLogger.Infof("Starting AWS SSO authentication using synfinatic/aws-sso-cli library...")

	// Create SSO config with actual values from AWS config
	ssoConfig := &sso.SSOConfig{
		SSORegion:     config.SSORegion,
		StartUrl:      config.SSOStartURL,
		DefaultRegion: config.Region,
		MaxBackoff:    30,
		MaxRetry:      3,
	}

	// Create AWSSSO instance with nil storage (we don't need persistent storage)
	awsSSO := sso.NewAWSSSO(ssoConfig, nil)

	// Attempt interactive authentication
	// The empty strings are for browser and browser-exec-path parameters
	// The library will handle opening the browser and the authentication flow
	err := awsSSO.Authenticate("", "")
	if err != nil {
		return fmt.Errorf("AWS SSO authentication failed using synfinatic/aws-sso-cli library: %w", err)
	}

	GroveLogger.Infof("AWS SSO authentication successful using synfinatic/aws-sso-cli library")
	return nil
}

// setAWSEnvironmentVariablesForSession sets AWS environment variables for the shell session.
func setAWSEnvironmentVariablesForSession(config *AWSSSoConfig) error {
	// Set AWS_PROFILE environment variable
	err := os.Setenv("AWS_PROFILE", config.ProfileName)
	if err != nil {
		return fmt.Errorf("failed to set AWS_PROFILE: %w", err)
	}

	// Set region environment variables
	if config.Region != "" {
		os.Setenv("AWS_DEFAULT_REGION", config.Region)
		os.Setenv("AWS_REGION", config.Region)
		GroveLogger.Infof("Set AWS_PROFILE=%s, AWS_REGION=%s for this session", config.ProfileName, config.Region)
	} else {
		GroveLogger.Infof("Set AWS_PROFILE=%s for this session", config.ProfileName)
	}

	return nil
}

// handleNamedEnvironment loads a named environment configuration.
func handleNamedEnvironment(envName string) error {
	GroveLogger.Debugf("Loading named environment: %s", envName)

	// TODO: Implement named environment loading
	// For now, this is a placeholder that will be implemented in a future iteration
	GroveLogger.Infof("Named environment '%s' requested (not yet implemented)", envName)

	return nil
}

func init() {
	groveEnterCmd.Flags().BoolVar(&enterAuth, "auth", false, "prompt for AWS SSO authentication (session-only)")
	groveEnterCmd.Flags().StringVar(&enterEnv, "env", "", "use named environment configuration")
}
