package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/ui"

	"github.com/spf13/cobra"
)

var statusJSONOutput bool

func init() {
	statusCmd.Flags().BoolVar(&statusJSONOutput, "json", false, "output in JSON format")
}

func resetStatusCommandState() {
	statusJSONOutput = false
}

// FileStatus represents the encryption status of a secret file.
type FileStatus string

const (
	// StatusCurrent means the encrypted file is newer than the plaintext.
	StatusCurrent FileStatus = "current"
	// StatusStale means the plaintext was modified after encryption.
	StatusStale FileStatus = "stale"
	// StatusUnencrypted means plaintext exists with no encrypted version.
	StatusUnencrypted FileStatus = "unencrypted"
	// StatusEncryptedOnly means encrypted exists with no plaintext.
	StatusEncryptedOnly FileStatus = "encrypted_only"
)

// FileStatusInfo holds information about a file's encryption status.
type FileStatusInfo struct {
	Path           string     `json:"path"`
	Status         FileStatus `json:"status"`
	PlaintextMtime string     `json:"plaintext_mtime,omitempty"`
	EncryptedMtime string     `json:"encrypted_mtime,omitempty"`
}

// StatusResult holds the result of the status command.
type StatusResult struct {
	ProjectName string           `json:"project"`
	Files       []FileStatusInfo `json:"files"`
	Summary     StatusSummary    `json:"summary"`
}

// StatusSummary holds counts of files by status.
type StatusSummary struct {
	Current       int `json:"current"`
	Stale         int `json:"stale"`
	Unencrypted   int `json:"unencrypted"`
	EncryptedOnly int `json:"encrypted_only"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the encryption status of all secret files",
	Long: `Shows the encryption status of all .env and .kanuka files in the project.

Each file can have one of four statuses:
  - current:        Encrypted file is newer than plaintext (up to date)
  - stale:          Plaintext modified after encryption (needs re-encryption)
  - unencrypted:    Plaintext exists with no encrypted version (security risk)
  - encrypted_only: Encrypted exists with no plaintext (normal after cleanup)

Use --json for machine-readable output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting status command")

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project path: %s", projectPath)

		if projectPath == "" {
			if statusJSONOutput {
				fmt.Println(`{"error": "Kanuka has not been initialized"}`)
				return nil
			}
			fmt.Println(ui.Error.Sprint("✗") + " Kanuka has not been initialized")
			fmt.Println(ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first")
			return nil
		}

		// Load project config for project name.
		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			return Logger.ErrorfAndReturn("failed to load project config: %v", err)
		}
		projectName := projectConfig.Project.Name
		if projectName == "" {
			projectName = configs.ProjectKanukaSettings.ProjectName
		}
		Logger.Debugf("Project name: %s", projectName)

		// Discover all files and their statuses.
		files, err := discoverFileStatuses(projectPath)
		if err != nil {
			return Logger.ErrorfAndReturn("failed to discover file statuses: %v", err)
		}

		// Sort files by path for consistent output.
		sort.Slice(files, func(i, j int) bool {
			return files[i].Path < files[j].Path
		})

		// Build result.
		result := StatusResult{
			ProjectName: projectName,
			Files:       files,
			Summary:     calculateStatusSummary(files),
		}

		// Output results.
		if statusJSONOutput {
			return outputStatusJSON(result)
		}

		printStatusTable(result)
		return nil
	},
}

// discoverFileStatuses finds all .env and .kanuka files and determines their status.
func discoverFileStatuses(projectPath string) ([]FileStatusInfo, error) {
	Logger.Debugf("Discovering file statuses in: %s", projectPath)

	// Find all plaintext .env files (excluding .kanuka directory).
	envFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, false)
	if err != nil {
		return nil, fmt.Errorf("failed to find env files: %w", err)
	}
	Logger.Debugf("Found %d .env files", len(envFiles))

	// Find all encrypted .kanuka files.
	kanukaFiles, err := secrets.FindEnvOrKanukaFiles(projectPath, []string{}, true)
	if err != nil {
		return nil, fmt.Errorf("failed to find kanuka files: %w", err)
	}
	Logger.Debugf("Found %d .kanuka files", len(kanukaFiles))

	// Build a set of all base paths (without .kanuka extension).
	basePaths := make(map[string]bool)
	for _, f := range envFiles {
		basePaths[f] = true
		Logger.Debugf("Found env file: %s", f)
	}
	for _, f := range kanukaFiles {
		basePath := strings.TrimSuffix(f, ".kanuka")
		basePaths[basePath] = true
		Logger.Debugf("Found kanuka file: %s (base: %s)", f, basePath)
	}

	// Determine status for each base path.
	var files []FileStatusInfo
	for basePath := range basePaths {
		status, envMtime, kanukaMtime := determineFileStatus(basePath, projectPath)

		// Convert to relative path for display.
		relPath, err := filepath.Rel(projectPath, basePath)
		if err != nil {
			relPath = basePath
		}

		files = append(files, FileStatusInfo{
			Path:           relPath,
			Status:         status,
			PlaintextMtime: envMtime,
			EncryptedMtime: kanukaMtime,
		})
		Logger.Debugf("File %s: status=%s", relPath, status)
	}

	return files, nil
}

