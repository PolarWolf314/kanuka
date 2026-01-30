package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/utils"
)

// CheckStatus represents the result status of a health check.
type CheckStatus int

const (
	// CheckPass means the check passed.
	CheckPass CheckStatus = iota
	// CheckWarning means the check found a non-critical issue.
	CheckWarning
	// CheckError means the check found a critical issue.
	CheckError
)

// String returns a string representation of CheckStatus.
func (s CheckStatus) String() string {
	switch s {
	case CheckPass:
		return "pass"
	case CheckWarning:
		return "warning"
	case CheckError:
		return "error"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler for CheckStatus.
func (s CheckStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// CheckResult holds the result of a single health check.
type CheckResult struct {
	Name       string      `json:"name"`
	Status     CheckStatus `json:"status"`
	Message    string      `json:"message"`
	Suggestion string      `json:"suggestion,omitempty"`
}

// DoctorResult holds the complete result of the doctor workflow.
type DoctorResult struct {
	Checks      []CheckResult `json:"checks"`
	Summary     DoctorSummary `json:"summary"`
	Suggestions []string      `json:"suggestions,omitempty"`
}

// DoctorSummary holds counts of checks by status.
type DoctorSummary struct {
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Errors   int `json:"errors"`
}

// DoctorOptions configures the doctor workflow.
type DoctorOptions struct {
	// No options currently, but provides extensibility.
}

// Doctor runs health checks on the Kanuka project.
//
// The doctor workflow checks:
//   - Project configuration validity
//   - User configuration validity
//   - Private key existence and permissions
//   - Public key and encrypted symmetric key consistency
//   - Gitignore configuration for .env files
//   - Unencrypted .env files
func Doctor(ctx context.Context, opts DoctorOptions) (*DoctorResult, error) {
	// Run all health checks.
	checks := []func() CheckResult{
		checkProjectConfig,
		checkUserConfig,
		checkPrivateKeyExists,
		checkPrivateKeyPermissions,
		checkPublicKeyConsistency,
		checkKanukaFileConsistency,
		checkGitignore,
		checkUnencryptedFiles,
	}

	var results []CheckResult
	for _, check := range checks {
		result := check()
		results = append(results, result)
	}

	// Calculate summary.
	summary := calculateDoctorSummary(results)

	// Collect suggestions (deduplicated).
	var suggestions []string
	seen := make(map[string]bool)
	for _, result := range results {
		if result.Suggestion != "" && result.Status != CheckPass && !seen[result.Suggestion] {
			suggestions = append(suggestions, result.Suggestion)
			seen[result.Suggestion] = true
		}
	}

	return &DoctorResult{
		Checks:      results,
		Summary:     summary,
		Suggestions: suggestions,
	}, nil
}

// checkProjectConfig checks if the project config exists and parses correctly.
func checkProjectConfig() CheckResult {
	projectPath, err := utils.FindProjectKanukaRoot()
	if err != nil || projectPath == "" {
		return CheckResult{
			Name:       "Project configuration",
			Status:     CheckError,
			Message:    "Kanuka project not found",
			Suggestion: "Run 'kanuka secrets init' to initialize a project",
		}
	}

	configPath := filepath.Join(projectPath, ".kanuka", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return CheckResult{
			Name:       "Project configuration",
			Status:     CheckError,
			Message:    "Project config.toml not found",
			Suggestion: "Run 'kanuka secrets init' to initialize the project",
		}
	}

	// Try to load the config.
	projectConfig := &configs.ProjectConfig{
		Users:   make(map[string]string),
		Devices: make(map[string]configs.DeviceConfig),
	}
	if err := configs.LoadTOML(configPath, projectConfig); err != nil {
		return CheckResult{
			Name:       "Project configuration",
			Status:     CheckError,
			Message:    fmt.Sprintf("Failed to parse project config: %v", err),
			Suggestion: "Check the .kanuka/config.toml file for syntax errors",
		}
	}

	if projectConfig.Project.UUID == "" {
		return CheckResult{
			Name:       "Project configuration",
			Status:     CheckError,
			Message:    "Project UUID is missing from config",
			Suggestion: "Re-initialize the project with 'kanuka secrets init'",
		}
	}

	return CheckResult{
		Name:    "Project configuration",
		Status:  CheckPass,
		Message: "Project configuration valid",
	}
}

// checkUserConfig checks if the user config exists and parses correctly.
func checkUserConfig() CheckResult {
	if configs.UserKanukaSettings == nil || configs.UserKanukaSettings.UserConfigsPath == "" {
		return CheckResult{
			Name:       "User configuration",
			Status:     CheckError,
			Message:    "User settings not initialized",
			Suggestion: "Run 'kanuka secrets init' in a project first",
		}
	}

	configPath := filepath.Join(configs.UserKanukaSettings.UserConfigsPath, "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return CheckResult{
			Name:       "User configuration",
			Status:     CheckError,
			Message:    "User config.toml not found",
			Suggestion: "Run 'kanuka secrets init' to create user configuration",
		}
	}

	// Try to load the config.
	userConfig := &configs.UserConfig{
		Projects: make(map[string]configs.UserProjectEntry),
	}
	if err := configs.LoadTOML(configPath, userConfig); err != nil {
		return CheckResult{
			Name:       "User configuration",
			Status:     CheckError,
			Message:    fmt.Sprintf("Failed to parse user config: %v", err),
			Suggestion: "Check the user config.toml file for syntax errors",
		}
	}

	if userConfig.User.UUID == "" {
		return CheckResult{
			Name:       "User configuration",
			Status:     CheckWarning,
			Message:    "User UUID is missing from config",
			Suggestion: "Run 'kanuka secrets init' to generate a user UUID",
		}
	}

	return CheckResult{
		Name:    "User configuration",
		Status:  CheckPass,
		Message: "User configuration valid",
	}
}

// checkPrivateKeyExists checks if the private key exists for the current project.
func checkPrivateKeyExists() CheckResult {
	projectUUID := getProjectUUID()
	if projectUUID == "" {
		return CheckResult{
			Name:       "Private key exists",
			Status:     CheckError,
			Message:    "Cannot check private key: project not initialized",
			Suggestion: "Run 'kanuka secrets init' to initialize a project",
		}
	}

	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		return CheckResult{
			Name:       "Private key exists",
			Status:     CheckError,
			Message:    "Private key not found for this project",
			Suggestion: "Run 'kanuka secrets init' or register with this project",
		}
	}

	return CheckResult{
		Name:    "Private key exists",
		Status:  CheckPass,
		Message: "Private key exists for this project",
	}
}

// checkPrivateKeyPermissions checks if the private key has secure permissions.
func checkPrivateKeyPermissions() CheckResult {
	projectUUID := getProjectUUID()
	if projectUUID == "" {
		return CheckResult{
			Name:       "Private key permissions",
			Status:     CheckError,
			Message:    "Cannot check private key permissions: project not initialized",
			Suggestion: "Run 'kanuka secrets init' to initialize a project",
		}
	}

	privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
	info, err := os.Stat(privateKeyPath)
	if os.IsNotExist(err) {
		return CheckResult{
			Name:       "Private key permissions",
			Status:     CheckError,
			Message:    "Private key not found (skipping permissions check)",
			Suggestion: "Run 'kanuka secrets init' or register with this project",
		}
	}
	if err != nil {
		return CheckResult{
			Name:       "Private key permissions",
			Status:     CheckError,
			Message:    fmt.Sprintf("Failed to stat private key: %v", err),
			Suggestion: "Check that the private key file is accessible",
		}
	}

	// Check permissions (should be 0600).
	mode := info.Mode().Perm()
	if mode != 0600 {
		return CheckResult{
			Name:       "Private key permissions",
			Status:     CheckWarning,
			Message:    fmt.Sprintf("Private key has insecure permissions (%04o)", mode),
			Suggestion: fmt.Sprintf("Run 'chmod 600 %s' to fix permissions", privateKeyPath),
		}
	}

	return CheckResult{
		Name:    "Private key permissions",
		Status:  CheckPass,
		Message: "Private key has correct permissions (0600)",
	}
}

// checkPublicKeyConsistency checks if every public key has a corresponding .kanuka file.
func checkPublicKeyConsistency() CheckResult {
	projectPath, err := utils.FindProjectKanukaRoot()
	if err != nil || projectPath == "" {
		return CheckResult{
			Name:       "Public key consistency",
			Status:     CheckError,
			Message:    "Kanuka project not found",
			Suggestion: "Run 'kanuka secrets init' to initialize a project",
		}
	}

	publicKeysDir := filepath.Join(projectPath, ".kanuka", "public_keys")
	secretsDir := filepath.Join(projectPath, ".kanuka", "secrets")

	// Read public keys directory.
	entries, err := os.ReadDir(publicKeysDir)
	if os.IsNotExist(err) {
		return CheckResult{
			Name:    "Public key consistency",
			Status:  CheckPass,
			Message: "No public keys directory (no users registered)",
		}
	}
	if err != nil {
		return CheckResult{
			Name:       "Public key consistency",
			Status:     CheckError,
			Message:    fmt.Sprintf("Failed to read public keys directory: %v", err),
			Suggestion: "Check that the .kanuka/public_keys directory is accessible",
		}
	}

	// Check each public key has a corresponding .kanuka file.
	var missingKanukaFiles []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pub") {
			continue
		}
		uuid := strings.TrimSuffix(entry.Name(), ".pub")
		kanukaPath := filepath.Join(secretsDir, uuid+".kanuka")
		if _, err := os.Stat(kanukaPath); os.IsNotExist(err) {
			missingKanukaFiles = append(missingKanukaFiles, uuid)
		}
	}

	if len(missingKanukaFiles) > 0 {
		return CheckResult{
			Name:       "Public key consistency",
			Status:     CheckWarning,
			Message:    fmt.Sprintf("%d public key(s) without encrypted symmetric key (pending users)", len(missingKanukaFiles)),
			Suggestion: "Run 'kanuka secrets sync' to generate symmetric keys for pending users",
		}
	}

	return CheckResult{
		Name:    "Public key consistency",
		Status:  CheckPass,
		Message: "All public keys have corresponding encrypted symmetric keys",
	}
}

