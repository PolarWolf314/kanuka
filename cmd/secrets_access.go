package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/spf13/cobra"
)

var accessJSONOutput bool

func init() {
	accessCmd.Flags().BoolVar(&accessJSONOutput, "json", false, "output in JSON format")
}

func resetAccessCommandState() {
	accessJSONOutput = false
}

// UserStatus represents the access status of a user.
type UserStatus string

const (
	// StatusActive means the user has both public key and encrypted symmetric key.
	StatusActive UserStatus = "active"
	// StatusPending means the user has public key but no encrypted symmetric key.
	StatusPending UserStatus = "pending"
	// StatusOrphan means the user has encrypted symmetric key but no public key.
	StatusOrphan UserStatus = "orphan"
)

// UserAccessInfo holds information about a user's access to the project.
type UserAccessInfo struct {
	UUID       string     `json:"uuid"`
	Email      string     `json:"email"`
	DeviceName string     `json:"device_name,omitempty"`
	Status     UserStatus `json:"status"`
}

// AccessResult holds the result of the access command.
type AccessResult struct {
	ProjectName string           `json:"project"`
	Users       []UserAccessInfo `json:"users"`
	Summary     AccessSummary    `json:"summary"`
}

// AccessSummary holds counts of users by status.
type AccessSummary struct {
	Active  int `json:"active"`
	Pending int `json:"pending"`
	Orphan  int `json:"orphan"`
}

