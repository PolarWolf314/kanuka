package cmd

import (
	"bufio"
	"context"
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

	// Get current environment and ensure AWS variables are preserved
	env := os.Environ()
	
	// Extract AWS environment variables to ensure they're passed
	awsVars := make(map[string]string)
	for _, envVar := range env {
		if strings.HasPrefix(envVar, "AWS_") {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				awsVars[parts[0]] = parts[1]
			}
		}
	}
	
	// Create a new environment slice with AWS variables explicitly added
	newEnv := make([]string, 0, len(env)+len(awsVars))
	
	// Add all non-AWS environment variables first
	for _, envVar := range env {
		if !strings.HasPrefix(envVar, "AWS_") {
			newEnv = append(newEnv, envVar)
		}
	}
	
	// Explicitly add AWS variables at the end (to override any conflicts)
	for key, value := range awsVars {
		newEnv = append(newEnv, fmt.Sprintf("%s=%s", key, value))
	}
	
	env = newEnv

	// Execute devenv shell --clean, replacing the current process
	// This ensures all environment variables (including AWS credentials) are passed to the shell
	GroveLogger.Debugf("Executing: %s %v", devenvPath, args[1:])
	GroveLogger.Debugf("Environment variables count: %d", len(env))
	
	// Log AWS-related environment variables for debugging (without exposing secrets)
	awsEnvCount := 0
	for _, envVar := range env {
		if strings.HasPrefix(envVar, "AWS_") {
			awsEnvCount++
			if strings.HasPrefix(envVar, "AWS_ACCESS_KEY_ID=") || 
			   strings.HasPrefix(envVar, "AWS_SECRET_ACCESS_KEY=") || 
			   strings.HasPrefix(envVar, "AWS_SESSION_TOKEN=") {
				GroveLogger.Debugf("AWS credential env var set: %s=***", strings.Split(envVar, "=")[0])
			} else {
				GroveLogger.Debugf("AWS env var: %s", envVar)
			}
		}
	}
	GroveLogger.Debugf("Total AWS environment variables: %d (no AWS_PROFILE set - using direct credentials)", awsEnvCount)
	
	// Also verify the specific variables we need are set
	if accessKey := os.Getenv("AWS_ACCESS_KEY_ID"); accessKey != "" {
		GroveLogger.Debugf("AWS_ACCESS_KEY_ID is set (length: %d)", len(accessKey))
	} else {
		GroveLogger.Debugf("WARNING: AWS_ACCESS_KEY_ID is not set!")
	}
	
	if secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY"); secretKey != "" {
		GroveLogger.Debugf("AWS_SECRET_ACCESS_KEY is set (length: %d)", len(secretKey))
	} else {
		GroveLogger.Debugf("WARNING: AWS_SECRET_ACCESS_KEY is not set!")
	}
	
	if sessionToken := os.Getenv("AWS_SESSION_TOKEN"); sessionToken != "" {
		GroveLogger.Debugf("AWS_SESSION_TOKEN is set (length: %d)", len(sessionToken))
	} else {
		GroveLogger.Debugf("WARNING: AWS_SESSION_TOKEN is not set!")
	}
	
	err = syscall.Exec(devenvPath, args, env)
	if err != nil {
		return fmt.Errorf("failed to execute devenv shell --clean: %w", err)
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

	// Set AWS environment variables for the shell session
	err := setAWSEnvironmentVariablesForSession(ssoConfig, credentials)
	if err != nil {
		return &AuthenticationError{
			Type:    "env_setup_failed",
			Message: "Failed to set AWS environment variables",
			Details: err.Error(),
		}
	}

	// Verify environment variables are set and test credentials
	GroveLogger.Infof("AWS credentials configured for shell session")
	GroveLogger.Debugf("AWS_ACCESS_KEY_ID: %s", os.Getenv("AWS_ACCESS_KEY_ID")[:10]+"...")
	GroveLogger.Debugf("AWS_REGION: %s", os.Getenv("AWS_REGION"))

	// Test the credentials by calling AWS STS GetCallerIdentity
	err = testAWSCredentials()
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

// setAWSEnvironmentVariablesForSession sets AWS environment variables for the shell session.
func setAWSEnvironmentVariablesForSession(config *AWSSSoConfig, credentials *AWSCredentials) error {
	// Set AWS credentials environment variables
	err := os.Setenv("AWS_ACCESS_KEY_ID", credentials.AccessKeyID)
	if err != nil {
		return fmt.Errorf("failed to set AWS_ACCESS_KEY_ID: %w", err)
	}

	err = os.Setenv("AWS_SECRET_ACCESS_KEY", credentials.SecretAccessKey)
	if err != nil {
		return fmt.Errorf("failed to set AWS_SECRET_ACCESS_KEY: %w", err)
	}

	err = os.Setenv("AWS_SESSION_TOKEN", credentials.SessionToken)
	if err != nil {
		return fmt.Errorf("failed to set AWS_SESSION_TOKEN: %w", err)
	}

	// Set region environment variables
	if config.Region != "" {
		os.Setenv("AWS_DEFAULT_REGION", config.Region)
		os.Setenv("AWS_REGION", config.Region)
		GroveLogger.Infof("Set AWS credentials and region=%s for this session", config.Region)
	} else {
		GroveLogger.Infof("Set AWS credentials for this session")
	}

	// Don't set AWS_PROFILE - let AWS CLI use environment variables directly
	// This avoids issues with missing profile files in clean environments

	return nil
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
