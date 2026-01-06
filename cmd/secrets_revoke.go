package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"
	"github.com/PolarWolf314/kanuka/internal/utils"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	revokeUserEmail string
	revokeFilePath  string
	revokeDevice    string
	revokeYes       bool
)

func resetRevokeCommandState() {
	revokeUserEmail = ""
	revokeFilePath = ""
	revokeDevice = ""
	revokeYes = false
}

func init() {
	revokeCmd.Flags().StringVarP(&revokeUserEmail, "user", "u", "", "user email to revoke access from the secret store")
	revokeCmd.Flags().StringVarP(&revokeFilePath, "file", "f", "", "path to a .kanuka file to revoke along with its corresponding public key")
	revokeCmd.Flags().StringVar(&revokeDevice, "device", "", "specific device name to revoke (requires --user)")
	revokeCmd.Flags().BoolVarP(&revokeYes, "yes", "y", false, "skip confirmation prompts (for automation)")
}

var revokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revokes access to the secret store",
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting revoke command")
		spinner, cleanup := startSpinner("Revoking user access...", verbose)
		defer cleanup()

		Logger.Debugf("Checking command flags: revokeUserEmail=%s, revokeFilePath=%s, revokeDevice=%s, revokeYes=%t",
			revokeUserEmail, revokeFilePath, revokeDevice, revokeYes)

		if revokeUserEmail == "" && revokeFilePath == "" {
			finalMessage := color.RedString("✗") + " Either " + color.YellowString("--user") + " or " + color.YellowString("--file") + " flag is required.\n" +
				"Run " + color.YellowString("kanuka secrets revoke --help") + " to see the available commands.\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		if revokeUserEmail != "" && revokeFilePath != "" {
			finalMessage := color.RedString("✗") + " Cannot specify both " + color.YellowString("--user") + " and " + color.YellowString("--file") + " flags.\n" +
				"Run " + color.YellowString("kanuka secrets revoke --help") + " to see the available commands.\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// --device requires --user
		if revokeDevice != "" && revokeUserEmail == "" {
			finalMessage := color.RedString("✗") + " The " + color.YellowString("--device") + " flag requires the " + color.YellowString("--user") + " flag.\n" +
				"Run " + color.YellowString("kanuka secrets revoke --help") + " to see the available commands.\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Validate email format if provided
		if revokeUserEmail != "" && !utils.IsValidEmail(revokeUserEmail) {
			finalMessage := color.RedString("✗") + " Invalid email format: " + color.YellowString(revokeUserEmail) + "\n" +
				color.CyanString("→") + " Please provide a valid email address"
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

		displayName, filesToRevoke, err := getFilesToRevoke(spinner)
		if err != nil {
			return err
		}

		return revokeFiles(spinner, displayName, filesToRevoke)
	},
}

type fileToRevoke struct {
	Path string
	Name string
}

func getFilesToRevoke(spinner *spinner.Spinner) (string, []fileToRevoke, error) {
	if revokeUserEmail != "" {
		return getFilesByUserEmail(spinner)
	}
	return getFilesByPath(spinner)
}

func getFilesByUserEmail(spinner *spinner.Spinner) (string, []fileToRevoke, error) {
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	Logger.Debugf("Project public key path: %s, Project secrets path: %s", projectPublicKeyPath, projectSecretsPath)

	// Load project config to look up user by email
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return "", nil, Logger.ErrorfAndReturn("Failed to load project config: %v", err)
	}

	// Get all devices for this email
	devices := projectConfig.GetDevicesByEmail(revokeUserEmail)
	if len(devices) == 0 {
		finalMessage := color.RedString("✗") + " User " + color.YellowString(revokeUserEmail) + " not found in this project\n" +
			color.CyanString("→") + " No devices found for this user\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	// If --device is specified, find that specific device
	if revokeDevice != "" {
		targetUserUUID, found := projectConfig.GetUserUUIDByEmailAndDevice(revokeUserEmail, revokeDevice)
		if !found {
			finalMessage := color.RedString("✗") + " Device " + color.YellowString(revokeDevice) + " not found for user " + color.YellowString(revokeUserEmail) + "\n" +
				color.CyanString("→") + " Available devices:\n"
			for _, device := range devices {
				finalMessage += "    - " + color.YellowString(device.Name) + "\n"
			}
			spinner.FinalMSG = finalMessage
			return "", nil, nil
		}

		// Return files for this specific device
		return getFilesForUUID(spinner, targetUserUUID, revokeUserEmail+" ("+revokeDevice+")")
	}

	// No --device specified, handle all devices for this user
	// If multiple devices and no --yes, prompt for confirmation
	if len(devices) > 1 && !revokeYes {
		spinner.Stop()

		fmt.Printf("\n%s Warning: %s has %d devices:\n", color.YellowString("⚠"), revokeUserEmail, len(devices))
		for _, device := range devices {
			fmt.Printf("  - %s (created: %s)\n", device.Name, device.CreatedAt.Format("Jan 2, 2006"))
		}
		fmt.Println("\nThis will revoke ALL devices for this user.")

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Proceed? [y/N]: ")
		response, err := reader.ReadString('\n')
		if err != nil {
			return "", nil, Logger.ErrorfAndReturn("Failed to read response: %v", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			finalMessage := color.YellowString("⚠") + " Revocation cancelled\n"
			spinner.FinalMSG = finalMessage
			spinner.Restart()
			return "", nil, nil
		}

		spinner.Restart()
	}

	// Collect all files for all devices
	var allFiles []fileToRevoke
	for userUUID := range devices {
		publicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
		kanukaKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")

		if _, err := os.Stat(publicKeyPath); err == nil {
			allFiles = append(allFiles, fileToRevoke{Path: publicKeyPath, Name: userUUID + ".pub"})
		}
		if _, err := os.Stat(kanukaKeyPath); err == nil {
			allFiles = append(allFiles, fileToRevoke{Path: kanukaKeyPath, Name: userUUID + ".kanuka"})
		}
	}

	if len(allFiles) == 0 {
		finalMessage := color.RedString("✗") + " No files found for user " + color.YellowString(revokeUserEmail) + "\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	return revokeUserEmail, allFiles, nil
}

func getFilesForUUID(spinner *spinner.Spinner, userUUID, displayName string) (string, []fileToRevoke, error) {
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath

	publicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
	kanukaKeyPath := filepath.Join(projectSecretsPath, userUUID+".kanuka")

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
		Logger.Infof("User %s does not exist in the project", displayName)
		finalMessage := color.RedString("✗") + " User " + color.YellowString(displayName) + " does not exist in this project\n" +
			color.CyanString("→") + " No files found for this user\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	var files []fileToRevoke
	if publicKeyExists {
		files = append(files, fileToRevoke{Path: publicKeyPath, Name: userUUID + ".pub"})
	}
	if kanukaKeyExists {
		files = append(files, fileToRevoke{Path: kanukaKeyPath, Name: userUUID + ".kanuka"})
	}

	return displayName, files, nil
}

func getFilesByPath(spinner *spinner.Spinner) (string, []fileToRevoke, error) {
	projectSecretsPath := configs.ProjectKanukaSettings.ProjectSecretsPath
	projectPublicKeyPath := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	Logger.Debugf("Project secrets path: %s, Project public key path: %s", projectSecretsPath, projectPublicKeyPath)

	absFilePath, err := filepath.Abs(revokeFilePath)
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
	userUUID := baseName[:len(baseName)-len(".kanuka")]

	Logger.Debugf("Extracted user UUID from file: %s", userUUID)

	// Try to find email for display purposes
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		Logger.Warnf("Could not load project config for display name lookup: %v", err)
	}
	displayName := userUUID
	if projectConfig != nil {
		if email, exists := projectConfig.Users[userUUID]; exists && email != "" {
			displayName = email
		}
	}

	var files []fileToRevoke
	files = append(files, fileToRevoke{Path: absFilePath, Name: baseName})

	publicKeyPath := filepath.Join(projectPublicKeyPath, userUUID+".pub")
	if _, err := os.Stat(publicKeyPath); err == nil {
		files = append(files, fileToRevoke{Path: publicKeyPath, Name: userUUID + ".pub"})
	} else if !os.IsNotExist(err) {
		Logger.Errorf("Failed to check public key file %s: %v", publicKeyPath, err)
		finalMessage := color.RedString("✗") + " Failed to check public key file\n" +
			color.RedString("Error: ") + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return "", nil, nil
	}

	return displayName, files, nil
}

func revokeFiles(spinner *spinner.Spinner, displayName string, filesToRevoke []fileToRevoke) error {
	if len(filesToRevoke) == 0 {
		return nil
	}

	// Load user config for current user UUID
	userConfig, err := configs.EnsureUserConfig()
	if err != nil {
		return Logger.ErrorfAndReturn("failed to load user config: %v", err)
	}
	currentUserUUID := userConfig.User.UUID

	// Load project config for project UUID
	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		return Logger.ErrorfAndReturn("failed to load project config: %v", err)
	}
	projectUUID := projectConfig.Project.UUID

	currentUserKeysPath := configs.UserKanukaSettings.UserKeysPath

	Logger.Debugf("Current user UUID: %s, Project UUID: %s", currentUserUUID, projectUUID)

	var revokedFiles []string
	var errors []error

	for _, file := range filesToRevoke {
		Logger.Debugf("Revoking file: %s", file.Path)
		if err := os.Remove(file.Path); err != nil {
			Logger.Errorf("Failed to revoke file %s: %v", file.Path, err)
			errors = append(errors, err)
		} else {
			revokedFiles = append(revokedFiles, file.Name)
			Logger.Infof("Successfully revoked file: %s", file.Name)
		}
	}

	if len(errors) > 0 {
		finalMessage := color.RedString("✗") + " Failed to completely revoke files for " + color.YellowString(displayName) + "\n"
		for _, err := range errors {
			finalMessage += color.RedString("Error: ") + err.Error() + "\n"
		}
		if len(revokedFiles) > 0 {
			finalMessage += color.YellowString("Warning: ") + "Some files were revoked successfully\n"
		}
		spinner.FinalMSG = finalMessage
		return nil
	}

	allUsers, err := secrets.GetAllUsersInProject()
	if err != nil {
		Logger.Errorf("Failed to get list of users: %v", err)
		finalMessage := color.RedString("✗") + " Files were revoked but failed to rotate key: " + err.Error() + "\n"
		spinner.FinalMSG = finalMessage
		return nil
	}

	if len(allUsers) > 0 {
		spinner.Suffix = " Rotating symmetric key for remaining users..."
		Logger.Infof("Rotating symmetric key for %d remaining users", len(allUsers))

		privateKeyPath := filepath.Join(currentUserKeysPath, projectUUID)
		privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			Logger.Errorf("Failed to load private key: %v", err)
			finalMessage := color.RedString("✗") + " Files were revoked but failed to rotate key: " + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		if err := secrets.RotateSymmetricKey(currentUserUUID, privateKey, verbose); err != nil {
			Logger.Errorf("Failed to rotate symmetric key: %v", err)
			finalMessage := color.RedString("✗") + " Files were revoked but failed to rotate key: " + err.Error() + "\n"
			spinner.FinalMSG = finalMessage
			return nil
		}

		Logger.Infof("Symmetric key rotated successfully")
	}

	Logger.Infof("Files revocation completed successfully for: %s", displayName)
	finalMessage := color.GreenString("✓") + " Access for " + color.YellowString(displayName) + " has been revoked successfully!\n" +
		color.CyanString("→") + " Revoked: "
	for i, file := range revokedFiles {
		if i > 0 {
			finalMessage += ", "
		}
		finalMessage += color.YellowString(file)
	}
	finalMessage += "\n"
	if len(allUsers) > 0 {
		finalMessage += color.CyanString("→") + " Symmetric key has been rotated for remaining users\n"
	}
	finalMessage += color.YellowString("⚠") + color.RedString(" Warning: ") + color.YellowString(displayName) + " may still have access to old secrets from their local git history.\n" +
		color.CyanString("→") + " If necessary, rotate your actual secret values after this revocation.\n"
	spinner.FinalMSG = finalMessage
	return nil
}
