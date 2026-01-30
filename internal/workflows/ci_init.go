package workflows

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/utils"
)

// CIUserEmail is the email address used for the GitHub Actions CI user.
// This is the official GitHub Actions bot email.
const CIUserEmail = "41898282+github-actions[bot]@users.noreply.github.com"

// CIWorkflowPath is the path where the GitHub Actions workflow will be created.
const CIWorkflowPath = ".github/workflows/kanuka-decrypt.yml"

// CIInitOptions configures the ci-init workflow.
type CIInitOptions struct {
	Verbose bool
	Debug   bool
}

// CIInitResult contains the outcome of a ci-init operation.
type CIInitResult struct {
	CIUserUUID      string
	CIUserEmail     string
	WorkflowCreated bool
	WorkflowPath    string
	PrivateKeyPEM   []byte
	GitHubRepoURL   string
}

// IsCIUserRegistered checks if the CI user is already registered in the project.
func IsCIUserRegistered() (bool, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return false, fmt.Errorf("initializing project settings: %w", err)
	}

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return false, fmt.Errorf("loading project config: %w", err)
	}

	_, found := projectConfig.GetUserUUIDByEmail(CIUserEmail)
	return found, nil
}

// detectGitHubRepoURL attempts to detect the GitHub repository URL from git remote.
// Returns a placeholder if detection fails.
func detectGitHubRepoURL() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "https://github.com/<owner>/<repo>"
	}

	remoteURL := strings.TrimSpace(string(output))

	// Convert SSH URL to HTTPS URL.
	// ssh://git@github.com/owner/repo.git -> https://github.com/owner/repo
	// git@github.com:owner/repo.git -> https://github.com/owner/repo

	// Handle ssh:// format.
	if strings.HasPrefix(remoteURL, "ssh://git@github.com/") {
		remoteURL = strings.TrimPrefix(remoteURL, "ssh://git@github.com/")
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
		return "https://github.com/" + remoteURL
	}

	// Handle git@ format.
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		remoteURL = strings.TrimPrefix(remoteURL, "git@github.com:")
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
		return "https://github.com/" + remoteURL
	}

	// Handle HTTPS format.
	if strings.HasPrefix(remoteURL, "https://github.com/") {
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
		return remoteURL
	}

	return "https://github.com/<owner>/<repo>"
}