// determineFileStatus determines the encryption status of a file.
func determineFileStatus(basePath, projectPath string) (FileStatus, string, string) {
	kanukaPath := basePath + ".kanuka"

	envInfo, envErr := os.Stat(basePath)
	kanukaInfo, kanukaErr := os.Stat(kanukaPath)

	envExists := envErr == nil
	kanukaExists := kanukaErr == nil

	var envMtime, kanukaMtime string
	if envExists {
		envMtime = envInfo.ModTime().Format("2006-01-02T15:04:05Z07:00")
	}
	if kanukaExists {
		kanukaMtime = kanukaInfo.ModTime().Format("2006-01-02T15:04:05Z07:00")
	}

	switch {
	case envExists && kanukaExists:
		// Both exist - check modification times.
		if kanukaInfo.ModTime().After(envInfo.ModTime()) {
			return StatusCurrent, envMtime, kanukaMtime
		}
		return StatusStale, envMtime, kanukaMtime

	case envExists && !kanukaExists:
		return StatusUnencrypted, envMtime, ""

	case !envExists && kanukaExists:
		return StatusEncryptedOnly, "", kanukaMtime

	default:
		// Neither exists - shouldn't happen.
		return StatusUnencrypted, "", ""
	}
}

// calculateStatusSummary calculates the counts of files by status.
func calculateStatusSummary(files []FileStatusInfo) StatusSummary {
	var summary StatusSummary
	for _, file := range files {
		switch file.Status {
		case StatusCurrent:
			summary.Current++
		case StatusStale:
			summary.Stale++
		case StatusUnencrypted:
			summary.Unencrypted++
		case StatusEncryptedOnly:
			summary.EncryptedOnly++
		}
	}
	return summary
}

// outputStatusJSON outputs the result as JSON.
func outputStatusJSON(result StatusResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// printStatusTable prints a formatted table of file statuses.
func printStatusTable(result StatusResult) {
	fmt.Printf("Project: %s\n", ui.Highlight.Sprint(result.ProjectName))
	fmt.Println()

	if len(result.Files) == 0 {
		fmt.Println(ui.Success.Sprint("✓") + " No secret files found.")
		return
	}

	fmt.Println("Secret files status:")
	fmt.Println()

	// Calculate column width for file path.
	pathWidth := 30
	for _, file := range result.Files {
		if len(file.Path) > pathWidth {
			pathWidth = len(file.Path)
		}
	}
	// Cap at reasonable width.
	if pathWidth > 60 {
		pathWidth = 60
	}

	// Print header.
	fmt.Printf("  %-*s  %s\n", pathWidth, "FILE", "STATUS")

	// Print files.
	for _, file := range result.Files {
		displayPath := file.Path
		if len(displayPath) > pathWidth {
			displayPath = "..." + displayPath[len(displayPath)-pathWidth+3:]
		}

		var statusStr string
		switch file.Status {
		case StatusCurrent:
			statusStr = ui.Success.Sprint("✓") + " encrypted (up to date)"
		case StatusStale:
			statusStr = ui.Warning.Sprint("⚠") + " stale (plaintext modified after encryption)"
		case StatusUnencrypted:
			statusStr = ui.Error.Sprint("✗") + " not encrypted"
		case StatusEncryptedOnly:
			statusStr = ui.Muted.Sprint("◌") + " encrypted only (no plaintext)"
		}

		fmt.Printf("  %-*s  %s\n", pathWidth, displayPath, statusStr)
	}

	// Print summary.
	fmt.Println()
	fmt.Println("Summary:")

	if result.Summary.Current > 0 {
		fmt.Printf("  %d file(s) up to date\n", result.Summary.Current)
	}
	if result.Summary.Stale > 0 {
		fmt.Printf("  %d file(s) stale (run '%s' to update)\n",
			result.Summary.Stale, ui.Code.Sprint("kanuka secrets encrypt"))
	}
	if result.Summary.Unencrypted > 0 {
		fmt.Printf("  %d file(s) not encrypted (run '%s' to secure)\n",
			result.Summary.Unencrypted, ui.Code.Sprint("kanuka secrets encrypt"))
	}
	if result.Summary.EncryptedOnly > 0 {
		fmt.Printf("  %d file(s) encrypted only (plaintext removed, this is normal)\n", result.Summary.EncryptedOnly)
	}
}
