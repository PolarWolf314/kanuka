package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/workflows"

	"github.com/spf13/cobra"
)

var (
	doctorJSONOutput bool
	// doctorExitFunc is the function called to exit with a specific code.
	// Can be overridden for testing.
	doctorExitFunc = os.Exit
)

func init() {
	doctorCmd.Flags().BoolVar(&doctorJSONOutput, "json", false, "output in JSON format")
}

func resetDoctorCommandState() {
	doctorJSONOutput = false
	doctorExitFunc = os.Exit
}

// SetDoctorExitFunc sets the exit function for testing purposes.
func SetDoctorExitFunc(f func(int)) {
	doctorExitFunc = f
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run health checks on the Kanuka project",
	Long: `Runs a series of health checks on the Kanuka project and reports issues.

The doctor command checks:
  - Project configuration validity
  - User configuration validity
  - Private key existence and permissions
  - Public key and encrypted symmetric key consistency
  - Gitignore configuration for .env files
  - Unencrypted .env files

Exit codes:
  0 - All checks passed
  1 - Warnings found (non-critical issues)
  2 - Errors found (critical issues)

Use --json for machine-readable output.`,
	RunE: runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	Logger.Infof("Starting doctor command")

	spinner, cleanup := startSpinner("Running health checks...", verbose)
	defer cleanup()

	result, err := workflows.Doctor(context.Background(), workflows.DoctorOptions{})
	if err != nil {
		spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to run health checks: " + err.Error()
		return err
	}

	for _, check := range result.Checks {
		Logger.Debugf("Check %s: status=%s, message=%s", check.Name, check.Status.String(), check.Message)
	}

	// Output results.
	if doctorJSONOutput {
		spinner.FinalMSG = ""
		if err := outputDoctorJSON(result); err != nil {
			return err
		}
	} else {
		spinner.FinalMSG = ""
		printDoctorResults(result)
		if result.Summary.Errors > 0 {
			spinner.FinalMSG = ui.Error.Sprint("✗") + " Health checks completed with errors"
		} else if result.Summary.Warnings > 0 {
			spinner.FinalMSG = ui.Warning.Sprint("⚠") + " Health checks completed with warnings"
		} else {
			spinner.FinalMSG = ui.Success.Sprint("✓") + " Health checks completed"
		}
	}

	// Set exit code based on results.
	if result.Summary.Errors > 0 {
		doctorExitFunc(2)
	}
	if result.Summary.Warnings > 0 {
		doctorExitFunc(1)
	}
	return nil
}

// outputDoctorJSON outputs the result as JSON.
func outputDoctorJSON(result *workflows.DoctorResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// printDoctorResults prints the doctor results in a human-readable format.
func printDoctorResults(result *workflows.DoctorResult) {
	fmt.Println("Running health checks...")
	fmt.Println()

	// Print each check result.
	for _, check := range result.Checks {
		var statusIcon string
		switch check.Status {
		case workflows.CheckPass:
			statusIcon = ui.Success.Sprint("✓")
		case workflows.CheckWarning:
			statusIcon = ui.Warning.Sprint("⚠")
		case workflows.CheckError:
			statusIcon = ui.Error.Sprint("✗")
		}
		fmt.Printf("%s %s\n", statusIcon, check.Message)
	}

	// Print summary.
	fmt.Println()
	fmt.Printf("Summary: %d passed", result.Summary.Passed)
	if result.Summary.Warnings > 0 {
		fmt.Printf(", %s", ui.Warning.Sprint(fmt.Sprintf("%d warning(s)", result.Summary.Warnings)))
	}
	if result.Summary.Errors > 0 {
		fmt.Printf(", %s", ui.Error.Sprint(fmt.Sprintf("%d error(s)", result.Summary.Errors)))
	}
	fmt.Println()

	// Print suggestions if any.
	if len(result.Suggestions) > 0 {
		fmt.Println()
		fmt.Println("Suggestions:")
		for _, suggestion := range result.Suggestions {
			fmt.Printf("  %s %s\n", ui.Info.Sprint("→"), suggestion)
		}
	}
}
