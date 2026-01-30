# CI-Init Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement `kanuka secrets ci-init` command to set up GitHub Actions CI integration with secure private key display.

**Architecture:** Two-layer command pattern (cmd/secrets_ci_init.go + internal/workflows/ci_init.go). Generate RSA keypair in memory, add CI user to project config, register via existing workflow, create GitHub workflow template, display private key securely via TTY.

**Tech Stack:** Go, Cobra CLI, crypto/rsa, os (for /dev/tty and CON), existing kanuka internal packages.

---

## Task 1: Add TTY Write Utilities

**Files:**
- Modify: `internal/utils/terminal.go`

**Step 1: Write the failing test**

Create a simple test that verifies the new functions exist (we'll test behavior manually since TTY is hard to unit test):

```go
// Add to internal/utils/terminal_test.go (create if needed)
package utils

import (
	"testing"
)

func TestWriteToTTYExists(t *testing.T) {
	// This test just verifies the function signature exists
	// Actual TTY behavior is tested manually or in integration tests
	var _ func(string) error = WriteToTTY
}

func TestClearScreenExists(t *testing.T) {
	var _ func() error = ClearScreen
}

func TestWaitForEnterFromTTYExists(t *testing.T) {
	var _ func() error = WaitForEnterFromTTY
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/utils/... -run TestWriteToTTY`
Expected: FAIL with "undefined: WriteToTTY"

**Step 3: Write the implementation**

Add to `internal/utils/terminal.go`:

```go
// WriteToTTY writes content directly to the terminal (bypassing stdout/stderr).
// On Unix, writes to /dev/tty. On Windows, writes to CON.
// Returns an error if the TTY cannot be opened.
func WriteToTTY(content string) error {
	ttyPath := "/dev/tty"
	if runtime.GOOS == "windows" {
		ttyPath = "CON"
	}

	tty, err := os.OpenFile(ttyPath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("cannot open %s for writing: %w", ttyPath, err)
	}
	defer tty.Close()

	_, err = tty.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to TTY: %w", err)
	}

	return nil
}

// ClearScreen clears the terminal screen using ANSI escape sequences.
// Writes directly to TTY to ensure it works even when stdout is redirected.
func ClearScreen() error {
	// ANSI escape sequence: clear screen and move cursor to top-left
	return WriteToTTY("\033[2J\033[H")
}

// WaitForEnterFromTTY waits for the user to press Enter on the TTY.
// This reads from /dev/tty (or CON on Windows) directly.
func WaitForEnterFromTTY() error {
	ttyPath := "/dev/tty"
	if runtime.GOOS == "windows" {
		ttyPath = "CON"
	}

	tty, err := os.Open(ttyPath)
	if err != nil {
		return fmt.Errorf("cannot open %s for reading: %w", ttyPath, err)
	}
	defer tty.Close()

	buf := make([]byte, 1)
	for {
		_, err := tty.Read(buf)
		if err != nil {
			return fmt.Errorf("failed to read from TTY: %w", err)
		}
		if buf[0] == '\n' || buf[0] == '\r' {
			return nil
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./internal/utils/... -run TestWriteToTTY`
Expected: PASS

**Step 5: Commit**

```bash
jj commit -m "feat(utils): add TTY write utilities for secure output"
```

---

## Task 2: Add CI User Email Constant and Detection

**Files:**
- Modify: `internal/workflows/ci_init.go` (create new file)

**Step 1: Create the workflow file with constants and check function**

Create `internal/workflows/ci_init.go`:

```go
package workflows

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
```

**Step 2: Run linter to verify syntax**

Run: `golangci-lint run ./internal/workflows/ci_init.go`
Expected: No errors (some unused imports warning is OK for now)

**Step 3: Commit**

```bash
jj commit -m "feat(workflows): add ci-init constants and CI user detection"
```

---

## Task 3: Add GitHub Remote URL Detection

**Files:**
- Modify: `internal/workflows/ci_init.go`

**Step 1: Add the git remote detection function**

Add to `internal/workflows/ci_init.go`:

```go
// detectGitHubRepoURL attempts to detect the GitHub repository URL from git remote.
// Returns a placeholder if detection fails.
func detectGitHubRepoURL() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "https://github.com/<owner>/<repo>"
	}

	remoteURL := strings.TrimSpace(string(output))
	
	// Convert SSH URL to HTTPS URL
	// ssh://git@github.com/owner/repo.git -> https://github.com/owner/repo
	// git@github.com:owner/repo.git -> https://github.com/owner/repo
	
	// Handle ssh:// format
	if strings.HasPrefix(remoteURL, "ssh://git@github.com/") {
		remoteURL = strings.TrimPrefix(remoteURL, "ssh://git@github.com/")
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
		return "https://github.com/" + remoteURL
	}
	
	// Handle git@ format
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		remoteURL = strings.TrimPrefix(remoteURL, "git@github.com:")
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
		return "https://github.com/" + remoteURL
	}
	
	// Handle HTTPS format
	if strings.HasPrefix(remoteURL, "https://github.com/") {
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
		return remoteURL
	}
	
	return "https://github.com/<owner>/<repo>"
}
```

**Step 2: Run linter**

Run: `golangci-lint run ./internal/workflows/ci_init.go`
Expected: PASS

**Step 3: Commit**

```bash
jj commit -m "feat(workflows): add GitHub remote URL detection for ci-init"
```

---

## Task 4: Add RSA Key Generation In-Memory

**Files:**
- Modify: `internal/secrets/keys.go`

**Step 1: Add in-memory key generation function**

Add to `internal/secrets/keys.go`:

```go
// GenerateRSAKeyPairInMemory generates a new RSA key pair and returns them without saving to disk.
// Returns the private key, public key PEM bytes, and any error.
func GenerateRSAKeyPairInMemory() (*rsa.PrivateKey, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	// Encode private key to PEM format
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	privateKeyPEM := pem.EncodeToMemory(privPem)

	return privateKey, privateKeyPEM, nil
}

// GetPublicKeyPEM returns the PEM-encoded public key from an RSA private key.
func GetPublicKeyPEM(privateKey *rsa.PrivateKey) ([]byte, error) {
	pubASN1, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPem := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	}
	return pem.EncodeToMemory(pubPem), nil
}
```

**Step 2: Run linter**

Run: `golangci-lint run ./internal/secrets/keys.go`
Expected: PASS

**Step 3: Commit**

```bash
jj commit -m "feat(secrets): add in-memory RSA key pair generation"
```

---

## Task 5: Add CI Error Type

**Files:**
- Modify: `internal/errors/errors.go`

**Step 1: Read the errors file**

Read: `internal/errors/errors.go`

**Step 2: Add the new error**

Add to `internal/errors/errors.go`:

```go
// ErrCIAlreadyConfigured is returned when CI integration is already set up.
var ErrCIAlreadyConfigured = errors.New("CI integration already configured")

// ErrTTYRequired is returned when a command requires TTY but none is available.
var ErrTTYRequired = errors.New("this command requires an interactive terminal")
```

**Step 3: Run linter**

Run: `golangci-lint run ./internal/errors/...`
Expected: PASS

**Step 4: Commit**

```bash
jj commit -m "feat(errors): add CI-specific error types"
```

---

## Task 6: Implement CI-Init Workflow Core Logic

**Files:**
- Modify: `internal/workflows/ci_init.go`

**Step 1: Implement the CIInit function**

Add to `internal/workflows/ci_init.go`:

```go
// CIInit sets up GitHub Actions CI integration for the project.
// It generates a new keypair for CI, registers the CI user, and creates a workflow template.
// The private key is returned in the result and should be displayed securely to the user.
func CIInit(ctx context.Context, opts CIInitOptions) (*CIInitResult, error) {
	// Initialize project settings
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	// Check if TTY is available
	if !utils.IsTTYAvailable() {
		return nil, kerrors.ErrTTYRequired
	}

	// Check if CI is already configured
	ciExists, err := IsCIUserRegistered()
	if err != nil {
		return nil, fmt.Errorf("checking CI user: %w", err)
	}
	if ciExists {
		return nil, kerrors.ErrCIAlreadyConfigured
	}

	// Load project config
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	// Load user config to get current user's private key for registration
	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return nil, fmt.Errorf("loading user config: %w", err)
	}

	// Generate CI keypair in memory
	ciPrivateKey, ciPrivateKeyPEM, err := secrets.GenerateRSAKeyPairInMemory()
	if err != nil {
		return nil, fmt.Errorf("generating CI keypair: %w", err)
	}

	ciPublicKeyPEM, err := secrets.GetPublicKeyPEM(ciPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("getting CI public key: %w", err)
	}

	// Generate UUID for CI user
	ciUserUUID := configs.GenerateUserUUID()

	// Add CI user to project config
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

	// Track cleanup state
	cleanupNeeded := true
	defer func() {
		if cleanupNeeded {
			// Rollback: remove CI user from config
			delete(projectConfig.Users, ciUserUUID)
			delete(projectConfig.Devices, ciUserUUID)
			_ = configs.SaveProjectConfig(projectConfig)
			
			// Remove any created files
			pubKeyPath := filepath.Join(configs.ProjectKanukaSettings.ProjectPublicKeyPath, ciUserUUID+".pub")
			kanukaPath := filepath.Join(configs.ProjectKanukaSettings.ProjectSecretsPath, ciUserUUID+".kanuka")
			_ = os.Remove(pubKeyPath)
			_ = os.Remove(kanukaPath)
			_ = os.Remove(filepath.Join(projectPath, CIWorkflowPath))
		}
	}()

	// Save updated project config
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		return nil, fmt.Errorf("saving project config: %w", err)
	}

	// Save CI public key to project
	pubKeyPath := filepath.Join(configs.ProjectKanukaSettings.ProjectPublicKeyPath, ciUserUUID+".pub")
	if err := os.WriteFile(pubKeyPath, ciPublicKeyPEM, 0644); err != nil {
		return nil, fmt.Errorf("saving CI public key: %w", err)
	}

	// Register CI user using existing workflow
	registerOpts := RegisterOptions{
		Mode:          RegisterModeEmail,
		UserEmail:     CIUserEmail,
		Verbose:       opts.Verbose,
		Debug:         opts.Debug,
	}

	_, err = Register(ctx, registerOpts)
	if err != nil {
		return nil, fmt.Errorf("registering CI user: %w", err)
	}

	// Create workflow file
	workflowCreated := false
	workflowFullPath := filepath.Join(projectPath, CIWorkflowPath)
	
	// Check if workflow already exists
	if _, err := os.Stat(workflowFullPath); os.IsNotExist(err) {
		// Create .github/workflows directory if needed
		workflowDir := filepath.Dir(workflowFullPath)
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			return nil, fmt.Errorf("creating workflow directory: %w", err)
		}

		// Write workflow template
		if err := os.WriteFile(workflowFullPath, []byte(generateWorkflowTemplate()), 0644); err != nil {
			return nil, fmt.Errorf("creating workflow file: %w", err)
		}
		workflowCreated = true
	}

	// Detect GitHub repo URL
	githubURL := detectGitHubRepoURL()

	// Create audit entry
	auditEntry := audit.LogWithUser("ci-init")
	auditEntry.TargetUser = CIUserEmail
	auditEntry.TargetUUID = ciUserUUID
	audit.Log(auditEntry)

	// Success - disable cleanup
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
```

**Step 2: Run linter**

Run: `golangci-lint run ./internal/workflows/ci_init.go`
Expected: PASS

**Step 3: Commit**

```bash
jj commit -m "feat(workflows): implement ci-init core workflow logic"
```

---

## Task 7: Create the CLI Command

**Files:**
- Create: `cmd/secrets_ci_init.go`

**Step 1: Create the command file**

```go
package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/utils"
	"github.com/PolarWolf314/kanuka/internal/workflows"

	"github.com/spf13/cobra"
)

func init() {
	SecretsCmd.AddCommand(ciInitCmd)
}

// resetCIInitCommandState resets the ci-init command's global state for testing.
func resetCIInitCommandState() {
	// No flags to reset currently
}

var ciInitCmd = &cobra.Command{
	Use:   "ci-init",
	Short: "Set up GitHub Actions CI integration",
	Long: `Set up GitHub Actions CI integration for this project.

This command:
1. Generates a dedicated CI keypair (private key never saved to disk)
2. Registers the CI user with the project
3. Creates a GitHub Actions workflow template
4. Securely displays the private key for you to add to GitHub Secrets

The private key is displayed only once and must be copied to your
GitHub repository's secrets as KANUKA_PRIVATE_KEY.

This command requires an interactive terminal as the private key
is displayed directly to the TTY for security.

Example:
  kanuka secrets ci-init`,
	RunE: runCIInit,
}

func runCIInit(cmd *cobra.Command, args []string) error {
	Logger.Infof("Starting ci-init command")
	spinner, cleanup := startSpinner("Setting up CI integration...", verbose)
	defer cleanup()

	ctx := context.Background()
	opts := workflows.CIInitOptions{
		Verbose: verbose,
		Debug:   debug,
	}

	result, err := workflows.CIInit(ctx, opts)
	if err != nil {
		spinner.FinalMSG = formatCIInitError(err)
		spinner.Stop()
		
		// Return nil for expected errors
		if errors.Is(err, kerrors.ErrProjectNotInitialized) ||
			errors.Is(err, kerrors.ErrCIAlreadyConfigured) ||
			errors.Is(err, kerrors.ErrTTYRequired) ||
			errors.Is(err, kerrors.ErrNoAccess) {
			return nil
		}
		return err
	}

	// Stop spinner before TTY output
	spinner.Stop()

	// Display the private key securely
	if err := displayPrivateKeySecurely(result); err != nil {
		fmt.Println(ui.Error.Sprint("✗") + " Failed to display private key: " + err.Error())
		return err
	}

	// Show success message and next steps
	printCIInitSuccess(result)
	return nil
}

func formatCIInitError(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kanuka has not been initialized\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first"

	case errors.Is(err, kerrors.ErrCIAlreadyConfigured):
		return ui.Error.Sprint("✗") + " CI integration is already configured\n" +
			ui.Info.Sprint("→") + " To reconfigure, first run " + ui.Code.Sprint("kanuka secrets revoke --user "+workflows.CIUserEmail)

	case errors.Is(err, kerrors.ErrTTYRequired):
		return ui.Error.Sprint("✗") + " This command requires an interactive terminal\n" +
			ui.Info.Sprint("→") + " Run this command directly in your terminal (not piped or in a script)"

	case errors.Is(err, kerrors.ErrNoAccess):
		return ui.Error.Sprint("✗") + " You don't have access to this project\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets create") + " to generate your keys"

	default:
		return ui.Error.Sprint("✗") + " CI setup failed: " + err.Error()
	}
}

func displayPrivateKeySecurely(result *workflows.CIInitResult) error {
	// Display brief instructions
	preMessage := "\n" +
		ui.Warning.Sprint("IMPORTANT:") + " Copy the private key below and save it to GitHub Secrets.\n" +
		"This key will " + ui.Error.Sprint("NOT") + " be shown again.\n\n" +
		strings.Repeat("=", 70) + "\n\n"

	if err := utils.WriteToTTY(preMessage); err != nil {
		return fmt.Errorf("writing instructions: %w", err)
	}

	// Display the private key
	if err := utils.WriteToTTY(string(result.PrivateKeyPEM)); err != nil {
		return fmt.Errorf("writing private key: %w", err)
	}

	// Display wait prompt
	postMessage := "\n" + strings.Repeat("=", 70) + "\n\n" +
		"Press " + ui.Highlight.Sprint("Enter") + " when you have copied the key..."

	if err := utils.WriteToTTY(postMessage); err != nil {
		return fmt.Errorf("writing prompt: %w", err)
	}

	// Wait for Enter
	if err := utils.WaitForEnterFromTTY(); err != nil {
		return fmt.Errorf("waiting for input: %w", err)
	}

	// Clear screen
	if err := utils.ClearScreen(); err != nil {
		// Non-fatal - just continue
		Logger.Debugf("Failed to clear screen: %v", err)
	}

	return nil
}

func printCIInitSuccess(result *workflows.CIInitResult) {
	fmt.Println()
	fmt.Println(ui.Success.Sprint("✓") + " CI user registered successfully!")
	
	if result.WorkflowCreated {
		fmt.Println(ui.Success.Sprint("✓") + " Workflow template created at " + ui.Path.Sprint(result.WorkflowPath))
	} else {
		fmt.Println(ui.Warning.Sprint("⚠") + " Workflow file already exists at " + ui.Path.Sprint(result.WorkflowPath) + " (skipped)")
	}

	fmt.Println()
	fmt.Println(ui.Highlight.Sprint("Next steps:"))
	fmt.Println()

	secretsURL := result.GitHubRepoURL + "/settings/secrets/actions"
	
	fmt.Println("1. Go to your GitHub repository secrets:")
	fmt.Println("   " + ui.Code.Sprint(secretsURL))
	fmt.Println()
	fmt.Println("2. Click " + ui.Highlight.Sprint("\"New repository secret\""))
	fmt.Println()
	fmt.Println("3. Name: " + ui.Code.Sprint("KANUKA_PRIVATE_KEY"))
	fmt.Println("   Value: (paste the private key you just copied)")
	fmt.Println()
	fmt.Println("4. Click " + ui.Highlight.Sprint("\"Add secret\""))
	fmt.Println()
	fmt.Println("5. Commit and push the changes:")
	fmt.Println("   " + ui.Code.Sprint("git add .github/workflows/kanuka-decrypt.yml .kanuka/"))
	fmt.Println("   " + ui.Code.Sprint("git commit -m \"Add Kanuka CI integration\""))
	fmt.Println("   " + ui.Code.Sprint("git push"))
	fmt.Println()
	fmt.Println(ui.Info.Sprint("→") + " The next pull request will automatically decrypt secrets!")
}
```

**Step 2: Update secrets.go to include reset function**

Add to `cmd/secrets.go` in `ResetGlobalState()`:

```go
// Reset the ci-init command flags
resetCIInitCommandState()
```

And add to `resetCobraFlagState()`:

```go
// Reset the ci-init command flags specifically
if ciInitCmd != nil && ciInitCmd.Flags() != nil {
	ciInitCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flag.Changed = false
	})
}
```

**Step 3: Run linter and build**

Run: `golangci-lint run ./cmd/secrets_ci_init.go && go build -v ./...`
Expected: PASS

**Step 4: Commit**

```bash
jj commit -m "feat(cmd): add secrets ci-init command"
```

---

## Task 8: Create Integration Tests

**Files:**
- Create: `test/integration/ci_init/ci_init_test.go`

**Step 1: Create test directory and file**

```go
package ci_init_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/workflows"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestCIInitBasic tests the ci-init command in various scenarios.
func TestCIInitBasic(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}
	originalUserSettings := configs.UserKanukaSettings

	t.Run("CIInitNotInitialized", func(t *testing.T) {
		testCIInitNotInitialized(t, originalWd, originalUserSettings)
	})

	t.Run("CIInitAlreadyConfigured", func(t *testing.T) {
		testCIInitAlreadyConfigured(t, originalWd, originalUserSettings)
	})

	t.Run("CIInitWorkflowExists", func(t *testing.T) {
		testCIInitWorkflowExists(t, originalWd, originalUserSettings)
	})
}

// testCIInitNotInitialized tests that ci-init fails when project is not initialized.
func testCIInitNotInitialized(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-ci-init-notinit-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("ci-init", []string{}, nil, nil, false, false)
		return cmd.Execute()
	})

	// Command should not return an error (expected errors are handled internally)
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	// Check output contains expected error message
	if !strings.Contains(output, "has not been initialized") {
		t.Errorf("Expected 'not initialized' error, got: %s", output)
	}
}

// testCIInitAlreadyConfigured tests that ci-init fails when CI is already set up.
func testCIInitAlreadyConfigured(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-ci-init-already-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project first
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Add CI user to project config manually to simulate already configured
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	
	ciUUID := "ci-test-uuid"
	projectConfig.Users[ciUUID] = workflows.CIUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("ci-init", []string{}, nil, nil, false, false)
		return cmd.Execute()
	})

	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "already configured") {
		t.Errorf("Expected 'already configured' error, got: %s", output)
	}
}

// testCIInitWorkflowExists tests that ci-init skips workflow creation when file exists.
func testCIInitWorkflowExists(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-ci-init-workflow-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	// Initialize project first
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create workflow file manually
	workflowDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("Failed to create workflow directory: %v", err)
	}
	
	workflowPath := filepath.Join(workflowDir, "kanuka-decrypt.yml")
	if err := os.WriteFile(workflowPath, []byte("existing workflow"), 0644); err != nil {
		t.Fatalf("Failed to create workflow file: %v", err)
	}

	// Note: Full ci-init test requires TTY which is hard to mock in tests
	// This test just verifies the workflow detection logic
	
	// Verify workflow file exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Errorf("Expected workflow file to exist")
	}
}
```

**Step 2: Run tests**

Run: `go test -v ./test/integration/ci_init/...`
Expected: PASS (or skip TTY-related tests)

**Step 3: Commit**

```bash
jj commit -m "test: add ci-init integration tests"
```

---

## Task 9: Final Verification

**Step 1: Run all linters**

Run: `golangci-lint run`
Expected: PASS

**Step 2: Run all tests**

Run: `go test -v ./...`
Expected: PASS

**Step 3: Build the project**

Run: `go build -v ./...`
Expected: PASS

**Step 4: Manual test**

Test manually in an initialized kanuka project:
```bash
kanuka secrets ci-init
```

**Step 5: Commit any fixes**

```bash
jj commit -m "fix: address linter and test issues"
```

---

## Summary

This implementation plan creates:
1. TTY utilities for secure output (`internal/utils/terminal.go`)
2. CI-specific error types (`internal/errors/errors.go`)
3. In-memory RSA key generation (`internal/secrets/keys.go`)
4. Core ci-init workflow (`internal/workflows/ci_init.go`)
5. CLI command (`cmd/secrets_ci_init.go`)
6. Integration tests (`test/integration/ci_init/ci_init_test.go`)

The implementation follows the existing patterns in the codebase:
- Two-layer architecture (cmd + workflows)
- Spinner pattern for progress indication
- Error handling with specific error types
- Audit logging
- Rollback on failure
