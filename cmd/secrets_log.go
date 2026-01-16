package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/spf13/cobra"
)

var (
	logLimit     int
	logReverse   bool
	logUser      string
	logOperation string
	logSince     string
	logUntil     string
	logOneline   bool
	logJSON      bool
)

func init() {
	logCmd.Flags().IntVarP(&logLimit, "number", "n", 0, "limit number of entries shown")
	logCmd.Flags().BoolVar(&logReverse, "reverse", false, "show most recent entries first")
	logCmd.Flags().StringVar(&logUser, "user", "", "filter by user email")
	logCmd.Flags().StringVar(&logOperation, "operation", "", "filter by operation type (comma-separated)")
	logCmd.Flags().StringVar(&logSince, "since", "", "show entries after date (YYYY-MM-DD)")
	logCmd.Flags().StringVar(&logUntil, "until", "", "show entries before date (YYYY-MM-DD)")
	logCmd.Flags().BoolVar(&logOneline, "oneline", false, "compact one-line format")
	logCmd.Flags().BoolVar(&logJSON, "json", false, "output as JSON array")

	SecretsCmd.AddCommand(logCmd)
}

// resetLogCommandState resets the log command's global state for testing.
func resetLogCommandState() {
	logLimit = 0
	logReverse = false
	logUser = ""
	logOperation = ""
	logSince = ""
	logUntil = ""
	logOneline = false
	logJSON = false
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "View the audit log",
	Long: `Displays the audit log of secrets operations.

Shows who performed what operation and when. Use filters to narrow down
the results.

Examples:
  kanuka secrets log                              # View full log
  kanuka secrets log -n 10                        # Last 10 entries
  kanuka secrets log --reverse                    # Most recent first
  kanuka secrets log --user alice@example.com     # Filter by user
  kanuka secrets log --operation encrypt,decrypt  # Filter by operation
  kanuka secrets log --since 2024-01-01           # Filter by date
  kanuka secrets log --json                       # JSON output`,
	RunE: runLog,
}

func runLog(cmd *cobra.Command, args []string) error {
	Logger.Infof("Starting log command")

	spinner, cleanup := startSpinner("Loading audit log...", verbose)
	defer cleanup()

	Logger.Debugf("Initializing project settings")
	if err := configs.InitProjectSettings(); err != nil {
		spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to initialize project settings\n"
		return Logger.ErrorfAndReturn("failed to init project settings: %v", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	Logger.Debugf("Project path: %s", projectPath)

	if projectPath == "" {
		spinner.FinalMSG = ui.Error.Sprint("✗") + " Kānuka has not been initialized\n"
		fmt.Println(ui.Info.Sprint("→") + " Run " + ui.Code.Sprint("kanuka secrets init") + " first")
		return nil
	}

	// Get audit log path.
	logPath := audit.LogPath()
	if logPath == "" {
		spinner.FinalMSG = ui.Info.Sprint("ℹ") + " No audit log found. Operations will be logged after running any secrets command.\n"
		return nil
	}

	// Read log file.
	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		spinner.FinalMSG = ui.Info.Sprint("ℹ") + " No audit log found. Operations will be logged after running any secrets command.\n"
		return nil
	}
	if err != nil {
		spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to read audit log\n"
		return Logger.ErrorfAndReturn("failed to read audit log: %v", err)
	}

	// Parse entries.
	entries, err := audit.ParseEntries(data)
	if err != nil {
		return Logger.ErrorfAndReturn("failed to parse audit log: %v", err)
	}
	Logger.Debugf("Parsed %d entries from audit log", len(entries))

	if len(entries) == 0 {
		fmt.Println("No audit log entries found.")
		return nil
	}

	// Apply filters.
	filtered := entries

	if logUser != "" {
		filtered = filterByUser(filtered, logUser)
	}

	if logOperation != "" {
		ops := strings.Split(logOperation, ",")
		for i := range ops {
			ops[i] = strings.TrimSpace(ops[i])
		}
		filtered = filterByOperations(filtered, ops)
	}

	if logSince != "" {
		sinceTime, err := time.Parse("2006-01-02", logSince)
		if err != nil {
			return Logger.ErrorfAndReturn("invalid --since date format, use YYYY-MM-DD: %v", err)
		}
		filtered = filterSince(filtered, sinceTime)
	}

	if logUntil != "" {
		untilTime, err := time.Parse("2006-01-02", logUntil)
		if err != nil {
			return Logger.ErrorfAndReturn("invalid --until date format, use YYYY-MM-DD: %v", err)
		}
		// Include the entire day by setting to end of day.
		untilTime = untilTime.Add(24*time.Hour - time.Nanosecond)
		filtered = filterUntil(filtered, untilTime)
	}

	Logger.Debugf("After filtering: %d entries", len(filtered))

	// Apply ordering.
	if logReverse {
		for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		}
	}

	// Apply limit.
	if logLimit > 0 && len(filtered) > logLimit {
		if logReverse {
			// When reversed, limit takes first N (most recent).
			filtered = filtered[:logLimit]
		} else {
			// When not reversed, limit takes last N (most recent).
			filtered = filtered[len(filtered)-logLimit:]
		}
	}

	if len(filtered) == 0 {
		fmt.Println("No audit log entries found matching the filters.")
		return nil
	}

	// Output.
	if logJSON {
		if err := outputLogJSON(filtered); err != nil {
			spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to output log\n"
			return err
		}
		return nil
	}

	if logOneline {
		if err := outputLogOneline(filtered); err != nil {
			spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to output log\n"
			return err
		}
		return nil
	}

	if err := outputLogDefault(filtered); err != nil {
		spinner.FinalMSG = ui.Error.Sprint("✗") + " Failed to output log\n"
		return err
	}
	spinner.FinalMSG = ui.Success.Sprint("✓") + " Audit log displayed\n"
	return nil
}