// checkKanukaFileConsistency checks if every user .kanuka file has a corresponding public key.
func checkKanukaFileConsistency() CheckResult {
	projectPath, err := utils.FindProjectKanukaRoot()
	if err != nil || projectPath == "" {
		return CheckResult{
			Name:       "Encrypted key consistency",
			Status:     CheckError,
			Message:    "Kanuka project not found",
			Suggestion: "Run 'kanuka secrets init' to initialize a project",
		}
	}

	publicKeysDir := filepath.Join(projectPath, ".kanuka", "public_keys")
	secretsDir := filepath.Join(projectPath, ".kanuka", "secrets")

	// Read secrets directory for user .kanuka files.
	entries, err := os.ReadDir(secretsDir)
	if os.IsNotExist(err) {
		return CheckResult{
			Name:    "Encrypted key consistency",
			Status:  CheckPass,
			Message: "No secrets directory (no users registered)",
		}
	}
	if err != nil {
		return CheckResult{
			Name:       "Encrypted key consistency",
			Status:     CheckError,
			Message:    fmt.Sprintf("Failed to read secrets directory: %v", err),
			Suggestion: "Check that the .kanuka/secrets directory is accessible",
		}
	}

	// Check each user .kanuka file has a corresponding public key.
	var orphanedKanukaFiles []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".kanuka") {
			continue
		}
		uuid := strings.TrimSuffix(entry.Name(), ".kanuka")
		publicKeyPath := filepath.Join(publicKeysDir, uuid+".pub")
		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			orphanedKanukaFiles = append(orphanedKanukaFiles, uuid)
		}
	}

	if len(orphanedKanukaFiles) > 0 {
		return CheckResult{
			Name:       "Encrypted key consistency",
			Status:     CheckError,
			Message:    fmt.Sprintf("%d encrypted symmetric key(s) without public key (orphaned)", len(orphanedKanukaFiles)),
			Suggestion: "Run 'kanuka secrets clean' to remove orphaned entries",
		}
	}

	return CheckResult{
		Name:    "Encrypted key consistency",
		Status:  CheckPass,
		Message: "All encrypted symmetric keys have corresponding public keys",
	}
}

