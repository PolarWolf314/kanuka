package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/utils"
)

// InitOptions configures the init workflow.
type InitOptions struct {
	// ProjectName is the name for the project. If empty, uses the directory name.
	ProjectName string

	// Verbose enables verbose logging output.
	Verbose bool
}

// InitResult contains the outcome of an init operation.
type InitResult struct {
	// ProjectName is the name of the initialized project.
	ProjectName string

	// ProjectUUID is the unique identifier assigned to the project.
	ProjectUUID string

	// DeviceName is the name assigned to this device for the project.
	DeviceName string

	// ProjectPath is the root path of the project.
	ProjectPath string
}

// Init initializes a new Kānuka secrets store in the current directory.
//
// It creates the .kanuka directory structure, generates cryptographic keys,
// and registers the current user as the first project member.
//
// Returns ErrProjectAlreadyInitialized if a .kanuka directory already exists.
// Returns errors from key generation or configuration if they fail.
func Init(ctx context.Context, opts InitOptions) (*InitResult, error) {
	kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
	if err != nil {
		return nil, fmt.Errorf("checking project settings: %w", err)
	}
	if kanukaExists {
		return nil, kerrors.ErrProjectAlreadyInitialized
	}

	if err := secrets.EnsureUserSettings(); err != nil {
		return nil, fmt.Errorf("ensuring user settings: %w", err)
	}

	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return nil, fmt.Errorf("ensuring user config: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	projectName := opts.ProjectName
	if projectName == "" {
		projectName = filepath.Base(wd)
	}

	kanukaDir := filepath.Join(wd, ".kanuka")
	cleanupNeeded := false
	defer func() {
		if cleanupNeeded {
			os.RemoveAll(kanukaDir)
		}
	}()

	if err := secrets.EnsureKanukaSettings(); err != nil {
		return nil, fmt.Errorf("creating .kanuka folders: %w", err)
	}
	cleanupNeeded = true

	deviceName, err := utils.GenerateDeviceName([]string{})
	if err != nil {
		return nil, fmt.Errorf("generating device name: %w", err)
	}

	projectConfig := &configs.ProjectConfig{
		Project: configs.Project{
			UUID: configs.GenerateProjectUUID(),
			Name: projectName,
		},
		Users:   make(map[string]string),
		Devices: make(map[string]configs.DeviceConfig),
	}

	projectConfig.Users[userConfig.User.UUID] = userConfig.User.Email
	projectConfig.Devices[userConfig.User.UUID] = configs.DeviceConfig{
		Email:     userConfig.User.Email,
		Name:      deviceName,
		CreatedAt: time.Now().UTC(),
	}

	configs.ProjectKanukaSettings.ProjectPath = wd
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		return nil, fmt.Errorf("saving project config: %w", err)
	}

	if userConfig.Projects == nil {
		userConfig.Projects = make(map[string]configs.UserProjectEntry)
	}
	userConfig.Projects[projectConfig.Project.UUID] = configs.UserProjectEntry{
		DeviceName:  deviceName,
		ProjectName: projectName,
	}
	if err := configs.SaveUserConfig(userConfig); err != nil {
		return nil, fmt.Errorf("updating user config with project: %w", err)
	}

	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	if err := secrets.CreateAndSaveRSAKeyPair(opts.Verbose); err != nil {
		return nil, fmt.Errorf("generating RSA key pair: %w", err)
	}

	if _, err := secrets.CopyUserPublicKeyToProject(); err != nil {
		return nil, fmt.Errorf("copying public key to project: %w", err)
	}

	if err := secrets.CreateAndSaveEncryptedSymmetricKey(opts.Verbose); err != nil {
		return nil, fmt.Errorf("creating encrypted symmetric key: %w", err)
	}

	auditEntry := audit.LogWithUser("init")
	auditEntry.ProjectName = projectName
	auditEntry.ProjectUUID = projectConfig.Project.UUID
	auditEntry.DeviceName = deviceName
	audit.Log(auditEntry)

	cleanupNeeded = false

	return &InitResult{
		ProjectName: projectName,
		ProjectUUID: projectConfig.Project.UUID,
		DeviceName:  deviceName,
		ProjectPath: wd,
	}, nil
}

// CheckUserConfigComplete checks if the user configuration has email and UUID set.
func CheckUserConfigComplete() (bool, error) {
	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		return false, nil
	}
	return userConfig.User.Email != "" && userConfig.User.UUID != "", nil
}
