package log_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsLogIntegration contains integration tests for the `kanuka secrets log` command.
func TestSecretsLogIntegration(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("LogInEmptyFolder", func(t *testing.T) {
		testLogInEmptyFolder(t, originalWd, originalUserSettings)
	})

	t.Run("LogInInitializedFolderWithNoAuditLog", func(t *testing.T) {
		testLogInInitializedFolderWithNoAuditLog(t, originalWd, originalUserSettings)
	})

	t.Run("LogShowsEntriesAfterEncrypt", func(t *testing.T) {
		testLogShowsEntriesAfterEncrypt(t, originalWd, originalUserSettings)
	})

	t.Run("LogWithLimitFlag", func(t *testing.T) {
		testLogWithLimitFlag(t, originalWd, originalUserSettings)
	})

	t.Run("LogWithReverseFlag", func(t *testing.T) {
		testLogWithReverseFlag(t, originalWd, originalUserSettings)
	})

	t.Run("LogWithUserFilter", func(t *testing.T) {
		testLogWithUserFilter(t, originalWd, originalUserSettings)
	})

	t.Run("LogWithOperationFilter", func(t *testing.T) {
		testLogWithOperationFilter(t, originalWd, originalUserSettings)
	})

	t.Run("LogWithOnelineFormat", func(t *testing.T) {
		testLogWithOnelineFormat(t, originalWd, originalUserSettings)
	})

	t.Run("LogWithJSONFormat", func(t *testing.T) {
		testLogWithJSONFormat(t, originalWd, originalUserSettings)
	})
}

// testLogInEmptyFolder tests log command in an empty folder (should fail gracefully).
func testLogInEmptyFolder(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-log-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("log", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "Kānuka has not been initialized") {
		t.Errorf("Expected 'not initialized' message not found in output: %s", output)
	}
}

// testLogInInitializedFolderWithNoAuditLog tests log in initialized folder with no audit log.
func testLogInInitializedFolderWithNoAuditLog(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-log-no-audit-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Remove audit log if it was created during init.
	auditPath := filepath.Join(tempDir, ".kanuka", "audit.jsonl")
	_ = os.Remove(auditPath)

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("log", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed unexpectedly: %v", err)
	}

	if !strings.Contains(output, "No audit log found") {
		t.Errorf("Expected 'no audit log found' message not found in output: %s", output)
	}
}

// testLogShowsEntriesAfterEncrypt tests that log shows entries after encrypt operation.
func testLogShowsEntriesAfterEncrypt(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-log-entries-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	envContent := "DATABASE_URL=postgres://localhost:5432/mydb\n"
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Run encrypt to generate audit log entry.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Now check the log.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("log", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Should show encrypt operation.
	if !strings.Contains(output, "encrypt") {
		t.Errorf("Expected 'encrypt' operation in log output: %s", output)
	}

	// Should show the user email.
	if !strings.Contains(output, "testuser@example.com") {
		t.Errorf("Expected user email in log output: %s", output)
	}
}

// testLogWithLimitFlag tests the -n flag to limit number of entries.
func testLogWithLimitFlag(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-log-limit-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt multiple .env files to generate multiple log entries.
	for i := 0; i < 3; i++ {
		envPath := filepath.Join(tempDir, ".env")
		// #nosec G306 -- Writing a file that should be modifiable.
		if err := os.WriteFile(envPath, []byte("KEY=value\n"), 0644); err != nil {
			t.Fatalf("Failed to create .env file: %v", err)
		}

		_, err = shared.CaptureOutput(func() error {
			cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
			return cmd.Execute()
		})
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}
	}

	// Get log with limit of 1.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("log", []string{"-n", "1"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Count lines (should only be 1 entry).
	lines := strings.Split(strings.TrimSpace(output), "\n")
	encryptLines := 0
	for _, line := range lines {
		if strings.Contains(line, "encrypt") {
			encryptLines++
		}
	}
	if encryptLines != 1 {
		t.Errorf("Expected 1 encrypt entry with -n 1, got %d. Output: %s", encryptLines, output)
	}
}

