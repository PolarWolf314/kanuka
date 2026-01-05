package cmd

import (
	"os"
	"path/filepath"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var removeUsername string
var removeFilePath string

// resetRemoveCommandState resets all remove command global variables to their default values for testing.
func resetRemoveCommandState() {
	removeUsername = ""
	removeFilePath = ""
}

func init() {
	removeCmd.Flags().StringVarP(&removeUsername, "user", "u", "", "username to remove access from the secret store")
	removeCmd.Flags().StringVarP(&removeFilePath, "file", "f", "", "path to a .kanuka file to remove along with its corresponding public key")
}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Removes access to the secret store",
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting remove command")
		spinner, cleanup := startSpinner("Removing user access...", verbose)
		defer cleanup()

		// Check for required flags
		Logger.Debugf("Checking command flags: removeUsername=%s, removeFilePath=%s", removeUsername, removeFilePath)
		if removeUsername == "" && removeFilePath == "" {
			finalMessage := color.RedString("✗") + " Either " + color.YellowString("--user") + " or " + color.YellowString("--file") + " flag is required.\n" +
				"Run " + color.YellowString("kanuka secrets remove --help") + " to see the available commands.\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		if removeUsername != "" && removeFilePath != "" {
			finalMessage := color.RedString("✗") + " Cannot specify both " + color.YellowString("--user") + " and " + color.YellowString("--file") + " flags.\n" +
				"Run " + color.YellowString("kanuka secrets remove --help") + " to see the available commands.\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}

		// Check if project is initialized
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		if projectPath == "" {
			finalMessage := color.RedString("✗") + " Kānuka has not been initialized\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " first\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Check if project exists
		exists, err := secrets.DoesProjectKanukaSettingsExist()
		if err != nil {
			return Logger.ErrorfAndReturn("failed to check if project exists: %v", err)
		}
		if !exists {
			finalMessage := color.RedString("✗") + " Kānuka project not found\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " first\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		if removeFilePath != "" {
			return handleFileRemoval(spinner)
		}
		return handleUserRemoval(spinner)
	},
}

func handleUserRemoval(spinner *spinner.Spinner) error {
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	Logger.Debugf("Project public key path: %s, Project secrets path: %s", projectPublicKeyPath, projectSecretsPath)

	// Define file paths for the user
	publicKeyPath := filepath.Join(projectPublicKeyPath, removeUsername+".pub")
	kanukaKeyPath := filepath.Join(projectSecretsPath, removeUsername+".kanuka")

	Logger.Debugf("Checking for user files: %s, %s", publicKeyPath, kanukaKeyPath)

	// Check if user exists (has both files)
	publicKeyExists := false
	kanukaKeyExists := false

	if _, err := os.Stat(publicKeyPath); err == nil {
		publicKeyExists = true
	} else if !os.IsNotExist(err) {
		Logger.Errorf("Failed to check public key file %s: %v", publicKeyPath, err)
		finalMessage := color.RedString("✗") + " Failed to check user's public key file\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if _, err := os.Stat(kanukaKeyPath); err == nil {
		kanukaKeyExists = true
	} else if !os.IsNotExist(err) {
		Logger.Errorf("Failed to check kanuka key file %s: %v", kanukaKeyPath, err)
		finalMessage := color.RedString("✗") + " Failed to check user's kanuka key file\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// If neither file exists, user doesn't exist
	if !publicKeyExists && !kanukaKeyExists {
		Logger.Infof("User %s does not exist in the project", removeUsername)
		finalMessage := color.RedString("✗") + " User " + color.YellowString(removeUsername) + " does not exist in this project\n" +
			color.CyanString("→") + " No files found for this user\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	// Remove files that exist
	var removedFiles []string
	var errors []error

	if publicKeyExists {
		Logger.Debugf("Removing public key file: %s", publicKeyPath)
		if err := os.Remove(publicKeyPath); err != nil {
			Logger.Errorf("Failed to remove public key file %s: %v", publicKeyPath, err)
			errors = append(errors, err)
		} else {
			removedFiles = append(removedFiles, removeUsername+".pub")
			Logger.Infof("Successfully removed public key file")
		}
	}

	if kanukaKeyExists {
		Logger.Debugf("Removing kanuka key file: %s", kanukaKeyPath)
		if err := os.Remove(kanukaKeyPath); err != nil {
			Logger.Errorf("Failed to remove kanuka key file %s: %v", kanukaKeyPath, err)
			errors = append(errors, err)
		} else {
			removedFiles = append(removedFiles, removeUsername+".kanuka")
			Logger.Infof("Successfully removed kanuka key file")
		}
	}

	// Report results
	if len(errors) > 0 {
		finalMessage := color.RedString("✗") + " Failed to completely remove user " + color.YellowString(removeUsername) + "\n"
		for _, err := range errors {
			finalMessage += color.RedString("Error: ") + err.Error() + "\n"
		}
		if len(removedFiles) > 0 {
			finalMessage += color.YellowString("Warning: ") + "Some files were removed successfully\n"
		}
		spinner.FinalMSG = finalMessage
		return nil
	}

	Logger.Infof("User removal completed successfully for: %s", removeUsername)
	finalMessage := color.GreenString("✓") + " User " + color.YellowString(removeUsername) + " has been removed successfully!\n" +
		color.CyanString("→") + " They no longer have access to decrypt the repository's secrets\n"
	spinner.FinalMSG = finalMessage
	return nil
}

func handleFileRemoval(spinner *spinner.Spinner) error {
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	Logger.Debugf("Project secrets path: %s, Project public key path: %s", projectSecretsPath, projectPublicKeyPath)

	absFilePath, err := filepath.Abs(removeFilePath)
	if err != nil {
		finalMessage := color.RedString("✗") + " Failed to resolve file path: " + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	Logger.Debugf("Absolute file path: %s", absFilePath)

	fileInfo, err := os.Stat(absFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			Logger.Infof("File does not exist: %s", absFilePath)
			finalMessage := color.RedString("✗") + " File " + color.YellowString(absFilePath) + " does not exist\n"
			spinner.FinalMSG = finalMessage
			return nil
		}
		Logger.Errorf("Failed to check file %s: %v", absFilePath, err)
		finalMessage := color.RedString("✗") + " Failed to check file\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if fileInfo.IsDir() {
		finalMessage := color.RedString("✗") + " Path " + color.YellowString(absFilePath) + " is a directory, not a file\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	absProjectSecretsPath, err := filepath.Abs(projectSecretsPath)
	if err != nil {
		finalMessage := color.RedString("✗") + " Failed to resolve project secrets path: " + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if filepath.Dir(absFilePath) != absProjectSecretsPath {
		finalMessage := color.RedString("✗") + " File " + color.YellowString(absFilePath) + " is not in the project secrets directory\n" +
			color.CyanString("→") + " Expected directory: " + color.YellowString(absProjectSecretsPath) + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if filepath.Ext(absFilePath) != ".kanuka" {
		finalMessage := color.RedString("✗") + " File " + color.YellowString(absFilePath) + " is not a .kanuka file\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	baseName := filepath.Base(absFilePath)
	username := baseName[:len(baseName)-len(".kanuka")]

	Logger.Debugf("Extracted username from file: %s", username)

	kanukaKeyPath := absFilePath
	publicKeyPath := filepath.Join(projectPublicKeyPath, username+".pub")

	Logger.Debugf("Checking for user files: %s, %s", publicKeyPath, kanukaKeyPath)

	publicKeyExists := false
	if _, err := os.Stat(publicKeyPath); err == nil {
		publicKeyExists = true
	} else if !os.IsNotExist(err) {
		Logger.Errorf("Failed to check public key file %s: %v", publicKeyPath, err)
		finalMessage := color.RedString("✗") + " Failed to check public key file\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	var removedFiles []string
	var errors []error

	Logger.Debugf("Removing kanuka key file: %s", kanukaKeyPath)
	if err := os.Remove(kanukaKeyPath); err != nil {
		Logger.Errorf("Failed to remove kanuka key file %s: %v", kanukaKeyPath, err)
		errors = append(errors, err)
	} else {
		removedFiles = append(removedFiles, baseName)
		Logger.Infof("Successfully removed kanuka key file")
	}

	if publicKeyExists {
		Logger.Debugf("Removing public key file: %s", publicKeyPath)
		if err := os.Remove(publicKeyPath); err != nil {
			Logger.Errorf("Failed to remove public key file %s: %v", publicKeyPath, err)
			errors = append(errors, err)
		} else {
			removedFiles = append(removedFiles, username+".pub")
			Logger.Infof("Successfully removed public key file")
		}
	}

	if len(errors) > 0 {
		finalMessage := color.RedString("✗") + " Failed to completely remove files for " + color.YellowString(username) + "\n"
		for _, err := range errors {
			finalMessage += color.RedString("Error: ") + err.Error() + "\n"
		}
		if len(removedFiles) > 0 {
			finalMessage += color.YellowString("Warning: ") + "Some files were removed successfully\n"
		}
		spinner.FinalMSG = finalMessage
		return nil
	}

	Logger.Infof("File removal completed successfully for: %s", username)
	finalMessage := color.GreenString("✓") + " Files for " + color.YellowString(username) + " have been removed successfully!\n" +
		color.CyanString("→") + " Removed: " + color.YellowString(removedFiles[0])
	if len(removedFiles) > 1 {
		finalMessage += ", " + color.YellowString(removedFiles[1])
	}
	finalMessage += "\n"
	spinner.FinalMSG = finalMessage
	return nil
}
