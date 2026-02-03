package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/utils"
	"github.com/PolarWolf314/kanuka/internal/workflows"

	"github.com/spf13/cobra"
)

func init() {
	SecretsCmd.AddCommand(ciInitCmd)
}

// resetCIInitCommandState resets the ci-init command's global state for testing.
func resetCIInitCommandState() {
	// No flags to reset currently.
}

var ciInitCmd = &cobra.Command{
	Use:   "ci-init",
	Short: "Set up GitHub Actions CI integration",
	Long: `Set up GitHub Actions CI integration for this project.

This command:
1. Generates a dedicated CI keypair (private key never saved to disk)
2. Registers the CI user with the project
3. Creates a GitHub Actions workflow template
4. Securely displays the private key for you to add to GitHub Secrets

The private key is displayed only once and must be copied to your
GitHub repository's secrets as KANUKA_PRIVATE_KEY.

This command requires an interactive terminal as the private key
is displayed directly to the TTY for security.

Example:
  kanuka secrets ci-init`,
	RunE: runCIInit,
}

func runCIInit(cmd *cobra.Command, args []string) error {
	Logger.Infof("Starting ci-init command")
	spinner, cleanup := startSpinner("Setting up CI integration...", verbose)
	defer cleanup()

	ctx := context.Background()
	opts := workflows.CIInitOptions{
		Verbose: verbose,
		Debug:   debug,
	}

	result, err := workflows.CIInit(ctx, opts)
	if err != nil {
		spinner.FinalMSG = formatCIInitError(err)

		// Return nil for expected errors (they've been displayed via defer cleanup).
		if errors.Is(err, kerrors.ErrProjectNotInitialized) ||
			errors.Is(err, kerrors.ErrCIAlreadyConfigured) ||
			errors.Is(err, kerrors.ErrTTYRequired) ||
			errors.Is(err, kerrors.ErrNoAccess) {
			return nil
		}
		return err
	}

	// Stop spinner before TTY output. Clear FinalMSG since we handle output manually.
	spinner.FinalMSG = ""
	cleanup()

	// Display the private key securely.
	if err := displayPrivateKeySecurely(result); err != nil {
		fmt.Println(ui.Error.Sprint("✗") + " Failed to display private key: " + err.Error())
		return err
	}

	// Show success message and next steps.
	printCIInitSuccess(result)
	return nil
}

func formatCIInitError(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectNotInitialized):
		return ui.Error.Sprint("✗") + " Kanuka has not been initialized\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first"

	case errors.Is(err, kerrors.ErrCIAlreadyConfigured):
		return ui.Error.Sprint("✗") + " CI integration is already configured\n" +
			ui.Info.Sprint("→") + " To reconfigure, first run " + ui.Code.Sprint("kanuka secrets revoke --user "+workflows.CIUserEmail)

	case errors.Is(err, kerrors.ErrTTYRequired):
		return ui.Error.Sprint("✗") + " This command requires an interactive terminal\n" +
			ui.Info.Sprint("→") + " Run this command directly in your terminal (not piped or in a script)"

	case errors.Is(err, kerrors.ErrNoAccess):
		return ui.Error.Sprint("✗") + " You don't have access to this project\n" +
			ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets create") + " to generate your keys"

	default:
		return ui.Error.Sprint("✗") + " CI setup failed: " + err.Error() + "\n"
	}
}

func displayPrivateKeySecurely(result *workflows.CIInitResult) error {
	// Display brief instructions.
	preMessage := "\n" +
		ui.Warning.Sprint("IMPORTANT:") + " Copy the private key below and save it to GitHub Secrets.\n" +
		"This key will " + ui.Error.Sprint("NOT") + " be shown again.\n\n" +
		strings.Repeat("=", 70) + "\n\n"

	if err := utils.WriteToTTY(preMessage); err != nil {
		return fmt.Errorf("writing instructions: %w", err)
	}

	// Display the private key.
	if err := utils.WriteToTTY(string(result.PrivateKeyPEM)); err != nil {
		return fmt.Errorf("writing private key: %w", err)
	}

	// Display wait prompt.
	postMessage := "\n" + strings.Repeat("=", 70) + "\n\n" +
		"Press " + ui.Highlight.Sprint("Enter") + " when you have copied the key..."

	if err := utils.WriteToTTY(postMessage); err != nil {
		return fmt.Errorf("writing prompt: %w", err)
	}

	// Wait for Enter.
	if err := utils.WaitForEnterFromTTY(); err != nil {
		return fmt.Errorf("waiting for input: %w", err)
	}

	// Clear screen.
	if err := utils.ClearScreen(); err != nil {
		// Non-fatal - just continue.
		Logger.Debugf("Failed to clear screen: %v", err)
	}

	return nil
}

func printCIInitSuccess(result *workflows.CIInitResult) {
	fmt.Println()
	fmt.Println(ui.Success.Sprint("✓") + " CI user registered successfully!")

	if result.WorkflowCreated {
		fmt.Println(ui.Success.Sprint("✓") + " Workflow template created at " + ui.Path.Sprint(result.WorkflowPath))
	} else {
		fmt.Println(ui.Warning.Sprint("⚠") + " Workflow file already exists at " + ui.Path.Sprint(result.WorkflowPath) + " (skipped)")
	}

	fmt.Println()
	fmt.Println(ui.Highlight.Sprint("Next steps:"))
	fmt.Println()

	secretsURL := result.GitHubRepoURL + "/settings/secrets/actions"

	fmt.Println("1. Go to your GitHub repository secrets:")
	fmt.Println("   " + ui.Code.Sprint(secretsURL))
	fmt.Println()
	fmt.Println("2. Click " + ui.Highlight.Sprint("\"New repository secret\""))
	fmt.Println()
	fmt.Println("3. Name: " + ui.Code.Sprint("KANUKA_PRIVATE_KEY"))
	fmt.Println("   Value: (paste the private key you just copied)")
	fmt.Println()
	fmt.Println("4. Click " + ui.Highlight.Sprint("\"Add secret\""))
	fmt.Println()
	fmt.Println("5. Commit and push the changes:")
	fmt.Println("   " + ui.Code.Sprint("git add .github/workflows/kanuka-decrypt.yml .kanuka/"))
	fmt.Println("   " + ui.Code.Sprint("git commit -m \"Add Kanuka CI integration\""))
	fmt.Println("   " + ui.Code.Sprint("git push"))
	fmt.Println()
	fmt.Println(ui.Info.Sprint("→") + " The next pull request will automatically decrypt secrets!")
}
