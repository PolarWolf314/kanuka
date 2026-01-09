package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
)

func TestLog_CreatesFile(t *testing.T) {
	// Create temp directory for project.
	tempDir, err := os.MkdirTemp("", "kanuka-audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .kanuka directory.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka dir: %v", err)
	}

	// Set up project settings.
	originalSettings := configs.ProjectKanukaSettings
	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectPath: tempDir,
	}
	defer func() {
		configs.ProjectKanukaSettings = originalSettings
	}()

	// Log an entry.
	entry := Entry{
		User:      "test@example.com",
		UserUUID:  "test-uuid",
		Operation: "encrypt",
		Files:     []string{".env"},
	}
	Log(entry)

	// Verify file was created.
	logPath := filepath.Join(kanukaDir, "audit.jsonl")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("Audit log file was not created")
	}
}

func TestLog_AppendsEntries(t *testing.T) {
	// Create temp directory for project.
	tempDir, err := os.MkdirTemp("", "kanuka-audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .kanuka directory.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka dir: %v", err)
	}

	// Set up project settings.
	originalSettings := configs.ProjectKanukaSettings
	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectPath: tempDir,
	}
	defer func() {
		configs.ProjectKanukaSettings = originalSettings
	}()

	// Log multiple entries.
	Log(Entry{User: "alice@example.com", Operation: "encrypt"})
	Log(Entry{User: "bob@example.com", Operation: "decrypt"})
	Log(Entry{User: "charlie@example.com", Operation: "register"})

	// Read and verify.
	logPath := filepath.Join(kanukaDir, "audit.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read audit log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

func TestLog_ValidJSON(t *testing.T) {
	// Create temp directory for project.
	tempDir, err := os.MkdirTemp("", "kanuka-audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .kanuka directory.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka dir: %v", err)
	}

	// Set up project settings.
	originalSettings := configs.ProjectKanukaSettings
	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectPath: tempDir,
	}
	defer func() {
		configs.ProjectKanukaSettings = originalSettings
	}()

	// Log an entry with various fields.
	entry := Entry{
		User:       "test@example.com",
		UserUUID:   "test-uuid",
		Operation:  "encrypt",
		Files:      []string{".env", ".env.local"},
		TargetUser: "target@example.com",
	}
	Log(entry)

	// Read and parse.
	logPath := filepath.Join(kanukaDir, "audit.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read audit log: %v", err)
	}

	var parsed Entry
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &parsed); err != nil {
		t.Fatalf("Entry is not valid JSON: %v", err)
	}

	if parsed.User != "test@example.com" {
		t.Errorf("Expected user test@example.com, got %s", parsed.User)
	}
	if parsed.Operation != "encrypt" {
		t.Errorf("Expected operation encrypt, got %s", parsed.Operation)
	}
	if len(parsed.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(parsed.Files))
	}
}

func TestLog_TimestampFormat(t *testing.T) {
	// Create temp directory for project.
	tempDir, err := os.MkdirTemp("", "kanuka-audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .kanuka directory.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka dir: %v", err)
	}

	// Set up project settings.
	originalSettings := configs.ProjectKanukaSettings
	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectPath: tempDir,
	}
	defer func() {
		configs.ProjectKanukaSettings = originalSettings
	}()

	// Log an entry without timestamp (should be auto-set).
	entry := Entry{
		User:      "test@example.com",
		Operation: "encrypt",
	}
	Log(entry)

	// Read and parse.
	logPath := filepath.Join(kanukaDir, "audit.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read audit log: %v", err)
	}

	var parsed Entry
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &parsed); err != nil {
		t.Fatalf("Entry is not valid JSON: %v", err)
	}

	// Check timestamp format: 2006-01-02T15:04:05.000000Z.
	if parsed.Timestamp == "" {
		t.Errorf("Timestamp should be auto-set")
	}
	if !strings.HasSuffix(parsed.Timestamp, "Z") {
		t.Errorf("Timestamp should end with Z, got %s", parsed.Timestamp)
	}
	if !strings.Contains(parsed.Timestamp, ".") {
		t.Errorf("Timestamp should contain microseconds, got %s", parsed.Timestamp)
	}
}

