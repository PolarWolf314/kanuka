package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/utils"
)

// CreateOptions configures the create workflow.
type CreateOptions struct {
	// Email is the user's email address for identification.
	Email string

	// DeviceName is a custom device name (auto-generated from hostname if empty).
	DeviceName string

	// Force overwrites existing keys if true.
	Force bool
}

// CreateResult contains the outcome of a create operation.
type CreateResult struct {
	// Email is the validated email address used.
	Email string

	// DeviceName is the device name (provided or auto-generated).
	DeviceName string

	// UserUUID is the user's unique identifier.
	UserUUID string

	// PublicKeyPath is where the public key was saved in the project.
	PublicKeyPath string

	// KanukaKeyDeleted indicates if an existing .kanuka key was removed.
	KanukaKeyDeleted bool

	// DeletedKanukaKeyPath is the path of the deleted key (if any).
	DeletedKanukaKeyPath string
}

// CreatePreCheckResult contains information needed before prompting for email.
type CreatePreCheckResult struct {
	// NeedsEmail indicates whether an email needs to be provided.
	NeedsEmail bool

	// ExistingEmail is the email from the user config (if any).
	ExistingEmail string
}

// CreatePreCheck validates the project state before prompting for user input.
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrInvalidProjectConfig if the project config is malformed.
func CreatePreCheck(ctx context.Context) (*CreatePreCheckResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	projectConfigPath := filepath.Join(projectPath, ".kanuka", "config.toml")
	if _, err := os.Stat(projectConfigPath); os.IsNotExist(err) {
		return nil, kerrors.ErrProjectNotInitialized
	}

	if err := secrets.EnsureUserSettings(); err != nil {
		return nil, fmt.Errorf("ensuring user settings: %w", err)
	}

	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return nil, fmt.Errorf("ensuring user config: %w", err)
	}

	return &CreatePreCheckResult{
		NeedsEmail:    userConfig.User.Email == "",
		ExistingEmail: userConfig.User.Email,
	}, nil
}

// Create creates a new RSA key pair for accessing the project's encrypted secrets.
//
// This command generates a unique cryptographic identity for the user on this device,
// identified by their email address. Each device gets its own key pair.
//
// The workflow:
//  1. Generates an RSA key pair (stored locally in ~/.local/share/kanuka/keys/)
//  2. Copies the public key to the project's .kanuka/public_keys/ directory
//  3. Registers the device in the project configuration
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrInvalidProjectConfig if the project config is malformed.
// Returns ErrInvalidEmail if the email format is invalid.
// Returns ErrDeviceNameTaken if the device name is already in use.
// Returns ErrPublicKeyExists if a public key already exists (unless Force is true).
func Create(ctx context.Context, opts CreateOptions) (*CreateResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	projectConfigPath := filepath.Join(projectPath, ".kanuka", "config.toml")
	if _, err := os.Stat(projectConfigPath); os.IsNotExist(err) {
		return nil, kerrors.ErrProjectNotInitialized
	}

	if err := secrets.EnsureUserSettings(); err != nil {
		return nil, fmt.Errorf("ensuring user settings: %w", err)
	}

	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return nil, fmt.Errorf("ensuring user config: %w", err)
	}
	userUUID := userConfig.User.UUID

	// Determine email.
	userEmail := opts.Email
	if userEmail == "" {
		userEmail = userConfig.User.Email
	}

	// Validate email format.
	if !utils.IsValidEmail(userEmail) {
		return nil, fmt.Errorf("%w: %s", kerrors.ErrInvalidEmail, userEmail)
	}

	// Update user config with email if changed.
	if userConfig.User.Email != userEmail {
		userConfig.User.Email = userEmail
		if err := configs.SaveUserConfig(userConfig); err != nil {
			return nil, fmt.Errorf("saving user config: %w", err)
		}
	}

	// Load project config.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		if strings.Contains(err.Error(), "toml:") {
			return nil, fmt.Errorf("%w: .kanuka/config.toml is not valid TOML", kerrors.ErrInvalidProjectConfig)
		}
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	// Determine device name.
	existingDeviceNames := projectConfig.GetDeviceNamesByEmail(userEmail)
	var deviceName string

	if opts.DeviceName != "" {
		deviceName = utils.SanitizeDeviceName(opts.DeviceName)
		if projectConfig.IsDeviceNameTakenByEmail(userEmail, deviceName) {
			return nil, fmt.Errorf("%w: %s", kerrors.ErrDeviceNameTaken, deviceName)
		}
	} else {
		deviceName, err = utils.GenerateDeviceName(existingDeviceNames)
		if err != nil {
			return nil, fmt.Errorf("generating device name: %w", err)
		}
	}

	// Check for existing public key (unless force is set).
	if !opts.Force {
		projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
		userPublicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
		userPublicKey, _ := secrets.LoadPublicKey(userPublicKeyPath)
		if userPublicKey != nil {
			return nil, kerrors.ErrPublicKeyExists
		}
	}

	// Create and save RSA key pair.
	// The verbose parameter is false since logging is handled at the cmd layer.
	if err := secrets.CreateAndSaveRSAKeyPair(false); err != nil {
		return nil, fmt.Errorf("creating RSA key pair: %w", err)
	}

	// Copy user public key to project.
	destPath, err := secrets.CopyUserPublicKeyToProject()
	if err != nil {
		return nil, fmt.Errorf("copying public key to project: %w", err)
	}

	// Add/update user in project config.
	projectConfig.Users[userUUID] = userEmail
	projectConfig.Devices[userUUID] = configs.DeviceConfig{
		Email:     userEmail,
		Name:      deviceName,
		CreatedAt: time.Now().UTC(),
	}

	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		return nil, fmt.Errorf("saving project config: %w", err)
	}

	// Update user config with project entry.
	if userConfig.Projects == nil {
		userConfig.Projects = make(map[string]configs.UserProjectEntry)
	}
	userConfig.Projects[projectConfig.Project.UUID] = configs.UserProjectEntry{
		DeviceName:  deviceName,
		ProjectName: projectConfig.Project.Name,
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		return nil, fmt.Errorf("updating user config with project: %w", err)
	}

	// Remove existing kanuka key if present.
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	userKanukaKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")
	kanukaKeyDeleted := false

	if err := os.Remove(userKanukaKeyPath); err == nil {
		kanukaKeyDeleted = true
	}

	// Log to audit trail.
	auditEntry := audit.LogWithUser("create")
	auditEntry.DeviceName = deviceName
	audit.Log(auditEntry)

	return &CreateResult{
		Email:                userEmail,
		DeviceName:           deviceName,
		UserUUID:             userUUID,
		PublicKeyPath:        destPath,
		KanukaKeyDeleted:     kanukaKeyDeleted,
		DeletedKanukaKeyPath: userKanukaKeyPath,
	}, nil
}
