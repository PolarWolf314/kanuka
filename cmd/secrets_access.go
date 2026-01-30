package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/workflows"

	"github.com/spf13/cobra"
)

var accessJSONOutput bool

func init() {
	accessCmd.Flags().BoolVar(&accessJSONOutput, "json", false, "output in JSON format")
}

func resetAccessCommandState() {
	accessJSONOutput = false
}

// accessJSONResult holds the JSON-serializable access result.
type accessJSONResult struct {
	ProjectName string            `json:"project"`
	Users       []accessJSONUser  `json:"users"`
	Summary     accessJSONSummary `json:"summary"`
}

type accessJSONUser struct {
	UUID       string `json:"uuid"`
	Email      string `json:"email"`
	DeviceName string `json:"device_name,omitempty"`
	Status     string `json:"status"`
}

type accessJSONSummary struct {
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

		result, err := workflows.Access(context.Background(), workflows.AccessOptions{})
		if err != nil {
			if accessJSONOutput {
				fmt.Printf(`{"error": "%s"}`+"\n", formatAccessErrorJSON(err))
				return nil
			}
			spinner.FinalMSG = formatAccessError(err)
			if isAccessUnexpectedError(err) {
				return err
			}
			return nil
		}

		// Output results.
		if accessJSONOutput {
			if err := outputAccessJSON(result); err != nil {
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

// formatAccessError formats workflow errors into user-friendly messages.
func formatAccessError(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kanuka has not been initialized.\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first"

	case errors.Is(err, kerrors.ErrInvalidProjectConfig):
		return ui.Error.Sprint("✗") + " Failed to load project configuration.\n\n" +
			ui.Info.Sprint("→") + " The .kanuka/config.toml file is not valid TOML.\n\n" +
			"   To fix this issue:\n" +
			"   1. Restore the file from git: " + ui.Code.Sprint("git checkout .kanuka/config.toml") + "\n" +
			"   2. Or contact your project administrator for assistance"

	default:
		return ui.Error.Sprint("✗") + " Failed to list access\n" +
			ui.Error.Sprint("Error: ") + err.Error()
	}
}

// formatAccessErrorJSON formats errors for JSON output.
func formatAccessErrorJSON(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return "Kanuka has not been initialized"

	case errors.Is(err, kerrors.ErrInvalidProjectConfig):
		return "Failed to load project configuration: config.toml is not valid TOML"

	default:
		return err.Error()
	}
}

// isAccessUnexpectedError returns true if the error is unexpected and should cause a non-zero exit.
func isAccessUnexpectedError(err error) bool {
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

// outputAccessJSON outputs the result as JSON.
func outputAccessJSON(result *workflows.AccessResult) error {
	// Convert to JSON-serializable format.
	jsonResult := accessJSONResult{
		ProjectName: result.ProjectName,
		Users:       make([]accessJSONUser, len(result.Users)),
		Summary: accessJSONSummary{
			Active:  result.Summary.Active,
			Pending: result.Summary.Pending,
			Orphan:  result.Summary.Orphan,
		},
	}

	for i, u := range result.Users {
		jsonResult.Users[i] = accessJSONUser{
			UUID:       u.UUID,
			Email:      u.Email,
			DeviceName: u.DeviceName,
			Status:     string(u.Status),
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonResult)
}

// printAccessTable prints a formatted table of users with access.
func printAccessTable(result *workflows.AccessResult) {
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
		case workflows.UserStatusActive:
			statusStr = ui.Success.Sprint("✓") + " active"
		case workflows.UserStatusPending:
			statusStr = ui.Warning.Sprint("⚠") + " pending"
		case workflows.UserStatusOrphan:
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