func TestLog_OmitsEmptyFields(t *testing.T) {
	// Create temp directory for project.
	tempDir, err := os.MkdirTemp("", "kanuka-audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .kanuka directory.
	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka dir: %v", err)
	}

	// Set up project settings.
	originalSettings := configs.ProjectKanukaSettings
	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectPath: tempDir,
	}
	defer func() {
		configs.ProjectKanukaSettings = originalSettings
	}()

	// Log an entry with only required fields.
	entry := Entry{
		User:      "test@example.com",
		Operation: "rotate",
	}
	Log(entry)

	// Read raw data.
	logPath := filepath.Join(kanukaDir, "audit.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read audit log: %v", err)
	}

	line := strings.TrimSpace(string(data))

	// Check that optional fields are not present.
	if strings.Contains(line, `"files"`) {
		t.Errorf("Empty files field should be omitted")
	}
	if strings.Contains(line, `"target_user"`) {
		t.Errorf("Empty target_user field should be omitted")
	}
	if strings.Contains(line, `"mode"`) {
		t.Errorf("Empty mode field should be omitted")
	}
}

func TestLog_NoProjectPath(t *testing.T) {
	// Set up project settings with no path.
	originalSettings := configs.ProjectKanukaSettings
	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectPath: "",
	}
	defer func() {
		configs.ProjectKanukaSettings = originalSettings
	}()

	// Log should not panic or error.
	entry := Entry{
		User:      "test@example.com",
		Operation: "encrypt",
	}
	Log(entry) // Should silently do nothing.
}

func TestParseEntries_ValidData(t *testing.T) {
	data := []byte(`{"ts":"2024-01-15T10:30:00.123456Z","user":"alice@example.com","op":"encrypt"}
{"ts":"2024-01-15T10:35:00.456789Z","user":"bob@example.com","op":"decrypt"}
`)

	entries, err := ParseEntries(data)
	if err != nil {
		t.Fatalf("ParseEntries failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	if entries[0].User != "alice@example.com" {
		t.Errorf("Expected first user alice@example.com, got %s", entries[0].User)
	}
	if entries[1].User != "bob@example.com" {
		t.Errorf("Expected second user bob@example.com, got %s", entries[1].User)
	}
}

func TestParseEntries_SkipsMalformedLines(t *testing.T) {
	data := []byte(`{"ts":"2024-01-15T10:30:00.123456Z","user":"alice@example.com","op":"encrypt"}
this is not valid json
{"ts":"2024-01-15T10:35:00.456789Z","user":"bob@example.com","op":"decrypt"}
`)

	entries, err := ParseEntries(data)
	if err != nil {
		t.Fatalf("ParseEntries failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 valid entries (malformed should be skipped), got %d", len(entries))
	}
}

func TestParseEntries_EmptyData(t *testing.T) {
	entries, err := ParseEntries([]byte{})
	if err != nil {
		t.Fatalf("ParseEntries failed: %v", err)
	}

	if entries != nil {
		t.Errorf("Expected nil entries for empty data, got %v", entries)
	}
}

func TestLogPath_WithProject(t *testing.T) {
	originalSettings := configs.ProjectKanukaSettings
	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectPath: "/test/project",
	}
	defer func() {
		configs.ProjectKanukaSettings = originalSettings
	}()

	path := LogPath()
	expected := "/test/project/.kanuka/audit.jsonl"
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestLogPath_NoProject(t *testing.T) {
	originalSettings := configs.ProjectKanukaSettings
	configs.ProjectKanukaSettings = &configs.ProjectSettings{
		ProjectPath: "",
	}
	defer func() {
		configs.ProjectKanukaSettings = originalSettings
	}()

	path := LogPath()
	if path != "" {
		t.Errorf("Expected empty path, got %s", path)
	}
}