// checkGitignore checks if .env patterns are in .gitignore.
func checkGitignore() CheckResult {
	projectPath, err := utils.FindProjectKanukaRoot()
	if err != nil || projectPath == "" {
		return CheckResult{
			Name:       "Gitignore configuration",
			Status:     CheckError,
			Message:    "Kanuka project not found",
			Suggestion: "Run 'kanuka secrets init' to initialize a project",
		}
	}

	gitignorePath := filepath.Join(projectPath, ".gitignore")

	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		return CheckResult{
			Name:       "Gitignore configuration",
			Status:     CheckWarning,
			Message:    "No .gitignore file found",
			Suggestion: "Create a .gitignore file with: .env, .env.*, and !*.kanuka",
		}
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return CheckResult{
			Name:       "Gitignore configuration",
			Status:     CheckError,
			Message:    fmt.Sprintf("Failed to read .gitignore: %v", err),
			Suggestion: "Check that the .gitignore file is accessible",
		}
	}

	// Check for .env patterns.
	lines := strings.Split(string(content), "\n")
	hasEnvPattern := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Check for common .env ignore patterns.
		if strings.Contains(line, ".env") {
			hasEnvPattern = true
			break
		}
	}

	if !hasEnvPattern {
		return CheckResult{
			Name:       "Gitignore configuration",
			Status:     CheckWarning,
			Message:    ".env patterns not found in .gitignore",
			Suggestion: "Add to .gitignore: .env, .env.*, and !*.kanuka (to keep encrypted files)",
		}
	}

	return CheckResult{
		Name:    "Gitignore configuration",
		Status:  CheckPass,
		Message: ".env patterns found in .gitignore",
	}
}

