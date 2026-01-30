package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/PolarWolf314/kanuka/internal/workflows"

	"github.com/spf13/cobra"
)

var (
	initYes         bool
	initProjectName string
)

func init() {
	initCmd.Flags().BoolVarP(&initYes, "yes", "y", false, "non-interactive mode (fail if user config is incomplete)")
	initCmd.Flags().StringVarP(&initProjectName, "name", "n", "", "project name (defaults to directory name)")
}

// resetInitCommandState resets the init command's global state for testing.
func resetInitCommandState() {
	initYes = false
	initProjectName = ""
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes the secrets store",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	Logger.Infof("Starting init command")
	spinner, cleanup := startSpinner("Initializing Kānuka...", verbose)
	defer cleanup()

	Logger.Debugf("Checking if project kanuka settings already exist")
	kanukaExists, err := secrets.DoesProjectKanukaSettingsExist()
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to check if project kanuka settings exists: %v", err)
	}
	if kanukaExists {
		spinner.FinalMSG = formatInitError(kerrors.ErrProjectAlreadyInitialized)
		return nil
	}

	Logger.Debugf("Ensuring user settings")
	if err := secrets.EnsureUserSettings(); err != nil {
		return Logger.ErrorfAndReturn("Failed ensuring user settings: %v", err)
	}
	Logger.Infof("User settings ensured successfully")

	Logger.Debugf("Checking if user config is complete")
	isComplete, err := IsUserConfigComplete()
	if err != nil {
		return Logger.ErrorfAndReturn("Failed to check user config: %v", err)
	}

	if !isComplete {
		Logger.Infof("User config is incomplete, need to run setup")

		if initYes {
			spinner.FinalMSG = ui.Error.Sprint("✗") + " User configuration is incomplete" +
				"\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka config init") + " first to set up your identity"
			return fmt.Errorf("user configuration required: run 'kanuka config init' first")
		}

		spinner.Stop()
		fmt.Println(ui.Warning.Sprint("⚠") + " User configuration not found.\n")
		fmt.Println("Running initial setup...")
		fmt.Println()

		setupPerformed, setupErr := RunConfigInit(verbose, debug)
		if setupErr != nil {
			return Logger.ErrorfAndReturn("Failed to set up user config: %v", setupErr)
		}

		if !setupPerformed {
			Logger.Debugf("Config init reported no setup needed")
		}

		fmt.Println("Initializing project...")
		spinner.Restart()
	}

	projectName, err := resolveProjectName(spinner)
	if err != nil {
		return err
	}

	opts := workflows.InitOptions{
		ProjectName: projectName,
		Verbose:     verbose,
	}

	result, err := workflows.Init(cmd.Context(), opts)
	if err != nil {
		Logger.Errorf("Init workflow failed: %v", err)
		spinner.FinalMSG = formatInitError(err)
		spinner.Stop()
		// Return error for unexpected failures so the command exits with non-zero status.
		// Only ErrProjectAlreadyInitialized is an "expected" error that shouldn't fail.
		if !errors.Is(err, kerrors.ErrProjectAlreadyInitialized) {
			return err
		}
		return nil
	}

	Logger.Infof("Init command completed successfully")

	spinner.Stop()
	Logger.WarnfUser("Remember to never commit .env files to version control - only commit .kanuka files")
	spinner.Restart()

	spinner.FinalMSG = ui.Success.Sprint("✓") + " Kānuka initialized successfully!" +
		"\n\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets encrypt") + " to encrypt your existing .env files" +
		"\n\n" + ui.Info.Sprint("Tip:") + " Working in a monorepo? You have two options:" +
		"\n  1. Keep this single .kanuka at the root and use selective encryption:" +
		"\n     " + ui.Code.Sprint("kanuka secrets encrypt services/api/.env") +
		"\n  2. Initialize separate .kanuka stores in each service:" +
		"\n     " + ui.Code.Sprint("cd services/api && kanuka secrets init")

	_ = result // result contains useful info for future enhancements
	return nil
}

// resolveProjectName determines the project name from flag, prompt, or default.
func resolveProjectName(spinner interface {
	Stop()
	Restart()
}) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", Logger.ErrorfAndReturn("Failed to get working directory: %v", err)
	}

	defaultProjectName := filepath.Base(wd)

	if initProjectName != "" {
		projectName := strings.TrimSpace(initProjectName)
		Logger.Debugf("Using project name from flag: %s", projectName)
		return projectName, nil
	}

	if initYes {
		Logger.Debugf("Using default project name (non-interactive): %s", defaultProjectName)
		return defaultProjectName, nil
	}

	spinner.Stop()
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Project name [%s]: ", defaultProjectName)
	input, readErr := reader.ReadString('\n')
	if readErr != nil {
		return "", Logger.ErrorfAndReturn("Failed to read project name: %v", readErr)
	}
	projectName := strings.TrimSpace(input)
	if projectName == "" {
		projectName = defaultProjectName
	}
	spinner.Restart()

	if projectName == "" {
		return "", Logger.ErrorfAndReturn("Project name cannot be empty")
	}

	Logger.Infof("Using project name: %s", projectName)
	return projectName, nil
}

func formatInitError(err error) string {
	switch {
	case errors.Is(err, kerrors.ErrProjectAlreadyInitialized):
		return ui.Error.Sprint("✗") + " Kānuka has already been initialized" +
			"\n" + ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets create") + " instead"

	default:
		return ui.Error.Sprint("✗") + " " + err.Error()
	}
}
