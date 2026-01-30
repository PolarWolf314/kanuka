package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
)

// UserStatus represents the access status of a user.
type UserStatus string

const (
	// UserStatusActive means the user has both public key and encrypted symmetric key.
	UserStatusActive UserStatus = "active"
	// UserStatusPending means the user has public key but no encrypted symmetric key.
	UserStatusPending UserStatus = "pending"
	// UserStatusOrphan means the user has encrypted symmetric key but no public key.
	UserStatusOrphan UserStatus = "orphan"
)

// UserAccessInfo holds information about a user's access to the project.
type UserAccessInfo struct {
	// UUID is the user's unique identifier.
	UUID string

	// Email is the user's email address.
	Email string

	// DeviceName is the user's device name.
	DeviceName string

	// Status is the user's access status.
	Status UserStatus
}

// AccessSummary holds counts of users by status.
type AccessSummary struct {
	// Active is the count of users with full access.
	Active int

	// Pending is the count of users awaiting access.
	Pending int

	// Orphan is the count of orphaned entries.
	Orphan int
}

// AccessOptions configures the access workflow.
type AccessOptions struct {
	// No options currently needed - included for consistency.
}

// AccessResult contains the outcome of an access operation.
type AccessResult struct {
	// ProjectName is the name of the project.
	ProjectName string

	// Users contains information about each user.
	Users []UserAccessInfo

	// Summary contains counts of users by status.
	Summary AccessSummary
}

// Access lists all users with access to the project's secrets.
//
// It discovers users from the public_keys and secrets directories and determines
// their access status:
//   - active: user has public key AND encrypted symmetric key (can decrypt)
//   - pending: user has public key but NO encrypted symmetric key (run 'sync')
//   - orphan: encrypted symmetric key exists but NO public key (inconsistent)
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrInvalidProjectConfig if the project config is malformed.
func Access(ctx context.Context, opts AccessOptions) (*AccessResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	// Load project config for project name and user email lookup.
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		if strings.Contains(err.Error(), "toml:") {
			return nil, fmt.Errorf("%w: .kanuka/config.toml is not valid TOML", kerrors.ErrInvalidProjectConfig)
		}
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	projectName := projectConfig.Project.Name
	if projectName == "" {
		projectName = configs.ProjectKanukaSettings.ProjectName
	}

	// Discover all users.
	users, err := discoverUsers(projectConfig)
	if err != nil {
		return nil, fmt.Errorf("discovering users: %w", err)
	}

	// Sort users by status (active first, then pending, then orphan), then by email.
	sortUsers(users)

	return &AccessResult{
		ProjectName: projectName,
		Users:       users,
		Summary:     calculateAccessSummary(users),
	}, nil
}

// discoverUsers finds all users from public_keys and secrets directories.
func discoverUsers(projectConfig *configs.ProjectConfig) ([]UserAccessInfo, error) {
	publicKeysDir := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	secretsDir := configs.ProjectKanukaSettings.ProjectSecretsPath

	// Collect all UUIDs from both directories.
	uuidSet := make(map[string]bool)

	// Read public keys directory.
	if entries, err := os.ReadDir(publicKeysDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pub") {
				uuid := strings.TrimSuffix(entry.Name(), ".pub")
				uuidSet[uuid] = true
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading public keys directory: %w", err)
	}

	// Read secrets directory for user .kanuka files.
	if entries, err := os.ReadDir(secretsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".kanuka") {
				uuid := strings.TrimSuffix(entry.Name(), ".kanuka")
				uuidSet[uuid] = true
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading secrets directory: %w", err)
	}

	// Build user info for each UUID.
	var users []UserAccessInfo
	for uuid := range uuidSet {
		status := determineUserStatus(uuid, publicKeysDir, secretsDir)
		email, deviceName := getEmailAndDeviceForUUID(uuid, projectConfig)

		users = append(users, UserAccessInfo{
			UUID:       uuid,
			Email:      email,
			DeviceName: deviceName,
			Status:     status,
		})
	}

	return users, nil
}

// determineUserStatus determines the status of a user based on file existence.
func determineUserStatus(uuid, publicKeysDir, secretsDir string) UserStatus {
	publicKeyPath := filepath.Join(publicKeysDir, uuid+".pub")
	kanukaPath := filepath.Join(secretsDir, uuid+".kanuka")

	hasPublicKey := fileExistsCheck(publicKeyPath)
	hasKanukaFile := fileExistsCheck(kanukaPath)

	switch {
	case hasPublicKey && hasKanukaFile:
		return UserStatusActive
	case hasPublicKey && !hasKanukaFile:
		return UserStatusPending
	case !hasPublicKey && hasKanukaFile:
		return UserStatusOrphan
	default:
		// Should not happen since we're iterating over discovered UUIDs.
		return UserStatusOrphan
	}
}

// fileExistsCheck checks if a file exists.
func fileExistsCheck(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getEmailAndDeviceForUUID looks up the email and device name for a UUID.
func getEmailAndDeviceForUUID(uuid string, projectConfig *configs.ProjectConfig) (string, string) {
	// First try the Devices map (has more detailed info).
	if device, ok := projectConfig.Devices[uuid]; ok {
		return device.Email, device.Name
	}

	// Fall back to the Users map.
	if email, ok := projectConfig.Users[uuid]; ok {
		return email, ""
	}

	// UUID not found in config.
	return "", ""
}

// sortUsers sorts users by status priority (active, pending, orphan), then by email.
func sortUsers(users []UserAccessInfo) {
	statusPriority := map[UserStatus]int{
		UserStatusActive:  0,
		UserStatusPending: 1,
		UserStatusOrphan:  2,
	}

	sort.Slice(users, func(i, j int) bool {
		// First sort by status.
		if statusPriority[users[i].Status] != statusPriority[users[j].Status] {
			return statusPriority[users[i].Status] < statusPriority[users[j].Status]
		}
		// Then by email (or UUID if no email).
		emailI := users[i].Email
		if emailI == "" {
			emailI = users[i].UUID
		}
		emailJ := users[j].Email
		if emailJ == "" {
			emailJ = users[j].UUID
		}
		return emailI < emailJ
	})
}

// calculateAccessSummary calculates the counts of users by status.
func calculateAccessSummary(users []UserAccessInfo) AccessSummary {
	var summary AccessSummary
	for _, user := range users {
		switch user.Status {
		case UserStatusActive:
			summary.Active++
		case UserStatusPending:
			summary.Pending++
		case UserStatusOrphan:
			summary.Orphan++
		}
	}
	return summary
}
