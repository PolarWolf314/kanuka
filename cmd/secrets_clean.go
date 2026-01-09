package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/ui"

	"github.com/spf13/cobra"
)

var (
	cleanForce  bool
	cleanDryRun bool
)

func init() {
	cleanCmd.Flags().BoolVar(&cleanForce, "force", false, "skip confirmation prompt")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "show what would be removed without making changes")
}

func resetCleanCommandState() {
	cleanForce = false
	cleanDryRun = false
}

// OrphanEntry represents an orphaned .kanuka file with no corresponding public key.
type OrphanEntry struct {
	UUID     string
	FilePath string
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove orphaned keys and inconsistent state",
	Long: `Removes orphaned entries detected by 'kanuka secrets access'.

An orphan is a .kanuka file in .kanuka/secrets/ that has no corresponding
public key in .kanuka/public_keys/. This can happen if:
  - A public key was manually deleted
  - A revoke operation was interrupted
  - Files were corrupted or partially restored

Use --dry-run to preview what would be removed.
Use --force to skip the confirmation prompt.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Logger.Infof("Starting clean command")

		Logger.Debugf("Initializing project settings")
		if err := configs.InitProjectSettings(); err != nil {
			return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
		}
		projectPath := configs.ProjectKanukaSettings.ProjectPath
		Logger.Debugf("Project path: %s", projectPath)

		if projectPath == "" {
			fmt.Println(ui.Error.Sprint("✗") + " Kanuka has not been initialized")
			fmt.Println(ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first")
			return nil
		}

		// Find orphaned entries.
		orphans, err := findOrphanedEntries()
		if err != nil {
			return Logger.ErrorfAndReturn("failed to find orphaned entries: %v", err)
		}

		if len(orphans) == 0 {
			fmt.Println(ui.Success.Sprint("✓") + " No orphaned entries found. Nothing to clean.")
			return nil
		}

		// Display orphans.
		if cleanDryRun {
			fmt.Printf("[dry-run] Would remove %d orphaned file(s):\n", len(orphans))
		} else {
			fmt.Printf("Found %d orphaned entry(ies):\n\n", len(orphans))
		}

		printOrphanTable(orphans, projectPath)

		// Dry run - don't delete anything.
		if cleanDryRun {
			fmt.Println("\nNo changes made.")
			return nil
		}

		// Confirm deletion (if not --force).
		if !cleanForce {
			fmt.Println("\nThis will permanently delete the orphaned files listed above.")
			fmt.Println("These files cannot be recovered.")
			fmt.Println()

			if !confirmCleanAction() {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Remove orphaned files.
		for _, orphan := range orphans {
			Logger.Debugf("Removing orphaned file: %s", orphan.FilePath)
			if err := os.Remove(orphan.FilePath); err != nil {
				return Logger.ErrorfAndReturn("failed to remove %s: %v", orphan.FilePath, err)
			}
		}

		// Log to audit trail.
		auditEntry := audit.LogWithUser("clean")
		auditEntry.RemovedCount = len(orphans)
		audit.Log(auditEntry)

		fmt.Printf("%s Removed %d orphaned file(s)\n", ui.Success.Sprint("✓"), len(orphans))
		return nil
	},
}

// findOrphanedEntries finds .kanuka files in secrets/ that have no corresponding public key.
func findOrphanedEntries() ([]OrphanEntry, error) {
	secretsDir := configs.ProjectKanukaSettings.ProjectSecretsPath
	publicKeysDir := configs.ProjectKanukaSettings.ProjectPublicKeyPath

	Logger.Debugf("Scanning secrets dir for orphans: %s", secretsDir)
	Logger.Debugf("Checking against public keys dir: %s", publicKeysDir)

	var orphans []OrphanEntry

	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return orphans, nil
		}
		return nil, fmt.Errorf("failed to read secrets directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".kanuka") {
			continue
		}

		uuid := strings.TrimSuffix(entry.Name(), ".kanuka")
		publicKeyPath := filepath.Join(publicKeysDir, uuid+".pub")

		if !fileExists(publicKeyPath) {
			orphanPath := filepath.Join(secretsDir, entry.Name())
			orphans = append(orphans, OrphanEntry{
				UUID:     uuid,
				FilePath: orphanPath,
			})
			Logger.Debugf("Found orphan: UUID=%s, file=%s", uuid, orphanPath)
		}
	}

	return orphans, nil
}

// printOrphanTable prints a formatted table of orphaned entries.
func printOrphanTable(orphans []OrphanEntry, projectPath string) {
	// Calculate column widths.
	uuidWidth := 36 // Standard UUID length.

	fmt.Printf("  %-*s  %s\n", uuidWidth, "UUID", "FILE")

	for _, orphan := range orphans {
		// Show relative path for cleaner output.
		relPath, err := filepath.Rel(projectPath, orphan.FilePath)
		if err != nil {
			relPath = orphan.FilePath
		}
		fmt.Printf("  %-*s  %s\n", uuidWidth, orphan.UUID, relPath)
	}
}

// confirmCleanAction prompts the user to confirm the clean operation.
func confirmCleanAction() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Do you want to continue? [y/N]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		Logger.Errorf("Failed to read response: %v", err)
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
