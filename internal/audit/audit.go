package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/PolarWolf314/kanuka/internal/configs"
)

// Entry represents a single audit log entry.
type Entry struct {
	Timestamp string `json:"ts"`   // RFC3339 with microseconds.
	User      string `json:"user"` // Email of user performing action.
	UserUUID  string `json:"uuid"` // UUID of user performing action.
	Operation string `json:"op"`   // Operation name.

	// Optional fields depending on operation.
	Files        []string `json:"files,omitempty"`         // For encrypt/decrypt.
	TargetUser   string   `json:"target_user,omitempty"`   // For register/revoke.
	TargetUUID   string   `json:"target_uuid,omitempty"`   // For register/revoke.
	Device       string   `json:"device,omitempty"`        // For device-specific revoke.
	UsersCount   int      `json:"users_count,omitempty"`   // For sync.
	FilesCount   int      `json:"files_count,omitempty"`   // For sync/import.
	RemovedCount int      `json:"removed_count,omitempty"` // For clean.
	Mode         string   `json:"mode,omitempty"`          // For import (merge/replace).
	OutputPath   string   `json:"output_path,omitempty"`   // For export.
	ProjectName  string   `json:"project_name,omitempty"`  // For init.
	ProjectUUID  string   `json:"project_uuid,omitempty"`  // For init.
	DeviceName   string   `json:"device_name,omitempty"`   // For create.
}

// Log appends an entry to the audit log.
// If logging fails, it logs a warning but does not return an error.
// Operations should not fail just because audit logging failed.
func Log(entry Entry) {
	// Set timestamp if not already set.
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	}

	// Get project path.
	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		// Project not initialized, skip logging.
		return
	}

	logPath := filepath.Join(projectPath, ".kanuka", "audit.jsonl")

	// Open file for appending (create if doesn't exist).
	// #nosec G306 -- audit log should be readable by team members.
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Log warning but don't fail the operation.
		return
	}
	defer f.Close()

	// Marshal entry to JSON.
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	// Write entry with newline.
	_, _ = f.Write(append(data, '\n'))
}

// LogWithUser is a convenience function that populates user fields from config.
func LogWithUser(op string) Entry {
	entry := Entry{Operation: op}

	userConfig, err := configs.LoadUserConfig()
	if err != nil {
		return entry
	}

	entry.User = userConfig.User.Email
	entry.UserUUID = userConfig.User.UUID

	return entry
}

// LogPath returns the path to the audit log file.
// Returns empty string if project is not initialized.
func LogPath() string {
	projectPath := configs.ProjectKanukaSettings.ProjectPath
	if projectPath == "" {
		return ""
	}
	return filepath.Join(projectPath, ".kanuka", "audit.jsonl")
}

// ReadEntries reads all entries from the audit log.
// Returns an empty slice if the log doesn't exist.
func ReadEntries() ([]Entry, error) {
	logPath := LogPath()
	if logPath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return ParseEntries(data)
}

// ParseEntries parses JSON Lines data into audit entries.
// Malformed lines are silently skipped.
func ParseEntries(data []byte) ([]Entry, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var entries []Entry
	start := 0

	for i := 0; i <= len(data); i++ {
		if i == len(data) || data[i] == '\n' {
			line := data[start:i]
			start = i + 1

			if len(line) == 0 {
				continue
			}

			var entry Entry
			if err := json.Unmarshal(line, &entry); err != nil {
				// Skip malformed entries.
				continue
			}
			entries = append(entries, entry)
		}
	}

	return entries, nil
}
