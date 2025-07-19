package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	enterAuth bool
	enterEnv  string
)

var groveEnterCmd = &cobra.Command{
	Use:   "enter",
	Short: "Enter the development shell environment",
	Long: `Enter the development shell environment using devenv.
This starts a new shell with all your configured packages and languages available.

Examples:
  kanuka grove enter                   # Enter basic development shell
  kanuka grove enter --auth            # Enter shell with AWS SSO authentication
  kanuka grove enter --env production  # Enter shell with production environment`,
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

		// Run devenv gc to clear cache before entering shell
		spinner.Suffix = " Clearing devenv cache..."
		GroveLogger.Debugf("Running devenv gc to clear cache")
		err = runDevenvGC()
		if err != nil {
			GroveLogger.Warnf("Failed to clear devenv cache: %v", err)
			// Continue anyway - this is not a fatal error
		}

		// Stop spinner before entering shell
		spinner.Stop()

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

// enterDevenvShell executes the devenv shell command
func enterDevenvShell() error {
	// Find devenv executable
	devenvPath, err := exec.LookPath("devenv")
	if err != nil {
		return fmt.Errorf("devenv command not found: %w", err)
	}

	// Prepare the command
	args := []string{"devenv", "shell"}
	
	// Execute devenv shell, replacing the current process
	GroveLogger.Debugf("Executing: %s %v", devenvPath, args[1:])
	err = syscall.Exec(devenvPath, args, os.Environ())
	if err != nil {
		return fmt.Errorf("failed to execute devenv shell: %w", err)
	}

	// This line should never be reached if syscall.Exec succeeds
	return nil
}

// handleAuthentication sets up AWS SSO authentication
func handleAuthentication(spinner *spinner.Spinner) error {
	GroveLogger.Debugf("Setting up AWS SSO authentication")
	
	// Check if AWS CLI is available
	awsPath, err := exec.LookPath("aws")
	if err != nil {
		return &AuthenticationError{
			Type:    "aws_cli_missing",
			Message: "AWS CLI not found",
			Suggestion: "kanuka grove add awscli2",
		}
	}
	
	// Check for AWS SSO configuration
	ssoConfig, err := findAWSSSoConfig()
	if err != nil {
		return &AuthenticationError{
			Type:    "sso_config_missing",
			Message: "AWS SSO configuration not found",
			Details: err.Error(),
			Suggestion: "aws configure sso",
		}
	}
	
	GroveLogger.Infof("Found AWS SSO configuration: %s", ssoConfig.ProfileName)
	
	// Check if already authenticated
	if isAWSSSoAuthenticated(ssoConfig.ProfileName) {
		GroveLogger.Infof("Already authenticated with AWS SSO profile: %s", ssoConfig.ProfileName)
		return nil
	}
	
	// Perform AWS SSO login
	GroveLogger.Infof("Initiating AWS SSO login for profile: %s", ssoConfig.ProfileName)
	spinner.Stop() // Stop spinner for interactive login
	err = performAWSSSoLogin(ssoConfig.ProfileName, awsPath)
	if err != nil {
		return &AuthenticationError{
			Type:    "sso_login_failed",
			Message: "AWS SSO login failed",
			Details: err.Error(),
			Suggestion: "aws sso login --profile " + ssoConfig.ProfileName,
		}
	}
	
	// Verify authentication succeeded
	if !isAWSSSoAuthenticated(ssoConfig.ProfileName) {
		return &AuthenticationError{
			Type:    "sso_verification_failed",
			Message: "AWS SSO authentication verification failed",
			Details: "Login completed but unable to verify credentials",
			Suggestion: "aws sso login --profile " + ssoConfig.ProfileName,
		}
	}
	
	GroveLogger.Infof("AWS SSO authentication successful")
	
	// Set AWS environment variables for the shell
	err = setAWSEnvironmentVariables(ssoConfig.ProfileName)
	if err != nil {
		return &AuthenticationError{
			Type:    "env_setup_failed",
			Message: "Failed to set AWS environment variables",
			Details: err.Error(),
		}
	}
	
	return nil
}

// runDevenvGC runs devenv gc to clear the cache
func runDevenvGC() error {
	// Find devenv executable
	devenvPath, err := exec.LookPath("devenv")
	if err != nil {
		return fmt.Errorf("devenv command not found: %w", err)
	}

	// Execute devenv gc
	cmd := exec.Command(devenvPath, "gc")
	cmd.Env = os.Environ()
	
	// Capture output but don't show it unless there's an error
	output, err := cmd.CombinedOutput()
	if err != nil {
		GroveLogger.Debugf("devenv gc output: %s", string(output))
		return fmt.Errorf("devenv gc failed: %w", err)
	}
	
	GroveLogger.Debugf("devenv gc completed successfully")
	return nil
}

// AuthenticationError represents a structured authentication error
type AuthenticationError struct {
	Type       string
	Message    string
	Details    string
	Suggestion string
}

func (e *AuthenticationError) Error() string {
	return e.Message
}

// formatAuthenticationError creates a user-friendly error message for authentication failures
func formatAuthenticationError(err error) string {
	authErr, ok := err.(*AuthenticationError)
	if !ok {
		// Fallback for unexpected errors
		return color.RedString("✗") + " Authentication failed: " + err.Error()
	}

	var message strings.Builder
	
	switch authErr.Type {
	case "aws_cli_missing":
		message.WriteString(color.RedString("✗") + " AWS CLI not found\n")
		message.WriteString(color.CyanString("→") + " Install AWS CLI: " + color.YellowString(authErr.Suggestion) + "\n")
		message.WriteString(color.CyanString("→") + " Then run: " + color.YellowString("kanuka grove enter --auth"))
		
	case "sso_config_missing":
		message.WriteString(color.RedString("✗") + " AWS SSO configuration not found\n")
		if authErr.Details != "" {
			message.WriteString(color.CyanString("→") + " " + authErr.Details + "\n")
		}
		message.WriteString(color.CyanString("→") + " Configure AWS SSO: " + color.YellowString(authErr.Suggestion) + "\n")
		message.WriteString(color.CyanString("→") + " Then run: " + color.YellowString("kanuka grove enter --auth"))
		
	case "sso_login_failed":
		message.WriteString(color.RedString("✗") + " AWS SSO login failed\n")
		if authErr.Details != "" {
			message.WriteString(color.CyanString("→") + " " + authErr.Details + "\n")
		}
		message.WriteString(color.CyanString("→") + " Try again: " + color.YellowString(authErr.Suggestion) + "\n")
		message.WriteString(color.CyanString("→") + " Or run: " + color.YellowString("kanuka grove enter --auth"))
		
	case "sso_verification_failed":
		message.WriteString(color.RedString("✗") + " AWS SSO authentication verification failed\n")
		message.WriteString(color.CyanString("→") + " " + authErr.Details + "\n")
		message.WriteString(color.CyanString("→") + " Try logging in again: " + color.YellowString(authErr.Suggestion) + "\n")
		message.WriteString(color.CyanString("→") + " Or check your AWS configuration")
		
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

// AWSSSoConfig holds AWS SSO configuration details
type AWSSSoConfig struct {
	ProfileName string
	SSOStartURL string
	SSORegion   string
	Region      string
}

// findAWSSSoConfig looks for AWS SSO configuration in ~/.aws/config
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

// isAWSSSoAuthenticated checks if the user is currently authenticated with AWS SSO
func isAWSSSoAuthenticated(profileName string) bool {
	// Try to get caller identity using the SSO profile
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--profile", profileName)
	cmd.Env = os.Environ()
	
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	// If we get valid JSON output, we're authenticated
	return strings.Contains(string(output), "UserId")
}

// performAWSSSoLogin initiates AWS SSO login
func performAWSSSoLogin(profileName, awsPath string) error {
	GroveLogger.Infof("Starting AWS SSO login process...")
	
	// Execute aws sso login
	cmd := exec.Command(awsPath, "sso", "login", "--profile", profileName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("aws sso login failed: %w", err)
	}
	
	// Give a moment for the login to complete
	time.Sleep(2 * time.Second)
	
	return nil
}

// setAWSEnvironmentVariables sets AWS environment variables for the shell
func setAWSEnvironmentVariables(profileName string) error {
	// Set AWS_PROFILE environment variable
	err := os.Setenv("AWS_PROFILE", profileName)
	if err != nil {
		return fmt.Errorf("failed to set AWS_PROFILE: %w", err)
	}
	
	// Try to get and set additional AWS environment variables
	cmd := exec.Command("aws", "configure", "get", "region", "--profile", profileName)
	if output, err := cmd.Output(); err == nil {
		region := strings.TrimSpace(string(output))
		if region != "" {
			os.Setenv("AWS_DEFAULT_REGION", region)
			os.Setenv("AWS_REGION", region)
		}
	}
	
	GroveLogger.Infof("Set AWS_PROFILE=%s", profileName)
	return nil
}

// handleNamedEnvironment loads a named environment configuration
func handleNamedEnvironment(envName string) error {
	GroveLogger.Debugf("Loading named environment: %s", envName)
	
	// TODO: Implement named environment loading
	// For now, this is a placeholder that will be implemented in a future iteration
	GroveLogger.Infof("Named environment '%s' requested (not yet implemented)", envName)
	
	return nil
}

func init() {
	groveEnterCmd.Flags().BoolVar(&enterAuth, "auth", false, "enable AWS SSO authentication")
	groveEnterCmd.Flags().StringVar(&enterEnv, "env", "", "use named environment configuration")
}