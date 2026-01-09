package cmd

import (
	"fmt"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/secrets"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var syncDryRun bool

func init() {
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "preview sync without making changes")
}

func resetSyncCommandState() {
	syncDryRun = false
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Re-encrypt all secrets with a new symmetric key",
	Long: `Re-encrypts all secret files with a newly generated symmetric key.

This command is useful for:
  - Periodic security key rotation
  - After adding new team members
  - If you suspect a key may have been compromised

All users with access will receive the new symmetric key, encrypted
with their public key. The old symmetric key will no longer work.

Use --dry-run to preview what would happen without making changes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting sync command")
		spinner, cleanup := startSpinner("Syncing secrets...", verbose)
		defer cleanup()

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project path: %s", projectPath)

		if projectPath == "" {
			finalMessage := color.RedString("✗") + " Kanuka has not been initialized\n" +
				color.CyanString("→") + " Run " + color.YellowString("kanuka secrets init") + " first"
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Load project config for project UUID.
		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			return Logger.ErrorfAndReturn("failed to load project config: %v", err)
		}
		projectUUID := projectConfig.Project.UUID
		Logger.Debugf("Project UUID: %s", projectUUID)

		// Load private key.
		privateKeyPath := configs.GetPrivateKeyPath(projectUUID)
		Logger.Debugf("Loading private key from: %s", privateKeyPath)
		privateKey, err := secrets.LoadPrivateKey(privateKeyPath)
		if err != nil {
			Logger.Errorf("Failed to load private key from %s: %v", privateKeyPath, err)
			finalMessage := color.RedString("✗") + " Failed to load your private key. Are you sure you have access?\n" +
				color.RedString("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}
		Logger.Infof("Private key loaded successfully")

		// Build sync options.
		opts := secrets.SyncOptions{
			DryRun:  syncDryRun,
			Verbose: verbose,
			Debug:   debug,
		}

		// Call sync function.
		result, err := secrets.SyncSecrets(privateKey, opts)
		if err != nil {
			Logger.Errorf("Sync failed: %v", err)
			finalMessage := color.RedString("✗") + " Failed to sync secrets\n" +
				color.RedString("Error: ") + err.Error()
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Display results.
		if syncDryRun {
			spinner.Stop()
			printSyncDryRun(result)
			spinner.FinalMSG = ""
			return nil
		}

		// Handle case where no secrets needed processing.
		if result.SecretsProcessed == 0 {
			finalMessage := color.GreenString("✓") + " No encrypted files found. Nothing to sync."
			spinner.FinalMSG = finalMessage
			return nil
		}

		// Log to audit trail.
		auditEntry := audit.LogWithUser("sync")
		auditEntry.UsersCount = result.UsersProcessed
		auditEntry.FilesCount = result.SecretsProcessed
		audit.Log(auditEntry)

		finalMessage := color.GreenString("✓") + " Secrets synced successfully\n" +
			fmt.Sprintf("  Re-encrypted %d secret file(s) for %d user(s).\n", result.SecretsProcessed, result.UsersProcessed) +
			"  New encryption key generated and distributed to all users."
		spinner.FinalMSG = finalMessage
		return nil
	},
}

// printSyncDryRun displays what would happen during a sync operation.
func printSyncDryRun(result *secrets.SyncResult) {
	fmt.Println()
	fmt.Println(color.YellowString("[dry-run]") + " Would sync secrets:")
	fmt.Println()

	if result.SecretsProcessed == 0 {
		fmt.Println("  No encrypted files found. Nothing to sync.")
		fmt.Println()
		fmt.Println(color.CyanString("No changes needed."))
		return
	}

	fmt.Printf("  - Decrypt %d secret file(s)\n", result.SecretsProcessed)
	fmt.Println("  - Generate new encryption key")
	fmt.Printf("  - Re-encrypt for %d user(s)\n", result.UsersProcessed)

	if result.UsersExcluded > 0 {
		fmt.Printf("  - Exclude %d user(s) from new key\n", result.UsersExcluded)
	}

	fmt.Printf("  - Re-encrypt %d secret file(s)\n", result.SecretsProcessed)
	fmt.Println()
	fmt.Println(color.CyanString("No changes made.") + " Run without --dry-run to execute.")
}
