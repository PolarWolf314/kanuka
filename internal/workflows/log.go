package workflows

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PolarWolf314/kanuka/internal/audit"
	"github.com/PolarWolf314/kanuka/internal/configs"
	kerrors "github.com/PolarWolf314/kanuka/internal/errors"
)

// LogOptions configures the log workflow.
type LogOptions struct {
	// Limit is the maximum number of entries to return. 0 means no limit.
	Limit int

	// Reverse orders entries from most recent to oldest when true.
	Reverse bool

	// User filters entries by user email.
	User string

	// Operations filters entries by operation types (comma-separated).
	Operations string

	// Since filters entries after this date (YYYY-MM-DD format).
	Since string

	// Until filters entries before this date (YYYY-MM-DD format).
	Until string
}

// LogResult contains the outcome of a log operation.
type LogResult struct {
	// Entries are the filtered audit log entries.
	Entries []audit.Entry

	// TotalEntriesBeforeFilter is the count of entries before filtering.
	TotalEntriesBeforeFilter int
}

// Log reads and filters the audit log.
//
// Returns ErrProjectNotInitialized if the project has no .kanuka directory.
// Returns ErrNoFilesFound if no audit log exists.
// Returns ErrInvalidDateFormat if the date format is invalid.
func Log(ctx context.Context, opts LogOptions) (*LogResult, error) {
	if err := configs.InitProjectSettings(); err != nil {
		return nil, fmt.Errorf("initializing project settings: %w", err)
	}

	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return nil, kerrors.ErrProjectNotInitialized
	}

	// Get audit log path.
	logPath := audit.LogPath()
	if logPath == "" {
		return nil, kerrors.ErrNoFilesFound
	}

	// Read log file.
	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return nil, kerrors.ErrNoFilesFound
	}
	if err != nil {
		return nil, fmt.Errorf("reading audit log: %w", err)
	}

	// Parse entries.
	entries, err := audit.ParseEntries(data)
	if err != nil {
		return nil, fmt.Errorf("parsing audit log: %w", err)
	}

	result := &LogResult{
		TotalEntriesBeforeFilter: len(entries),
	}

	if len(entries) == 0 {
		result.Entries = entries
		return result, nil
	}

	// Apply filters.
	filtered := entries

	if opts.User != "" {
		filtered = filterByUser(filtered, opts.User)
	}

	if opts.Operations != "" {
		ops := strings.Split(opts.Operations, ",")
		for i := range ops {
			ops[i] = strings.TrimSpace(ops[i])
		}
		filtered = filterByOperations(filtered, ops)
	}

	if opts.Since != "" {
		sinceTime, err := time.Parse("2006-01-02", opts.Since)
		if err != nil {
			return nil, fmt.Errorf("%w: --since date format invalid, use YYYY-MM-DD", kerrors.ErrInvalidDateFormat)
		}
		filtered = filterSince(filtered, sinceTime)
	}

	if opts.Until != "" {
		untilTime, err := time.Parse("2006-01-02", opts.Until)
		if err != nil {
			return nil, fmt.Errorf("%w: --until date format invalid, use YYYY-MM-DD", kerrors.ErrInvalidDateFormat)
		}
		// Include the entire day by setting to end of day.
		untilTime = untilTime.Add(24*time.Hour - time.Nanosecond)
		filtered = filterUntil(filtered, untilTime)
	}

	// Apply ordering.
	if opts.Reverse {
		for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		}
	}

	// Apply limit.
	if opts.Limit > 0 && len(filtered) > opts.Limit {
		if opts.Reverse {
			// When reversed, limit takes first N (most recent).
			filtered = filtered[:opts.Limit]
		} else {
			// When not reversed, limit takes last N (most recent).
			filtered = filtered[len(filtered)-opts.Limit:]
		}
	}

	result.Entries = filtered
	return result, nil
}

// filterByUser filters entries by user email (case-insensitive).
func filterByUser(entries []audit.Entry, user string) []audit.Entry {
	var result []audit.Entry
	for _, e := range entries {
		if strings.EqualFold(e.User, user) {
			result = append(result, e)
		}
	}
	return result
}

// filterByOperations filters entries by operation types.
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

// filterSince filters entries to only include those at or after the given time.
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

// filterUntil filters entries to only include those at or before the given time.
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

// FormatDate formats a timestamp string to YYYY-MM-DD format.
func FormatDate(ts string) string {
	t, err := time.Parse("2006-01-02T15:04:05.000000Z", ts)
	if err != nil {
		t, err = time.Parse(time.RFC3339, ts)
	}
	if err != nil {
		if len(ts) >= 10 {
			return ts[:10]
		}
		return ts
	}
	return t.Format("2006-01-02")
}

// FormatDateTime formats a timestamp string to YYYY-MM-DD HH:MM:SS format.
func FormatDateTime(ts string) string {
	t, err := time.Parse("2006-01-02T15:04:05.000000Z", ts)
	if err != nil {
		t, err = time.Parse(time.RFC3339, ts)
	}
	if err != nil {
		if len(ts) >= 19 {
			return ts[:19]
		}
		return ts
	}
	return t.Format("2006-01-02 15:04:05")
}

// FormatDetails formats the details for a log entry in verbose format.
func FormatDetails(e audit.Entry) string {
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

// FormatDetailsOneline formats the details for a log entry in oneline format.
func FormatDetailsOneline(e audit.Entry) string {
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
