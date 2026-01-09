package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTestFile is a helper to write test files with 0644 permissions.
// #nosec G306 -- Test files are temporary and don't contain sensitive data.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil { // #nosec G306
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func TestResolveFiles_EmptyPatterns(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Empty patterns should return nil (caller uses default behavior).
	files, err := ResolveFiles([]string{}, tmpDir, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if files != nil {
		t.Errorf("Expected nil, got: %v", files)
	}
}

func TestResolveFiles_SingleFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test .env file.
	envFile := filepath.Join(tmpDir, ".env")
	writeTestFile(t, envFile, "TEST=value")

	files, err := ResolveFiles([]string{".env"}, tmpDir, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got: %d", len(files))
	}
	if files[0] != envFile {
		t.Errorf("Expected %s, got: %s", envFile, files[0])
	}
}

func TestResolveFiles_MultipleFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files.
	files := []string{".env", ".env.local", ".env.production"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		writeTestFile(t, path, "TEST=value")
	}

	resolved, err := ResolveFiles([]string{".env", ".env.local", ".env.production"}, tmpDir, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(resolved) != 3 {
		t.Fatalf("Expected 3 files, got: %d", len(resolved))
	}
}

func TestResolveFiles_Directory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory with .env files.
	subDir := filepath.Join(tmpDir, "services", "api")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	envFile := filepath.Join(subDir, ".env")
	writeTestFile(t, envFile, "TEST=value")

	// Resolve the directory.
	files, err := ResolveFiles([]string{"services/api/"}, tmpDir, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got: %d", len(files))
	}
	if files[0] != envFile {
		t.Errorf("Expected %s, got: %s", envFile, files[0])
	}
}

func TestResolveFiles_GlobPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories with .env files.
	for _, service := range []string{"api", "web", "worker"} {
		subDir := filepath.Join(tmpDir, "services", service)
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}
		envFile := filepath.Join(subDir, ".env")
		writeTestFile(t, envFile, "TEST=value")
	}

	files, err := ResolveFiles([]string{"services/*/.env"}, tmpDir, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("Expected 3 files, got: %d", len(files))
	}
}

func TestResolveFiles_DoubleStarGlob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested directories with .env files.
	paths := []string{
		filepath.Join(tmpDir, ".env"),
		filepath.Join(tmpDir, "services", "api", ".env"),
		filepath.Join(tmpDir, "services", "api", "config", ".env"),
	}
	for _, p := range paths {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		writeTestFile(t, p, "TEST=value")
	}

	files, err := ResolveFiles([]string{"**/.env"}, tmpDir, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("Expected 3 files, got: %d", len(files))
	}
}

func TestResolveFiles_NonExistentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = ResolveFiles([]string{"nonexistent.env"}, tmpDir, true)
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}
}

func TestResolveFiles_Deduplication(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file.
	envFile := filepath.Join(tmpDir, ".env")
	writeTestFile(t, envFile, "TEST=value")

	// Request same file multiple times.
	files, err := ResolveFiles([]string{".env", ".env", ".env"}, tmpDir, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 file (deduplicated), got: %d", len(files))
	}
}

func TestResolveFiles_ExcludesKanukaDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a .kanuka directory with files that should be ignored.
	kanukaDir := filepath.Join(tmpDir, ".kanuka")
	if err := os.MkdirAll(kanukaDir, 0755); err != nil {
		t.Fatalf("Failed to create .kanuka dir: %v", err)
	}
	kanukaEnv := filepath.Join(kanukaDir, ".env")
	writeTestFile(t, kanukaEnv, "SHOULD_BE_IGNORED=true")

	// Create a regular .env file.
	envFile := filepath.Join(tmpDir, ".env")
	writeTestFile(t, envFile, "TEST=value")

	// Use glob pattern that might match .kanuka directory contents.
	files, err := ResolveFiles([]string{"**/.env"}, tmpDir, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("Expected 1 file (excluding .kanuka dir), got: %d", len(files))
	}
	if files[0] != envFile {
		t.Errorf("Expected %s, got: %s", envFile, files[0])
	}
}

func TestResolveFiles_ForDecryption(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a .kanuka file.
	kanukaFile := filepath.Join(tmpDir, ".env.kanuka")
	writeTestFile(t, kanukaFile, "encrypted")

	files, err := ResolveFiles([]string{".env.kanuka"}, tmpDir, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got: %d", len(files))
	}
	if files[0] != kanukaFile {
		t.Errorf("Expected %s, got: %s", kanukaFile, files[0])
	}
}

func TestResolveFiles_WrongFileType(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a .kanuka file but try to use it for encryption.
	kanukaFile := filepath.Join(tmpDir, ".env.kanuka")
	writeTestFile(t, kanukaFile, "encrypted")

	_, err = ResolveFiles([]string{".env.kanuka"}, tmpDir, true)
	if err == nil {
		t.Fatal("Expected error when using .kanuka file for encryption")
	}
}

func TestIsEnvFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{".env", true},
		{".env.local", true},
		{".env.production", true},
		{"path/to/.env", true},
		{".env.kanuka", false},
		{".env.local.kanuka", false},
		{"config.toml", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isEnvFile(tt.path)
			if result != tt.expected {
				t.Errorf("isEnvFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsKanukaFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{".env.kanuka", true},
		{".env.local.kanuka", true},
		{"path/to/.env.kanuka", true},
		{".env", false},
		{".env.local", false},
		{"config.toml", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isKanukaFile(tt.path)
			if result != tt.expected {
				t.Errorf("isKanukaFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsInKanukaDir(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{".kanuka/secrets/user.kanuka", true},
		{".kanuka/config.toml", true},
		{"path/to/.kanuka/file", true},
		{".env", false},
		{"services/api/.env", false},
		{".env.kanuka", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isInKanukaDir(tt.path)
			if result != tt.expected {
				t.Errorf("isInKanukaDir(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}