func filterByUser(entries []audit.Entry, user string) []audit.Entry {
	var result []audit.Entry
	for _, e := range entries {
		if strings.EqualFold(e.User, user) {
			result = append(result, e)
		}
	}
	return result
}

func filterByOperations(entries []audit.Entry, ops []string) []audit.Entry {
	opSet := make(map[string]bool)
	for _, op := range ops {
		opSet[strings.ToLower(op)] = true
	}

	var result []audit.Entry
	for _, e := range entries {
		if opSet[strings.ToLower(e.Operation)] {
			result = append(result, e)
		}
	}
	return result
}

func filterSince(entries []audit.Entry, since time.Time) []audit.Entry {
	var result []audit.Entry
	for _, e := range entries {
		t, err := time.Parse("2006-01-02T15:04:05.000000Z", e.Timestamp)
		if err != nil {
			// Try alternate format.
			t, err = time.Parse(time.RFC3339, e.Timestamp)
		}
		if err != nil {
			continue
		}
		if !t.Before(since) {
			result = append(result, e)
		}
	}
	return result
}

func filterUntil(entries []audit.Entry, until time.Time) []audit.Entry {
	var result []audit.Entry
	for _, e := range entries {
		t, err := time.Parse("2006-01-02T15:04:05.000000Z", e.Timestamp)
		if err != nil {
			// Try alternate format.
			t, err = time.Parse(time.RFC3339, e.Timestamp)
		}
		if err != nil {
			continue
		}
		if !t.After(until) {
			result = append(result, e)
		}
	}
	return result
}

func outputLogJSON(entries []audit.Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entries to JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func outputLogOneline(entries []audit.Entry) error {
	for _, e := range entries {
		date := formatDate(e.Timestamp)
		details := formatDetailsOneline(e)
		fmt.Printf("%s %s %s %s\n", date, e.User, e.Operation, details)
	}
	return nil
}

func outputLogDefault(entries []audit.Entry) error {
	for _, e := range entries {
		datetime := formatDateTime(e.Timestamp)
		details := formatDetails(e)
		fmt.Printf("%-19s  %-25s  %-10s  %s\n", datetime, e.User, e.Operation, details)
	}
	return nil
}

func formatDate(ts string) string {
	t, err := time.Parse("2006-01-02T15:04:05.000000Z", ts)
	if err != nil {
		t, err = time.Parse(time.RFC3339, ts)
	}
	if err != nil {
		return ts[:10] // Fall back to first 10 chars.
	}
	return t.Format("2006-01-02")
}

func formatDateTime(ts string) string {
	t, err := time.Parse("2006-01-02T15:04:05.000000Z", ts)
	if err != nil {
		t, err = time.Parse(time.RFC3339, ts)
	}
	if err != nil {
		return ts[:19] // Fall back to first 19 chars.
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatDetails(e audit.Entry) string {
	switch e.Operation {
	case "encrypt", "decrypt":
		if len(e.Files) == 0 {
			return ""
		}
		if len(e.Files) > 3 {
			return fmt.Sprintf("%d files", len(e.Files))
		}
		return strings.Join(e.Files, ", ")
	case "register":
		return e.TargetUser
	case "revoke":
		if e.Device != "" {
			return fmt.Sprintf("%s (%s)", e.TargetUser, e.Device)
		}
		return e.TargetUser
	case "sync":
		return fmt.Sprintf("%d users, %d files", e.UsersCount, e.FilesCount)
	case "rotate":
		return ""
	case "clean":
		return fmt.Sprintf("removed %d entries", e.RemovedCount)
	case "import":
		return fmt.Sprintf("%s, %d files", e.Mode, e.FilesCount)
	case "export":
		return e.OutputPath
	case "init":
		return e.ProjectName
	case "create":
		return e.DeviceName
	default:
		return ""
	}
}

func formatDetailsOneline(e audit.Entry) string {
	switch e.Operation {
	case "encrypt", "decrypt":
		if len(e.Files) == 0 {
			return ""
		}
		return fmt.Sprintf("%d files", len(e.Files))
	case "register":
		return e.TargetUser
	case "revoke":
		if e.Device != "" {
			return fmt.Sprintf("%s (%s)", e.TargetUser, e.Device)
		}
		return e.TargetUser
	case "sync":
		return fmt.Sprintf("%d users, %d files", e.UsersCount, e.FilesCount)
	case "rotate":
		return ""
	case "clean":
		return fmt.Sprintf("removed %d", e.RemovedCount)
	case "import":
		return fmt.Sprintf("%s %d files", e.Mode, e.FilesCount)
	case "export":
		return e.OutputPath
	case "init":
		return e.ProjectName
	case "create":
		return e.DeviceName
	default:
		return ""
	}
}