// CIInit sets up GitHub Actions CI integration for the project.
// It generates a new keypair for CI, registers the CI user, and creates a workflow template.
// The private key is returned in the result and should be displayed securely to the user.
func CIInit(ctx context.Context, opts CIInitOptions) (*CIInitResult, error) {
	// Initialize project settings.
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	// Check if TTY is available.
	if !utils.IsTTYAvailable() {
		return nil, kerrors.ErrTTYRequired
	}

	// Check if CI is already configured.
	ciExists, err := IsCIUserRegistered()
	if err != nil {
		return nil, fmt.Errorf("checking CI user: %w", err)
	}
	if ciExists {
		return nil, kerrors.ErrCIAlreadyConfigured
	}

	// Load project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	// Ensure current user has access to the project.
	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return nil, fmt.Errorf("loading user config: %w", err)
	}
	currentUserUUID := userConfig.User.UUID

	// Verify current user has access by checking for their kanuka key.
	_, err = secrets.GetProjectKanukaKey(currentUserUUID)
	if err != nil {
		return nil, kerrors.ErrNoAccess
	}

	// Generate CI keypair in memory.
	ciPrivateKey, ciPrivateKeyPEM, err := secrets.GenerateRSAKeyPairInMemory()
	if err != nil {
		return nil, fmt.Errorf("generating CI keypair: %w", err)
	}

	ciPublicKeyPEM, err := secrets.GetPublicKeyPEM(ciPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("getting CI public key: %w", err)
	}

	// Generate UUID for CI user.
	ciUserUUID := configs.GenerateUserUUID()

	// Add CI user to project config.
	if projectConfig.Users == nil {
		projectConfig.Users = make(map[string]string)
	}
	projectConfig.Users[ciUserUUID] = CIUserEmail

	if projectConfig.Devices == nil {
		projectConfig.Devices = make(map[string]configs.DeviceConfig)
	}
	projectConfig.Devices[ciUserUUID] = configs.DeviceConfig{
		Email:     CIUserEmail,
		Name:      "github-actions",
		CreatedAt: time.Now().UTC(),
	}

	// Track cleanup state for rollback on failure.
	var cleanupNeeded = true
	pubKeyPath := filepath.Join(configs.ProjectKanukaSettings.ProjectPublicKeyPath, ciUserUUID+".pub")
	kanukaPath := filepath.Join(configs.ProjectKanukaSettings.ProjectSecretsPath, ciUserUUID+".kanuka")
	workflowFullPath := filepath.Join(projectPath, CIWorkflowPath)

	defer func() {
		if cleanupNeeded {
			// Rollback: remove CI user from config.
			delete(projectConfig.Users, ciUserUUID)
			delete(projectConfig.Devices, ciUserUUID)
			_ = configs.SaveProjectConfig(projectConfig)

			// Remove any created files.
			_ = os.Remove(pubKeyPath)
			_ = os.Remove(kanukaPath)
			_ = os.Remove(workflowFullPath)
		}
	}()

	// Save updated project config.
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		return nil, fmt.Errorf("saving project config: %w", err)
	}

	// Register CI user using existing workflow with public key text.
	registerOpts := RegisterOptions{
		Mode:          RegisterModePubkeyText,
		UserEmail:     CIUserEmail,
		PublicKeyText: string(ciPublicKeyPEM),
		Verbose:       opts.Verbose,
		Debug:         opts.Debug,
	}

	_, err = Register(ctx, registerOpts)
	if err != nil {
		return nil, fmt.Errorf("registering CI user: %w", err)
	}

	// Create workflow file.
	workflowCreated := false

	// Check if workflow already exists.
	if _, err := os.Stat(workflowFullPath); os.IsNotExist(err) {
		// Create .github/workflows directory if needed.
		workflowDir := filepath.Dir(workflowFullPath)
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			return nil, fmt.Errorf("creating workflow directory: %w", err)
		}

		// Write workflow template.
		// #nosec G306 -- Workflow file should be world-readable.
		if err := os.WriteFile(workflowFullPath, []byte(generateWorkflowTemplate()), 0644); err != nil {
			return nil, fmt.Errorf("creating workflow file: %w", err)
		}
		workflowCreated = true
	}

	// Detect GitHub repo URL.
	githubURL := detectGitHubRepoURL()

	// Create audit entry.
	auditEntry := audit.LogWithUser("ci-init")
	auditEntry.TargetUser = CIUserEmail
	auditEntry.TargetUUID = ciUserUUID
	audit.Log(auditEntry)

	// Success - disable cleanup.
	cleanupNeeded = false

	return &CIInitResult{
		CIUserUUID:      ciUserUUID,
		CIUserEmail:     CIUserEmail,
		WorkflowCreated: workflowCreated,
		WorkflowPath:    CIWorkflowPath,
		PrivateKeyPEM:   ciPrivateKeyPEM,
		GitHubRepoURL:   githubURL,
	}, nil
}

// generateWorkflowTemplate returns the GitHub Actions workflow YAML content.
func generateWorkflowTemplate() string {
	return `# Kanuka Secrets - GitHub Actions Integration
# Generated by: kanuka secrets ci-init
#
# This workflow demonstrates how to decrypt Kanuka-managed secrets in CI.
# Modify the trigger and add your build/deploy steps as needed.

name: Kanuka Secrets Example

on: [pull_request]

jobs:
  example:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # Set up Kanuka CLI with the private key from GitHub Secrets
      - name: Setup Kanuka
        uses: PolarWolf314/kanuka-actions@v1
        with:
          private-key: ${{ secrets.KANUKA_PRIVATE_KEY }}

      # Decrypt all .kanuka files to .env files
      - name: Decrypt secrets
        run: cat "$KANUKA_PRIVATE_KEY_PATH" | kanuka secrets decrypt --private-key-stdin

      # Example: source the decrypted secrets and use them
      - name: Use secrets
        run: |
          source .env
          echo "Secrets are now available as environment variables"
          # Add your build/deploy commands here
`
}
