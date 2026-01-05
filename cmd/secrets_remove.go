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

		projectPath := configs.ProjectKanukaSettings.ProjectPath
		if projectPath == "" {
			finalMessage := color.RedString("✗") + " Kānuka has not been initialized\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " first\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

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

		username, filesToRemove, err := getFilesToRemove(spinner)
		if err != nil {
			return err
		}

		return removeFiles(spinner, username, filesToRemove)
	},
}

type fileToRemove struct {
	Path string
	Name string
}

func getFilesToRemove(spinner *spinner.Spinner) (string, []fileToRemove, error) {
	if removeUsername != "" {
		return getFilesByUsername(spinner)
	}
	return getFilesByPath(spinner)
}

func getFilesByUsername(spinner *spinner.Spinner) (string, []fileToRemove, error) {
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	Logger.Debugf("Project public key path: %s, Project secrets path: %s", projectPublicKeyPath, projectSecretsPath)

	publicKeyPath := filepath.Join(projectPublicKeyPath, removeUsername+".pub")
	kanukaKeyPath := filepath.Join(projectSecretsPath, removeUsername+".kanuka")

	Logger.Debugf("Checking for user files: %s, %s", publicKeyPath, kanukaKeyPath)

	publicKeyExists := false
	kanukaKeyExists := false

	if _, err := os.Stat(publicKeyPath); err == nil {
		publicKeyExists = true
	} else if !os.IsNotExist(err) {
		Logger.Errorf("Failed to check public key file %s: %v", publicKeyPath, err)
		finalMessage := color.RedString("✗") + " Failed to check user's public key file\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	if _, err := os.Stat(kanukaKeyPath); err == nil {
		kanukaKeyExists = true
	} else if !os.IsNotExist(err) {
		Logger.Errorf("Failed to check kanuka key file %s: %v", kanukaKeyPath, err)
		finalMessage := color.RedString("✗") + " Failed to check user's kanuka key file\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	if !publicKeyExists && !kanukaKeyExists {
		Logger.Infof("User %s does not exist in the project", removeUsername)
		finalMessage := color.RedString("✗") + " User " + color.YellowString(removeUsername) + " does not exist in this project\n" +
			color.CyanString("→") + " No files found for this user\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	var files []fileToRemove
	if publicKeyExists {
		files = append(files, fileToRemove{Path: publicKeyPath, Name: removeUsername + ".pub"})
	}
	if kanukaKeyExists {
		files = append(files, fileToRemove{Path: kanukaKeyPath, Name: removeUsername + ".kanuka"})
	}

	return removeUsername, files, nil
}

func getFilesByPath(spinner *spinner.Spinner) (string, []fileToRemove, error) {
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	Logger.Debugf("Project secrets path: %s, Project public key path: %s", projectSecretsPath, projectPublicKeyPath)

	absFilePath, err := filepath.Abs(removeFilePath)
	if err != nil {
		finalMessage := color.RedString("✗") + " Failed to resolve file path: " + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	Logger.Debugf("Absolute file path: %s", absFilePath)

	fileInfo, err := os.Stat(absFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			Logger.Infof("File does not exist: %s", absFilePath)
			finalMessage := color.RedString("✗") + " File " + color.YellowString(absFilePath) + " does not exist\n"
			spinner.FinalMSG = finalMessage
			return "", nil, nil
		}
		Logger.Errorf("Failed to check file %s: %v", absFilePath, err)
		finalMessage := color.RedString("✗") + " Failed to check file\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	if fileInfo.IsDir() {
		finalMessage := color.RedString("✗") + " Path " + color.YellowString(absFilePath) + " is a directory, not a file\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	absProjectSecretsPath, err := filepath.Abs(projectSecretsPath)
	if err != nil {
		finalMessage := color.RedString("✗") + " Failed to resolve project secrets path: " + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	if filepath.Dir(absFilePath) != absProjectSecretsPath {
		finalMessage := color.RedString("✗") + " File " + color.YellowString(absFilePath) + " is not in the project secrets directory\n" +
			color.CyanString("→") + " Expected directory: " + color.YellowString(absProjectSecretsPath) + "\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	if filepath.Ext(absFilePath) != ".kanuka" {
		finalMessage := color.RedString("✗") + " File " + color.YellowString(absFilePath) + " is not a .kanuka file\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	baseName := filepath.Base(absFilePath)
	username := baseName[:len(baseName)-len(".kanuka")]

	Logger.Debugf("Extracted username from file: %s", username)

	var files []fileToRemove
	files = append(files, fileToRemove{Path: absFilePath, Name: baseName})

	publicKeyPath := filepath.Join(projectPublicKeyPath, username+".pub")
	if _, err := os.Stat(publicKeyPath); err == nil {
		files = append(files, fileToRemove{Path: publicKeyPath, Name: username + ".pub"})
	} else if !os.IsNotExist(err) {
		Logger.Errorf("Failed to check public key file %s: %v", publicKeyPath, err)
		finalMessage := color.RedString("✗") + " Failed to check public key file\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	return username, files, nil
}

func removeFiles(spinner *spinner.Spinner, username string, filesToRemove []fileToRemove) error {
	if len(filesToRemove) == 0 {
		return nil
	}

	currentUsername := configs.UserKanukaSettings.Username
	currentUserKeysPath := configs.UserKanukaSettings.UserKeysPath
	projectName := configs.ProjectKanukaSettings.ProjectName

	Logger.Debugf("Current user: %s, Project: %s", currentUsername, projectName)

	var removedFiles []string
	var errors []error

	for _, file := range filesToRemove {
		Logger.Debugf("Removing file: %s", file.Path)
		if err := os.Remove(file.Path); err != nil {
			Logger.Errorf("Failed to remove file %s: %v", file.Path, err)
			errors = append(errors, err)
		} else {
			removedFiles = append(removedFiles, file.Name)
			Logger.Infof("Successfully removed file: %s", file.Name)
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

	allUsers, err := secrets.GetAllUsersInProject()
	if err != nil {
		Logger.Errorf("Failed to get list of users: %v", err)
		finalMessage := color.RedString("✗") + " Files were removed but failed to rotate key: " + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if len(allUsers) > 0 {
		spinner.Suffix = " Rotating symmetric key for remaining users..."
		Logger.Infof("Rotating symmetric key for %d remaining users", len(allUsers))

		privateKeyPath := filepath.Join(currentUserKeysPath, projectName)
		privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			Logger.Errorf("Failed to load private key: %v", err)
			finalMessage := color.RedString("✗") + " Files were removed but failed to rotate key: " + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		if err := secrets.RotateSymmetricKey(currentUsername, privateKey, verbose); err != nil {
			Logger.Errorf("Failed to rotate symmetric key: %v", err)
			finalMessage := color.RedString("✗") + " Files were removed but failed to rotate key: " + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Infof("Symmetric key rotated successfully")
	}

	Logger.Infof("Files removal completed successfully for: %s", username)
	finalMessage := color.GreenString("✓") + " Files for " + color.YellowString(username) + " have been removed successfully!\n" +
		color.CyanString("→") + " Removed: "
	for i, file := range removedFiles {
		if i > 0 {
			finalMessage += ", "
		}
		finalMessage += color.YellowString(file)
	}
	finalMessage += "\n"
	if len(allUsers) > 0 {
		finalMessage += color.CyanString("→") + " Symmetric key has been rotated for remaining users\n"
	}
	spinner.FinalMSG = finalMessage
	return nil
}
