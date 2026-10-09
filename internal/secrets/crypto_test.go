package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

// testSymKey returns a valid 32-byte symmetric key for tests.
func testSymKey() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}

func TestDecryptFiles_WritesPlaintextWithTightPermissions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-decrypt-perms-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	symKey := testSymKey()
	envPath := filepath.Join(tmpDir, ".env")
	kanukaPath := envPath + ".kanuka"

	// Create and encrypt a plaintext env file.
	writeTestFile(t, envPath, "SECRET=value\n")
	if err := EncryptFiles(symKey, []string{envPath}, false); err != nil {
		t.Fatalf("EncryptFiles failed: %v", err)
	}

	// Simulate a fresh clone: no plaintext on disk yet.
	if err := os.Remove(envPath); err != nil {
		t.Fatalf("Failed to remove plaintext: %v", err)
	}

	if err := DecryptFiles(symKey, []string{kanukaPath}, false); err != nil {
		t.Fatalf("DecryptFiles failed: %v", err)
	}

	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("Failed to stat decrypted file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("Expected decrypted file mode 0600, got %04o", perm)
	}
}

func TestDecryptFiles_TightensPreExistingFilePermissions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kanuka-test-decrypt-tighten-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	symKey := testSymKey()
	envPath := filepath.Join(tmpDir, ".env")
	kanukaPath := envPath + ".kanuka"

	writeTestFile(t, envPath, "SECRET=value\n")
	if err := EncryptFiles(symKey, []string{envPath}, false); err != nil {
		t.Fatalf("EncryptFiles failed: %v", err)
	}

	// Leave a stale world-readable plaintext behind (e.g. created by an
	// editor or an older kanuka version), then decrypt over it.
	writeTestFile(t, envPath, "STALE=content\n")

	if err := DecryptFiles(symKey, []string{kanukaPath}, false); err != nil {
		t.Fatalf("DecryptFiles failed: %v", err)
	}

	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("Failed to stat decrypted file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("Expected pre-existing file to be tightened to 0600, got %04o", perm)
	}

	// The decrypted content must replace the stale content.
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read decrypted file: %v", err)
	}
	if string(content) != "SECRET=value\n" {
		t.Errorf("Unexpected decrypted content: %q", string(content))
	}
}
