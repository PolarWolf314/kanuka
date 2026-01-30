package workflows

import (
	"fmt"

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