// checkUnencryptedFiles checks for unencrypted .env files.
func checkUnencryptedFiles() CheckResult {
	projectPath, err := utils.FindProjectKanukaRoot()
	if err != nil || projectPath == "" {
		return CheckResult{
			Name:       "Unencrypted files",
			Status:     CheckError,
			Message:    "Kanuka project not found",
			Suggestion: "Run 'kanuka secrets init' to initialize a project",
		}
	}

	// Find all .env files.
	envFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, false)
	if err != nil {
		return CheckResult{
			Name:       "Unencrypted files",
			Status:     CheckError,
			Message:    fmt.Sprintf("Failed to find .env files: %v", err),
			Suggestion: "Check that the project directory is accessible",
		}
	}

	// Check each .env file for a corresponding .kanuka file.
	var unencryptedFiles []string
	for _, envFile := range envFiles {
		kanukaFile := envFile + ".kanuka"
		if _, err := os.Stat(kanukaFile); os.IsNotExist(err) {
			// Make path relative for display.
			relPath, err := filepath.Rel(projectPath, envFile)
			if err != nil {
				relPath = envFile
			}
			unencryptedFiles = append(unencryptedFiles, relPath)
		}
	}

	if len(unencryptedFiles) > 0 {
		return CheckResult{
			Name:       "Unencrypted files",
			Status:     CheckWarning,
			Message:    fmt.Sprintf("Found %d unencrypted .env file(s)", len(unencryptedFiles)),
			Suggestion: "Run 'kanuka secrets encrypt' to encrypt unprotected files",
		}
	}

	return CheckResult{
		Name:    "Unencrypted files",
		Status:  CheckPass,
		Message: "All .env files have encrypted versions",
	}
}

// getProjectUUID returns the project UUID from the project config.
func getProjectUUID() string {
	projectPath, err := utils.FindProjectKanukaRoot()
	if err != nil || projectPath == "" {
		return ""
	}

	configPath := filepath.Join(projectPath, ".kanuka", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return ""
	}

	projectConfig := &configs.ProjectConfig{
		Users:   make(map[string]string),
		Devices: make(map[string]configs.DeviceConfig),
	}
	if err := configs.LoadTOML(configPath, projectConfig); err != nil {
		return ""
	}

	return projectConfig.Project.UUID
}

// calculateDoctorSummary calculates the counts of checks by status.
func calculateDoctorSummary(results []CheckResult) DoctorSummary {
	var summary DoctorSummary
	for _, result := range results {
		switch result.Status {
		case CheckPass:
			summary.Passed++
		case CheckWarning:
			summary.Warnings++
		case CheckError:
			summary.Errors++
		}
	}
	return summary
}
