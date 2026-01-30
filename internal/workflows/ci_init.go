package workflows

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
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