// testLogWithReverseFlag tests the --reverse flag.
func testLogWithReverseFlag(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-log-reverse-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(envPath, []byte("KEY=value\n"), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Run decrypt.
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	// Get log without reverse (oldest first).
	outputNormal, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("log", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Get log with reverse (newest first).
	outputReverse, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("log", []string{"--reverse"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// In normal order, first line should contain encrypt or init (older).
	// In reverse order, first line should contain decrypt (newer).
	normalLines := strings.Split(strings.TrimSpace(outputNormal), "\n")
	reverseLines := strings.Split(strings.TrimSpace(outputReverse), "\n")

	// The last line in normal should be first in reverse.
	if len(normalLines) > 1 && len(reverseLines) > 1 {
		// Last operation in normal should be first in reverse.
		lastNormal := normalLines[len(normalLines)-1]
		firstReverse := reverseLines[0]

		if strings.Contains(lastNormal, "decrypt") && !strings.Contains(firstReverse, "decrypt") {
			t.Errorf("Expected decrypt to be first in reversed output. Normal last: %s, Reverse first: %s", lastNormal, firstReverse)
		}
	}
}

// testLogWithUserFilter tests the --user filter.
func testLogWithUserFilter(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-log-user-filter-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(envPath, []byte("KEY=value\n"), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Filter by existing user.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("log", []string{"--user", "testuser@example.com"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	if !strings.Contains(output, "encrypt") {
		t.Errorf("Expected entries for testuser@example.com, got: %s", output)
	}

	// Filter by non-existing user.
	output, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("log", []string{"--user", "nobody@example.com"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	if !strings.Contains(output, "No audit log entries found matching the filters") {
		t.Errorf("Expected no entries for nobody@example.com, got: %s", output)
	}
}

// testLogWithOperationFilter tests the --operation filter.
func testLogWithOperationFilter(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-log-op-filter-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(envPath, []byte("KEY=value\n"), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Filter by encrypt operation.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("log", []string{"--operation", "encrypt"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	if !strings.Contains(output, "encrypt") {
		t.Errorf("Expected encrypt entries, got: %s", output)
	}

	// Filter by non-existing operation.
	output, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("log", []string{"--operation", "delete"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	if !strings.Contains(output, "No audit log entries found matching the filters") {
		t.Errorf("Expected no entries for delete operation, got: %s", output)
	}
}

// testLogWithOnelineFormat tests the --oneline format.
func testLogWithOnelineFormat(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-log-oneline-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(envPath, []byte("KEY=value\n"), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Get log with oneline format.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("log", []string{"--oneline"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Oneline format should show date, user, operation on same line.
	if !strings.Contains(output, "encrypt") {
		t.Errorf("Expected encrypt in oneline output: %s", output)
	}

	// Verify it's a compact format (contains date pattern like 2024-01-01).
	if !strings.Contains(output, "-") {
		t.Errorf("Expected date in oneline output: %s", output)
	}
}

// testLogWithJSONFormat tests the --json format.
func testLogWithJSONFormat(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-log-json-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// Create and encrypt a .env file.
	envPath := filepath.Join(tempDir, ".env")
	// #nosec G306 -- Writing a file that should be modifiable.
	if err := os.WriteFile(envPath, []byte("KEY=value\n"), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Get log with JSON format.
	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLIWithArgs("log", []string{"--json"}, nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// JSON format should start with [ and end with ].
	trimmedOutput := strings.TrimSpace(output)
	if !strings.HasPrefix(trimmedOutput, "[") || !strings.HasSuffix(trimmedOutput, "]") {
		t.Errorf("Expected JSON array output, got: %s", output)
	}

	// Should contain encrypt operation.
	if !strings.Contains(output, `"op": "encrypt"`) && !strings.Contains(output, `"op":"encrypt"`) {
		t.Errorf("Expected 'op: encrypt' in JSON output: %s", output)
	}

	// Should contain user field.
	if !strings.Contains(output, `"user"`) {
		t.Errorf("Expected 'user' field in JSON output: %s", output)
	}
}
