package decrypt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/cmd"
	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsMonorepoWithUnreadableDirs mirrors a real self-hosted-infrastructure
// monorepo: a single secrets store at the root, an .env in a service subdirectory,
// and container-owned runtime data (e.g. postgres pgdata) that kanuka cannot read.
// All tree-scanning commands must succeed without explicit file arguments, and
// decrypted plaintext must land with 0600 permissions.
func TestSecretsMonorepoWithUnreadableDirs(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping: running as root, which ignores directory permissions")
	}

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	tempDir, err := os.MkdirTemp("", "kanuka-test-monorepo-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempUserDir, err := os.MkdirTemp("", "kanuka-user-monorepo-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory: %v", err)
	}
	defer os.RemoveAll(tempUserDir)

	shared.SetupTestEnvironment(t, tempDir, tempUserDir, originalWd, originalUserSettings)
	shared.InitializeProject(t, tempDir, tempUserDir)

	// A service with secrets to protect.
	apiDir := filepath.Join(tempDir, "services", "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create services/api: %v", err)
	}
	envPath := filepath.Join(apiDir, ".env")
	// #nosec G306 -- Deliberately world-readable to verify decrypt tightens it.
	if err := os.WriteFile(envPath, []byte("DATABASE_URL=postgres://localhost/db\n"), 0644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Container-owned runtime data: unreadable recursively, no .env inside
	// that kanuka should care about.
	pgDataDir := filepath.Join(tempDir, "data", "postgres", "pgdata")
	if err := os.MkdirAll(filepath.Join(pgDataDir, "base"), 0755); err != nil {
		t.Fatalf("Failed to create pgdata: %v", err)
	}
	// #nosec G306 -- Fixture file inside fake postgres data, not sensitive.
	if err := os.WriteFile(filepath.Join(pgDataDir, "base", "PG_VERSION"), []byte("17\n"), 0644); err != nil {
		t.Fatalf("Failed to create pgdata file: %v", err)
	}
	if err := os.Chmod(pgDataDir, 0000); err != nil {
		t.Fatalf("Failed to lock pgdata: %v", err)
	}
	defer func() {
		if err := os.Chmod(pgDataDir, 0755); err != nil {
			t.Fatalf("Failed to restore pgdata permissions: %v", err)
		}
	}()

	// doctor reports warnings/errors through its exit code (1 = warnings,
	// 2 = errors) instead of a command error, so mock its exit function the
	// way the doctor integration tests do. The mock must be installed after
	// CreateTestCLIWithArgs, which resets command state.
	doctorExitCode := -1
	runCommand := func(subcommand string, extraArgs ...string) string {
		t.Helper()
		output, err := shared.CaptureOutput(func() error {
			rootCmd := shared.CreateTestCLIWithArgs(subcommand, extraArgs, nil, nil, false, false)
			if subcommand == "doctor" {
				cmd.SetDoctorExitFunc(func(code int) { doctorExitCode = code })
			}
			return rootCmd.Execute()
		})
		if err != nil {
			t.Errorf("%s command failed: %v\nOutput: %s", subcommand, err, output)
		}
		if strings.Contains(strings.ToLower(output), "permission denied") {
			t.Errorf("%s reported 'permission denied':\n%s", subcommand, output)
		}
		return output
	}

	// All tree-scanning commands must succeed without file arguments.
	runCommand("encrypt")
	runCommand("status")
	doctorOutput := runCommand("doctor")
	// Warnings (exit 1) are acceptable health observations; hard errors
	// (exit 2) mean the walk or checks actually failed.
	if doctorExitCode == 2 {
		t.Errorf("doctor exited with error code 2:\n%s", doctorOutput)
	}
	runCommand("export", "-o", filepath.Join(tempDir, "..", "kanuka-backup.tar.gz"))

	// Simulate a fresh clone: plaintext is gone, only encrypted files remain.
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove plaintext .env: %v", err)
	}

	runCommand("decrypt")

	// The decrypted plaintext must exist and be owner-only.
	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("Decrypted .env not found: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("Expected decrypted .env mode 0600, got %04o", perm)
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read decrypted .env: %v", err)
	}
	if !strings.Contains(string(content), "DATABASE_URL=postgres://localhost/db") {
		t.Errorf("Unexpected decrypted content: %q", string(content))
	}
}
