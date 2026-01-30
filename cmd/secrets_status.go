package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/workflows"

	"github.com/spf13/cobra"
)

var statusJSONOutput bool

func init() {
	statusCmd.Flags().BoolVar(&statusJSONOutput, "json", false, "output in JSON format")
}

func resetStatusCommandState() {
	statusJSONOutput = false
}

// statusJSONResult holds the JSON-serializable status result.
type statusJSONResult struct {
	ProjectName string            `json:"project"`
	Files       []statusJSONFile  `json:"files"`
	Summary     statusJSONSummary `json:"summary"`
}

type statusJSONFile struct {
	Path           string `json:"path"`
	Status         string `json:"status"`
	PlaintextMtime string `json:"plaintext_mtime,omitempty"`
	EncryptedMtime string `json:"encrypted_mtime,omitempty"`
}

type statusJSONSummary struct {
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

		spinner, cleanup := startSpinner("Checking file statuses...", verbose)
		defer cleanup()

		result, err := workflows.Status(context.Background(), workflows.StatusOptions{})
		if err != nil {
			if statusJSONOutput {
				fmt.Printf(`{"error": "%s"}`+"\n", formatStatusErrorJSON(err))
				return nil
			}
			spinner.FinalMSG = formatStatusError(err)
			if isStatusUnexpectedError(err) {
				return err
			}
			return nil
		}

		// Output results.
		if statusJSONOutput {
			if err := outputStatusJSON(result); err != nil {
				spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to output status."
				return err
			}
		} else {
			printStatusTable(result)
			spinner.FinalMSG = ui.Success.Sprint("✓") + " Status displayed."
		}
		return nil
	},
}

// formatStatusError formats workflow errors into user-friendly messages.
func formatStatusError(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kānuka has not been initialized.\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first"

	case errors.Is(err, kerrors.ErrInvalidProjectConfig):
		return ui.Error.Sprint("✗") + " Failed to load project configuration.\n\n" +
			ui.Info.Sprint("→") + " The .kanuka/config.toml file is not valid TOML.\n\n" +
			"   To fix this issue:\n" +
			"   1. Restore the file from git: " + ui.Code.Sprint("git checkout .kanuka/config.toml") + "\n" +
			"   2. Or contact your project administrator for assistance"

	default:
		return ui.Error.Sprint("✗") + " Failed to check status\n" +
			ui.Error.Sprint("Error: ") + err.Error()
	}
}

// formatStatusErrorJSON formats errors for JSON output.
func formatStatusErrorJSON(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return "Kanuka has not been initialized"

	case errors.Is(err, kerrors.ErrInvalidProjectConfig):
		return "Failed to load project configuration: config.toml is not valid TOML"

	default:
		return err.Error()
	}
}

// isStatusUnexpectedError returns true if the error is unexpected and should cause a non-zero exit.
func isStatusUnexpectedError(err error) bool {
	expectedErrors := []error{
		kerrors.ErrProjectNotInitialized,
		kerrors.ErrInvalidProjectConfig,
	}

	for _, expected := range expectedErrors {
		if errors.Is(err, expected) {
			return false
		}
	}
	return true
}

// outputStatusJSON outputs the result as JSON.
func outputStatusJSON(result *workflows.StatusResult) error {
	// Convert to JSON-serializable format.
	jsonResult := statusJSONResult{
		ProjectName: result.ProjectName,
		Files:       make([]statusJSONFile, len(result.Files)),
		Summary: statusJSONSummary{
			Current:       result.Summary.Current,
			Stale:         result.Summary.Stale,
			Unencrypted:   result.Summary.Unencrypted,
			EncryptedOnly: result.Summary.EncryptedOnly,
		},
	}

	for i, f := range result.Files {
		jsonResult.Files[i] = statusJSONFile{
			Path:           f.Path,
			Status:         string(f.Status),
			PlaintextMtime: f.PlaintextMtime,
			EncryptedMtime: f.EncryptedMtime,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonResult)
}

// printStatusTable prints a formatted table of file statuses.
func printStatusTable(result *workflows.StatusResult) {
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
		case workflows.StatusCurrent:
			statusStr = ui.Success.Sprint("✓") + " encrypted (up to date)"
		case workflows.StatusStale:
			statusStr = ui.Warning.Sprint("⚠") + " stale (plaintext modified after encryption)"
		case workflows.StatusUnencrypted:
			statusStr = ui.Error.Sprint("✗") + " not encrypted"
		case workflows.StatusEncryptedOnly:
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
