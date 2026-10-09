package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

// lockDir removes read/execute permissions on an existing directory so that
// directory walks cannot read it. It returns a cleanup function restoring
// the directory's permissions so the test can remove it again.
// The test must be skipped when running as root, since root ignores
// directory permissions.
func lockDir(t *testing.T, path string) func() {
	t.Helper()

	if os.Geteuid() == 0 {
		t.Skip("skipping: running as root, which ignores directory permissions")
	}

	if err := os.Chmod(path, 0000); err != nil {
		t.Fatalf("Failed to chmod dir %s: %v", path, err)
	}

	return func() {
		if err := os.Chmod(path, 0755); err != nil {
			t.Fatalf("Failed to restore permissions on %s: %v", path, err)
		}
	}
}

func TestFindEnvOrKanukaFiles_SkipsUnreadableSubdirs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-walk-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// A readable .env at the root.
	writeTestFile(t, filepath.Join(tmpDir, ".env"), "ROOT=1")

	// A readable nested .env.
	apiDir := filepath.Join(tmpDir, "services", "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	writeTestFile(t, filepath.Join(apiDir, ".env"), "API=1")

	// An unreadable subtree containing a decoy .env, like container-owned
	// runtime data (e.g. postgres pgdata).
	pgDataDir := filepath.Join(tmpDir, "data", "postgres", "pgdata")
	if err := os.MkdirAll(pgDataDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	writeTestFile(t, filepath.Join(pgDataDir, ".env"), "UNREADABLE=1")
	cleanup := lockDir(t, pgDataDir)
	defer cleanup()

	files, err := FindEnvOrKanukaFiles(tmpDir, []string{}, false)
	if err != nil {
		t.Fatalf("Expected walk to skip unreadable dirs without error, got: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 env files, got %d: %v", len(files), files)
	}

	expected := map[string]bool{
		filepath.Join(tmpDir, ".env"):                    true,
		filepath.Join(tmpDir, "services", "api", ".env"): true,
	}
	for _, f := range files {
		if !expected[f] {
			t.Errorf("Unexpected file in results: %s", f)
		}
	}
}

func TestFindEnvOrKanukaFiles_UnreadableRootStillErrors(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-walk-root-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	unreadableRoot := filepath.Join(tmpDir, "locked")
	if err := os.MkdirAll(unreadableRoot, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	cleanup := lockDir(t, unreadableRoot)
	defer cleanup()

	// Walking a root that cannot be read at all is a real error, not
	// something to skip silently.
	_, err = FindEnvOrKanukaFiles(unreadableRoot, []string{}, false)
	if err == nil {
		t.Fatal("Expected an error when the walk root itself is unreadable")
	}
}

func TestFindEnvOrKanukaFiles_IgnoresDotKanukaDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-walk-kanuka-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writeTestFile(t, filepath.Join(tmpDir, ".env"), "ROOT=1")

	kanukaDir := filepath.Join(tmpDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka dir: %v", err)
	}
	writeTestFile(t, filepath.Join(kanukaDir, ".env"), "IGNORED=1")

	files, err := FindEnvOrKanukaFiles(tmpDir, []string{}, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(files) != 1 || files[0] != filepath.Join(tmpDir, ".env") {
		t.Fatalf("Expected only the root .env, got: %v", files)
	}
}
