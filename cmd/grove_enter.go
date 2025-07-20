package cmd

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/PolarWolf314/kanuka/internal/grove"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssooidctypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/briandowns/spinner"
	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	enterAuth bool
	enterEnv  string
	
	// Global variables for AWS profile injection
	tempCredentialsFile string // Note: This now stores the profile name, not a file path
	devenvNixBackup     string // Backup file path for devenv.nix restoration
)

var groveEnterCmd = &cobra.Command{
	Use:   "enter",
	Short: "Enter the development shell environment",
	Long: `Enter the development shell environment using devenv with --clean flag.
This starts a clean, isolated shell with all your configured packages and languages available,
ensuring the environment doesn't depend on any system-specific configuration.

Authentication is handled using the official AWS Go SDK - no external dependencies required.
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

// enterDevenvShell executes the devenv shell command, optionally with --clean flag.
func enterDevenvShell() error {
	// Find devenv executable
	devenvPath, err := exec.LookPath("devenv")
	if err != nil {
		return fmt.Errorf("devenv command not found: %w", err)
	}

	// Always use --clean for consistent isolated environment
	args := []string{"devenv", "shell", "--clean"}
	GroveLogger.Debugf("Using devenv shell --clean for isolated environment")

	// Get current environment
	env := os.Environ()

	// If we have AWS credentials to inject, create a wrapper script for cleanup
	if tempCredentialsFile != "" {
		return executeWithAWSCleanupWrapper(devenvPath, args, env)
	}

	// No auth needed, execute directly
	GroveLogger.Debugf("Executing: %s %v", devenvPath, args[1:])
	err = syscall.Exec(devenvPath, args, env)
	if err != nil {
		return fmt.Errorf("failed to execute devenv shell --clean: %w", err)
	}

	// This line should never be reached if syscall.Exec succeeds
	return nil
}

// executeWithAWSCleanupWrapper creates a wrapper script that handles AWS profile cleanup and executes devenv
func executeWithAWSCleanupWrapper(devenvPath string, args []string, env []string) error {
	// Generate random filename for wrapper script
	randomSuffix, err := generateRandomString(8)
	if err != nil {
		return fmt.Errorf("failed to generate random string for wrapper: %w", err)
	}
	
	wrapperScript := filepath.Join(os.TempDir(), fmt.Sprintf("kanuka-wrapper-%s.sh", randomSuffix))
	
	// Create wrapper script content
	wrapperContent := fmt.Sprintf(`#!/bin/bash
# Kanuka Grove wrapper script for AWS profile cleanup
set -e

# Cleanup function
cleanup_aws_profile() {
    echo "🧹 Cleaning up Kanuka AWS session..."
    
    # Remove temporary AWS profile from ~/.aws/credentials
    if [ -f "$HOME/.aws/credentials" ]; then
        # Create a temporary file without the kanuka profile
        grep -v "^\[%s\]" "$HOME/.aws/credentials" | \
        awk '/^\[%s\]/{skip=1; next} /^\[.*\]/{skip=0} !skip' > "$HOME/.aws/credentials.tmp" 2>/dev/null || true
        
        # Replace the original file if temp file was created successfully
        if [ -f "$HOME/.aws/credentials.tmp" ]; then
            mv "$HOME/.aws/credentials.tmp" "$HOME/.aws/credentials"
            echo "✓ Removed temporary AWS profile '%s'"
        fi
    fi
    
    # Restore original devenv.nix if backup exists
    if [ -f "%s" ]; then
        mv "%s" devenv.nix
        echo "✓ Restored devenv.nix"
    fi
    
    # Remove wrapper script itself
    rm -f "%s"
    echo "✓ Cleanup complete"
}

# Set up cleanup trap - this will run when the shell exits
trap cleanup_aws_profile EXIT

# AWS_PROFILE is injected into devenv.nix, so it will be available in the clean environment
echo "✓ Using AWS profile: %s"

# Replace this shell process with devenv (no floating wrapper during session)
exec %s %s
`, 
		tempCredentialsFile, tempCredentialsFile, tempCredentialsFile,
		devenvNixBackup, devenvNixBackup,
		wrapperScript,
		tempCredentialsFile,
		devenvPath, strings.Join(args[1:], " "))

	// Create wrapper script with executable permissions
	err = os.WriteFile(wrapperScript, []byte(wrapperContent), 0755)
	if err != nil {
		return fmt.Errorf("failed to create wrapper script: %w", err)
	}

	GroveLogger.Debugf("Created wrapper script: %s", wrapperScript)
	GroveLogger.Debugf("Executing wrapper script with cleanup trap")

	// Execute the wrapper script, replacing the current process
	wrapperArgs := []string{wrapperScript}
	err = syscall.Exec(wrapperScript, wrapperArgs, env)
	if err != nil {
		// Clean up wrapper script if exec fails
		os.Remove(wrapperScript)
		return fmt.Errorf("failed to execute wrapper script: %w", err)
	}

	// This line should never be reached if syscall.Exec succeeds
	return nil
}

// handleAuthentication sets up AWS SSO authentication using the official AWS Go SDK.
func handleAuthentication(spinner *spinner.Spinner) error {
	GroveLogger.Debugf("Setting up AWS SSO authentication using official AWS Go SDK")

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

	credentials, loginErr := performAwsSsoLogin(ssoConfig)
	if loginErr != nil {
		return &AuthenticationError{
			Type:       "sso_login_failed",
			Message:    "AWS SSO login failed",
			Details:    loginErr.Error(),
			Suggestion: "Check your AWS SSO configuration and try again",
		}
	}

	GroveLogger.Infof("AWS SSO authentication successful for this session")

	// Prepare AWS credentials for secure injection into devenv shell
	err := prepareAWSCredentialsForDevenv(ssoConfig, credentials)
	if err != nil {
		return &AuthenticationError{
			Type:    "env_setup_failed",
			Message: "Failed to prepare AWS credentials for devenv",
			Details: err.Error(),
		}
	}

	// Test the credentials by calling AWS STS GetCallerIdentity
	err = testAWSCredentialsWithConfig(ssoConfig, credentials)
	if err != nil {
		GroveLogger.Debugf("Warning: AWS credentials test failed: %v", err)
		fmt.Printf("%s Warning: AWS credentials may not be working properly\n", color.YellowString("⚠"))
	} else {
		fmt.Printf("%s AWS credentials verified and working\n", color.GreenString("✓"))
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
	ctx := context.Background()
	
	// Try to load AWS config with SSO
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(config.SSORegion),
	)
	if err != nil {
		return false
	}

	// Try to get credentials to test if authenticated
	stsClient := sts.NewFromConfig(cfg)
	_, err = stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	return err == nil
}

// AWSCredentials holds temporary AWS credentials from SSO login
type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

// performAwsSsoLogin uses the official AWS Go SDK for SSO authentication.
func performAwsSsoLogin(config *AWSSSoConfig) (*AWSCredentials, error) {
	GroveLogger.Infof("Starting AWS SSO authentication using official AWS Go SDK...")
	
	// Create a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Create AWS config for the SSO region
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(config.SSORegion),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create SSOOIDC client for device authorization
	ssooidcClient := ssooidc.NewFromConfig(cfg)

	// Register the client
	registerResp, err := ssooidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String("kanuka-grove"),
		ClientType: aws.String("public"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register SSO client: %w", err)
	}

	// Start device authorization
	deviceAuthResp, err := ssooidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     registerResp.ClientId,
		ClientSecret: registerResp.ClientSecret,
		StartUrl:     aws.String(config.SSOStartURL),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start device authorization: %w", err)
	}

	// Poll for token with proper waiting and user feedback
	var accessToken string
	interval := time.Duration(deviceAuthResp.Interval) * time.Second
	expiresAt := time.Now().Add(time.Duration(deviceAuthResp.ExpiresIn) * time.Second)

	// Display instructions to user
	fmt.Printf("\n%s AWS SSO Login Required\n", color.YellowString("→"))
	fmt.Printf("%s Open this URL in your browser: %s\n", color.CyanString("→"), color.BlueString(*deviceAuthResp.VerificationUriComplete))
	fmt.Printf("%s Enter this code when prompted: %s\n", color.CyanString("→"), color.YellowString(*deviceAuthResp.UserCode))
	fmt.Printf("%s Waiting for authentication...\n", color.CyanString("→"))
	fmt.Printf("%s Poll interval: %v seconds\n", color.CyanString("→"), interval.Seconds())
	fmt.Printf("%s Expires at: %v\n\n", color.CyanString("→"), expiresAt.Format("15:04:05"))

	fmt.Printf("%s Polling for authentication completion...\n", color.CyanString("→"))
	
	for time.Now().Before(expiresAt) {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("authentication cancelled or timed out: %w", ctx.Err())
		default:
		}
		
		GroveLogger.Debugf("Attempting to create token...")
		tokenResp, err := ssooidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     registerResp.ClientId,
			ClientSecret: registerResp.ClientSecret,
			DeviceCode:   deviceAuthResp.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})

		if err == nil {
			accessToken = *tokenResp.AccessToken
			fmt.Printf("\r%s Authentication successful!                                        \n", color.GreenString("✓"))
			break
		}

		// Log the actual error for debugging
		GroveLogger.Debugf("Token creation error: %v", err)
		
		// Check for specific AWS SDK error types (proper way to handle AWS errors)
		var authPendingErr *ssooidctypes.AuthorizationPendingException
		var slowDownErr *ssooidctypes.SlowDownException
		var expiredTokenErr *ssooidctypes.ExpiredTokenException
		
		if errors.As(err, &authPendingErr) || errors.As(err, &slowDownErr) {
			GroveLogger.Debugf("Authorization still pending, continuing to poll...")
			fmt.Printf("\r%s Still waiting for authentication... (expires in %v)   ", 
				color.YellowString("⏳"), 
				expiresAt.Sub(time.Now()).Round(time.Second))
			time.Sleep(interval)
			continue
		}
		
		if errors.As(err, &expiredTokenErr) {
			fmt.Printf("%s Device code expired. Please try again.\n", color.RedString("✗"))
			return nil, fmt.Errorf("device code expired - please run the command again")
		}

		// Fallback to string matching for any other pending/slow down errors
		errStr := err.Error()
		if strings.Contains(errStr, "AuthorizationPendingException") || 
		   strings.Contains(errStr, "authorization_pending") || 
		   strings.Contains(errStr, "SlowDownException") ||
		   strings.Contains(errStr, "slow_down") {
			GroveLogger.Debugf("Authorization still pending (string match), continuing to poll...")
			fmt.Printf("\r%s Still waiting for authentication... (expires in %v)   ", 
				color.YellowString("⏳"), 
				expiresAt.Sub(time.Now()).Round(time.Second))
			time.Sleep(interval)
			continue
		}

		// Any other error is a failure - but let's be more specific
		fmt.Printf("%s Authentication error: %v\n", color.RedString("✗"), err)
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	if accessToken == "" {
		return nil, fmt.Errorf("authentication timed out - please try again")
	}

	// Get account info and role credentials
	ssoClient := sso.NewFromConfig(cfg)

	// List accounts
	accountsResp, err := ssoClient.ListAccounts(ctx, &sso.ListAccountsInput{
		AccessToken: aws.String(accessToken),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accountsResp.AccountList) == 0 {
		return nil, fmt.Errorf("no AWS accounts found")
	}

	// Select account (use first if only one, otherwise prompt)
	var account types.AccountInfo
	if len(accountsResp.AccountList) == 1 {
		account = accountsResp.AccountList[0]
		fmt.Printf("%s Using AWS account: %s (%s)\n", 
			color.CyanString("→"), 
			*account.AccountName, 
			*account.AccountId)
	} else {
		// Multiple accounts - prompt user to select
		fmt.Printf("%s Multiple AWS accounts found:\n", color.YellowString("→"))
		for i, acc := range accountsResp.AccountList {
			fmt.Printf("  %d. %s (%s)\n", i+1, *acc.AccountName, *acc.AccountId)
		}
		
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("%s Select account (1-%d): ", color.CyanString("→"), len(accountsResp.AccountList))
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read account selection: %w", err)
		}
		
		var selection int
		if _, err := fmt.Sscanf(strings.TrimSpace(input), "%d", &selection); err != nil || selection < 1 || selection > len(accountsResp.AccountList) {
			return nil, fmt.Errorf("invalid account selection")
		}
		
		account = accountsResp.AccountList[selection-1]
		fmt.Printf("%s Selected account: %s\n", color.GreenString("✓"), *account.AccountName)
	}

	// List account roles
	rolesResp, err := ssoClient.ListAccountRoles(ctx, &sso.ListAccountRolesInput{
		AccessToken: aws.String(accessToken),
		AccountId:   account.AccountId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list account roles: %w", err)
	}

	if len(rolesResp.RoleList) == 0 {
		return nil, fmt.Errorf("no roles found for account")
	}

	// Select role (use first if only one, otherwise prompt)
	var role types.RoleInfo
	if len(rolesResp.RoleList) == 1 {
		role = rolesResp.RoleList[0]
		fmt.Printf("%s Using role: %s\n", 
			color.CyanString("→"), 
			*role.RoleName)
	} else {
		// Multiple roles - prompt user to select
		fmt.Printf("%s Multiple roles found:\n", color.YellowString("→"))
		for i, r := range rolesResp.RoleList {
			fmt.Printf("  %d. %s\n", i+1, *r.RoleName)
		}
		
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("%s Select role (1-%d): ", color.CyanString("→"), len(rolesResp.RoleList))
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read role selection: %w", err)
		}
		
		var selection int
		if _, err := fmt.Sscanf(strings.TrimSpace(input), "%d", &selection); err != nil || selection < 1 || selection > len(rolesResp.RoleList) {
			return nil, fmt.Errorf("invalid role selection")
		}
		
		role = rolesResp.RoleList[selection-1]
		fmt.Printf("%s Selected role: %s\n", color.GreenString("✓"), *role.RoleName)
	}

	// Get role credentials
	credsResp, err := ssoClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
		AccessToken: aws.String(accessToken),
		AccountId:   account.AccountId,
		RoleName:    role.RoleName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get role credentials: %w", err)
	}

	GroveLogger.Infof("AWS SSO authentication successful using official AWS Go SDK")

	return &AWSCredentials{
		AccessKeyID:     *credsResp.RoleCredentials.AccessKeyId,
		SecretAccessKey: *credsResp.RoleCredentials.SecretAccessKey,
		SessionToken:    *credsResp.RoleCredentials.SessionToken,
		Expiration:      time.Unix(credsResp.RoleCredentials.Expiration/1000, 0),
	}, nil
}


// generateRandomString generates a cryptographically secure random string
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// prepareAWSCredentialsForDevenv sets up secure credential injection using ~/.aws/credentials
func prepareAWSCredentialsForDevenv(config *AWSSSoConfig, credentials *AWSCredentials) error {
	GroveLogger.Infof("Preparing AWS credentials using ~/.aws/credentials profile")

	// Generate unique profile name for this session
	randomSuffix, err := generateRandomString(8)
	if err != nil {
		return fmt.Errorf("failed to generate random string: %w", err)
	}
	
	tempCredentialsFile = fmt.Sprintf("kanuka-session-%s", randomSuffix) // Store profile name instead of file path
	
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	awsDir := filepath.Join(homeDir, ".aws")
	credentialsFile := filepath.Join(awsDir, "credentials")
	
	// Ensure ~/.aws directory exists
	if err := os.MkdirAll(awsDir, 0755); err != nil {
		return fmt.Errorf("failed to create ~/.aws directory: %w", err)
	}
	
	// Read existing credentials file if it exists
	var existingContent string
	if content, err := os.ReadFile(credentialsFile); err == nil {
		existingContent = string(content)
	}
	
	// Create temporary profile content
	profileContent := fmt.Sprintf(`
[%s]
aws_access_key_id = %s
aws_secret_access_key = %s
aws_session_token = %s
region = %s
`, tempCredentialsFile, credentials.AccessKeyID, credentials.SecretAccessKey, credentials.SessionToken, config.Region)
	
	// Append to existing credentials file
	newContent := existingContent + profileContent
	
	// Write updated credentials file with restrictive permissions
	if err := os.WriteFile(credentialsFile, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write to ~/.aws/credentials: %w", err)
	}
	
	GroveLogger.Debugf("Added temporary profile '%s' to ~/.aws/credentials", tempCredentialsFile)
	
	// Set AWS_PROFILE environment variable for devenv
	os.Setenv("AWS_PROFILE", tempCredentialsFile)
	GroveLogger.Debugf("Set AWS_PROFILE=%s for devenv session", tempCredentialsFile)

	// Since --clean flag isolates environment variables, we need to inject AWS_PROFILE via devenv.nix
	err = injectAWSProfileIntoDevenv(tempCredentialsFile)
	if err != nil {
		return fmt.Errorf("failed to inject AWS_PROFILE into devenv.nix: %w", err)
	}

	return nil
}

// injectAWSProfileIntoDevenv modifies devenv.nix to set AWS_PROFILE environment variable
func injectAWSProfileIntoDevenv(profileName string) error {
	devenvNixPath := "devenv.nix"
	
	// Check if devenv.nix exists
	if _, err := os.Stat(devenvNixPath); os.IsNotExist(err) {
		return fmt.Errorf("devenv.nix not found")
	}

	// Create backup
	backupSuffix, err := generateRandomString(6)
	if err != nil {
		return fmt.Errorf("failed to generate backup suffix: %w", err)
	}
	devenvNixBackup = fmt.Sprintf("devenv.nix.kanuka-backup-%s", backupSuffix)
	
	// Read original devenv.nix
	originalContent, err := os.ReadFile(devenvNixPath)
	if err != nil {
		return fmt.Errorf("failed to read devenv.nix: %w", err)
	}

	// Create backup
	if err := os.WriteFile(devenvNixBackup, originalContent, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	GroveLogger.Debugf("Created devenv.nix backup: %s", devenvNixBackup)

	// Inject AWS_PROFILE into devenv.nix
	modifiedContent, err := injectAWSProfileSafely(string(originalContent), profileName)
	if err != nil {
		return fmt.Errorf("failed to modify devenv.nix content: %w", err)
	}

	// Write modified devenv.nix
	if err := os.WriteFile(devenvNixPath, []byte(modifiedContent), 0644); err != nil {
		// Restore backup on error
		os.WriteFile(devenvNixPath, originalContent, 0644)
		os.Remove(devenvNixBackup)
		return fmt.Errorf("failed to write modified devenv.nix: %w", err)
	}

	GroveLogger.Debugf("Injected AWS_PROFILE=%s into devenv.nix", profileName)
	return nil
}

// injectAWSProfileSafely safely injects AWS_PROFILE environment variable into devenv.nix
func injectAWSProfileSafely(content, profileName string) (string, error) {
	lines := strings.Split(content, "\n")
	var result []string
	var inEnv bool
	var envIndent string
	profileInjected := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// Check if we're starting an env block
		if strings.Contains(trimmedLine, "env") && strings.Contains(trimmedLine, "=") && strings.Contains(trimmedLine, "{") {
			inEnv = true
			envIndent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			result = append(result, line)
			
			// Inject AWS_PROFILE right after opening the env block
			awsProfileLine := fmt.Sprintf(`%s  AWS_PROFILE = "%s";`, envIndent, profileName)
			result = append(result, awsProfileLine)
			profileInjected = true
			continue
		}
		
		// Check if we're ending an env block
		if inEnv && strings.Contains(trimmedLine, "}") {
			inEnv = false
		}
		
		result = append(result, line)
	}
	
	// If no env block was found, add one before the closing brace
	if !profileInjected {
		// Find the last closing brace and add env block before it
		for i := len(result) - 1; i >= 0; i-- {
			if strings.TrimSpace(result[i]) == "}" {
				envBlock := fmt.Sprintf(`
  # AWS profile injection by kanuka grove
  env = {
    AWS_PROFILE = "%s";
  };`, profileName)
				
				// Insert before the closing brace
				newResult := make([]string, 0, len(result)+1)
				newResult = append(newResult, result[:i]...)
				newResult = append(newResult, envBlock)
				newResult = append(newResult, result[i:]...)
				result = newResult
				break
			}
		}
	}
	
	return strings.Join(result, "\n"), nil
}

// testAWSCredentialsWithConfig tests credentials using the provided config and credentials
func testAWSCredentialsWithConfig(config *AWSSSoConfig, credentials *AWSCredentials) error {
	// Temporarily set environment variables for testing
	oldAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	oldSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	oldSessionToken := os.Getenv("AWS_SESSION_TOKEN")
	oldRegion := os.Getenv("AWS_REGION")

	// Set test credentials
	os.Setenv("AWS_ACCESS_KEY_ID", credentials.AccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", credentials.SecretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", credentials.SessionToken)
	os.Setenv("AWS_REGION", config.Region)

	// Test credentials
	err := testAWSCredentials()

	// Restore original environment
	if oldAccessKey != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", oldAccessKey)
	} else {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
	}
	if oldSecretKey != "" {
		os.Setenv("AWS_SECRET_ACCESS_KEY", oldSecretKey)
	} else {
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}
	if oldSessionToken != "" {
		os.Setenv("AWS_SESSION_TOKEN", oldSessionToken)
	} else {
		os.Unsetenv("AWS_SESSION_TOKEN")
	}
	if oldRegion != "" {
		os.Setenv("AWS_REGION", oldRegion)
	} else {
		os.Unsetenv("AWS_REGION")
	}

	return err
}

// testAWSCredentials tests if the AWS credentials are working by calling GetCallerIdentity
func testAWSCredentials() error {
	ctx := context.Background()
	
	// Load AWS config using environment variables
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create STS client and test credentials
	stsClient := sts.NewFromConfig(cfg)
	resp, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get caller identity: %w", err)
	}

	GroveLogger.Debugf("AWS credentials verified - Account: %s, User: %s", 
		*resp.Account, *resp.Arn)
	
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