var accessCmd = &cobra.Command{
	Use:   "access",
	Short: "List all users with access to this project's secrets",
	Long: `Shows all users who have access to decrypt secrets in this project.

Each user can have one of three statuses:
  - active:  User has public key AND encrypted symmetric key (can decrypt)
  - pending: User has public key but NO encrypted symmetric key (run 'sync')
  - orphan:  Encrypted symmetric key exists but NO public key (inconsistent)

Use --json for machine-readable output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting access command")

		spinner, cleanup := startSpinner("Discovering users with access...", verbose)
		defer cleanup()

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to initialize project settings."
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project path: %s", projectPath)

		if projectPath == "" {
			if accessJSONOutput {
				fmt.Println(`{"error": "Kanuka has not been initialized"}`)
				return nil
			}
			spinner.FinalMSG = ui.Error.Sprint("✗") + " Kanuka has not been initialized."
			fmt.Println(ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first")
			return nil
		}

		// Load project config for project name and user email lookup.
		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			if strings.Contains(err.Error(), "toml:") {
				if accessJSONOutput {
					fmt.Println(`{"error": "Failed to load project configuration: config.toml is not valid TOML"}`)
					return nil
				}
				spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to load project configuration."
				fmt.Println()
				fmt.Println(ui.Info.Sprint("→") + " The .kanuka/config.toml file is not valid TOML.")
				fmt.Println("   " + ui.Code.Sprint(err.Error()))
				fmt.Println()
				fmt.Println("   To fix this issue:")
				fmt.Println("   1. Restore the file from git: " + ui.Code.Sprint("git checkout .kanuka/config.toml"))
				fmt.Println("   2. Or contact your project administrator for assistance")
				return nil
			}
			spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to load project configuration\n"
			return Logger.ErrorfAndReturn("failed to load project config: %v", err)
		}
		projectName := projectConfig.Project.Name
		if projectName == "" {
			projectName = configs.ProjectKanukaSettings.ProjectName
		}
		Logger.Debugf("Project name: %s", projectName)

		// Discover all users.
		users, err := discoverUsers(projectConfig)
		if err != nil {
			return Logger.ErrorfAndReturn("failed to discover users: %v", err)
		}

		// Sort users by status (active first, then pending, then orphan), then by email.
		sortUsers(users)

		// Build result.
		result := AccessResult{
			ProjectName: projectName,
			Users:       users,
			Summary:     calculateSummary(users),
		}

		// Output results.
		if accessJSONOutput {
			if err := outputJSON(result); err != nil {
				spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to output access information."
				return err
			}
			return nil
		}

		printAccessTable(result)
		spinner.FinalMSG = ui.Success.Sprint("✓") + " Access information displayed."
		return nil
	},
}

// discoverUsers finds all users from public_keys and secrets directories.
func discoverUsers(projectConfig *configs.ProjectConfig) ([]UserAccessInfo, error) {
	publicKeysDir := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	secretsDir := configs.ProjectKanukaSettings.ProjectSecretsPath

	Logger.Debugf("Scanning public keys dir: %s", publicKeysDir)
	Logger.Debugf("Scanning secrets dir: %s", secretsDir)

	// Collect all UUIDs from both directories.
	uuidSet := make(map[string]bool)

	// Read public keys directory.
	if entries, err := os.ReadDir(publicKeysDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pub") {
				uuid := strings.TrimSuffix(entry.Name(), ".pub")
				uuidSet[uuid] = true
				Logger.Debugf("Found public key for UUID: %s", uuid)
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read public keys directory: %w", err)
	}

	// Read secrets directory for user .kanuka files.
	if entries, err := os.ReadDir(secretsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".kanuka") {
				uuid := strings.TrimSuffix(entry.Name(), ".kanuka")
				uuidSet[uuid] = true
				Logger.Debugf("Found kanuka file for UUID: %s", uuid)
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read secrets directory: %w", err)
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
		Logger.Debugf("User %s: email=%s, device=%s, status=%s", uuid, email, deviceName, status)
	}

	return users, nil
}

// determineUserStatus determines the status of a user based on file existence.
func determineUserStatus(uuid, publicKeysDir, secretsDir string) UserStatus {
	publicKeyPath := filepath.Join(publicKeysDir, uuid+".pub")
	kanukaPath := filepath.Join(secretsDir, uuid+".kanuka")

	hasPublicKey := fileExists(publicKeyPath)
	hasKanukaFile := fileExists(kanukaPath)

	switch {
	case hasPublicKey && hasKanukaFile:
		return StatusActive
	case hasPublicKey && !hasKanukaFile:
		return StatusPending
	case !hasPublicKey && hasKanukaFile:
		return StatusOrphan
	default:
		// Should not happen since we're iterating over discovered UUIDs.
		return StatusOrphan
	}
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
		StatusActive:  0,
		StatusPending: 1,
		StatusOrphan:  2,
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

// calculateSummary calculates the counts of users by status.
func calculateSummary(users []UserAccessInfo) AccessSummary {
	var summary AccessSummary
	for _, user := range users {
		switch user.Status {
		case StatusActive:
			summary.Active++
		case StatusPending:
			summary.Pending++
		case StatusOrphan:
			summary.Orphan++
		}
	}
	return summary
}

// outputJSON outputs the result as JSON.
func outputJSON(result AccessResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// printAccessTable prints a formatted table of users with access.
func printAccessTable(result AccessResult) {
	fmt.Printf("Project: %s\n", ui.Highlight.Sprint(result.ProjectName))
	fmt.Println()

	if len(result.Users) == 0 {
		fmt.Println("No users found.")
		return
	}

	fmt.Println("Users with access:")
	fmt.Println()

	// Calculate column widths.
	uuidWidth := 36 // Standard UUID length.
	emailWidth := 25
	for _, user := range result.Users {
		displayEmail := user.Email
		if user.DeviceName != "" {
			displayEmail = fmt.Sprintf("%s (%s)", user.Email, user.DeviceName)
		}
		if len(displayEmail) > emailWidth {
			emailWidth = len(displayEmail)
		}
	}

	// Print header.
	fmt.Printf("  %-*s  %-*s  %s\n", uuidWidth, "UUID", emailWidth, "EMAIL", "STATUS")

	// Print users.
	for _, user := range result.Users {
		displayEmail := user.Email
		if displayEmail == "" {
			displayEmail = ui.Muted.Sprint("unknown")
		} else if user.DeviceName != "" {
			displayEmail = fmt.Sprintf("%s (%s)", user.Email, user.DeviceName)
		}

		var statusStr string
		switch user.Status {
		case StatusActive:
			statusStr = ui.Success.Sprint("✓") + " active"
		case StatusPending:
			statusStr = ui.Warning.Sprint("⚠") + " pending"
		case StatusOrphan:
			statusStr = ui.Error.Sprint("✗") + " orphan"
		}

		fmt.Printf("  %-*s  %-*s  %s\n", uuidWidth, user.UUID, emailWidth, displayEmail, statusStr)
	}

	// Print legend.
	fmt.Println()
	fmt.Println("Legend:")
	fmt.Printf("  %s active  - User has public key and encrypted symmetric key\n", ui.Success.Sprint("✓"))
	fmt.Printf("  %s pending - User has public key but no encrypted symmetric key (run 'sync')\n", ui.Warning.Sprint("⚠"))
	fmt.Printf("  %s orphan  - Encrypted symmetric key exists but no public key (inconsistent)\n", ui.Error.Sprint("✗"))

	// Print summary.
	fmt.Println()
	parts := []string{}
	if result.Summary.Active > 0 {
		parts = append(parts, fmt.Sprintf("%d active", result.Summary.Active))
	}
	if result.Summary.Pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", result.Summary.Pending))
	}
	if result.Summary.Orphan > 0 {
		parts = append(parts, fmt.Sprintf("%d orphan", result.Summary.Orphan))
	}

	total := len(result.Users)
	if len(parts) > 0 {
		fmt.Printf("Total: %d user(s) (%s)\n", total, strings.Join(parts, ", "))
	} else {
		fmt.Printf("Total: %d user(s)\n", total)
	}

	// Print tip for orphans if any exist.
	if result.Summary.Orphan > 0 {
		fmt.Println()
		fmt.Println(ui.Info.Sprint("Tip:") + " Run '" + ui.Code.Sprint("kanuka secrets clean") + "' to remove orphaned entries.")
	}
}
