package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestParseOpenSSHPrivateKey_ValidUnencrypted(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Marshal to OpenSSH format (returns *pem.Block)
	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}

	// Encode to PEM bytes using encoding/pem
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Parse using our function
	parsed, err := parseOpenSSHPrivateKey(pemBytes, nil)
	if err != nil {
		t.Fatalf("parseOpenSSHPrivateKey failed: %v", err)
	}

	// Verify the parsed key matches the original
	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
	if parsed.E != privateKey.E {
		t.Error("parsed key exponent does not match original")
	}
	if parsed.D.Cmp(privateKey.D) != 0 {
		t.Error("parsed key private exponent does not match original")
	}
}

func TestParseOpenSSHPrivateKey_PassphraseProtected(t *testing.T) {
	passphrase := "test-passphrase-123"

	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Marshal to OpenSSH format with passphrase
	pemBlock, err := ssh.MarshalPrivateKeyWithPassphrase(privateKey, "", []byte(passphrase))
	if err != nil {
		t.Fatalf("failed to marshal private key with passphrase: %v", err)
	}

	pemBytes := pem.EncodeToMemory(pemBlock)

	// Test 1: Try to parse without passphrase - should return ErrPassphraseRequired
	_, err = parseOpenSSHPrivateKey(pemBytes, nil)
	if err == nil {
		t.Fatal("expected error when parsing passphrase-protected key without passphrase")
	}
	if !errors.Is(err, ErrPassphraseRequired) {
		t.Errorf("expected ErrPassphraseRequired, got: %v", err)
	}

	// Test 2: Parse with correct passphrase - should succeed
	parsed, err := parseOpenSSHPrivateKey(pemBytes, []byte(passphrase))
	if err != nil {
		t.Fatalf("parseOpenSSHPrivateKey with correct passphrase failed: %v", err)
	}

	// Verify the parsed key matches the original
	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}

	// Test 3: Parse with wrong passphrase - should return error
	_, err = parseOpenSSHPrivateKey(pemBytes, []byte("wrong-passphrase"))
	if err == nil {
		t.Fatal("expected error when parsing with wrong passphrase")
	}
}

func TestParseOpenSSHPrivateKey_NonRSAKey(t *testing.T) {
	t.Run("Ed25519KeyNotSupported", func(t *testing.T) {
		// This is a real Ed25519 OpenSSH private key format structure (test-only)
		// Generated using: ssh-keygen -t ed25519 -f test -N ""
		ed25519Key := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBHK9toM3stMC4dU+W0zOhpSYe3y8T0B7fF3vCXqoU+VwAAAJDe9N2Z3vTd
mQAAAAtzc2gtZWQyNTUxOQAAACBHK9toM3stMC4dU+W0zOhpSYe3y8T0B7fF3vCXqoU+Vw
AAAED+oSOemJl+aJvYwEqaGDhJT1DZW3o0QVQJCA6bQd3Y4Ecr22gzey0wLh1T5bTM6GlJ
h7fLxPQHt8Xe8JeqhT5XAAAADHRlc3RAZXhhbXBsZQE=
-----END OPENSSH PRIVATE KEY-----`

		_, err := parseOpenSSHPrivateKey([]byte(ed25519Key), nil)
		if err == nil {
			t.Fatal("expected error when parsing Ed25519 key")
		}
		if errors.Is(err, ErrPassphraseRequired) {
			t.Error("should not return ErrPassphraseRequired for non-RSA key")
		}
		// The error should mention unsupported key type
		t.Logf("Got expected error: %v", err)
	})

	t.Run("ECDSAKeyNotSupported", func(t *testing.T) {
		// This is a real ECDSA OpenSSH private key format structure (test-only)
		// Generated using: ssh-keygen -t ecdsa -b 256 -f test -N ""
		ecdsaKey := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAaAAAABNlY2RzYS
1zaGEyLW5pc3RwMjU2AAAACG5pc3RwMjU2AAAAQQShcmh3dqwvV4IvbuVo51gclyncEhbj
QHLvnYMxt5A0LndMr5dOBRmQPO8UA04iZuVh81eJSXb48g7FJLDldSFgAAAAqNv9EUnb/R
FJAAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBKFyaHd2rC9Xgi9u
5WjnWByXKdwSFuNAcu+dgzG3kDQud0yvl04FGZA87xQDTiJm5WHzV4lJdvjyDsUksOV1IW
AAAAAga54bLSlctRzA6GmwuZpO6WZxgrBqjDk1QSyzpuS6opwAAAAMdGVzdEBleGFtcGxl
AQIDBA==
-----END OPENSSH PRIVATE KEY-----`

		_, err := parseOpenSSHPrivateKey([]byte(ecdsaKey), nil)
		if err == nil {
			t.Fatal("expected error when parsing ECDSA key")
		}
		if errors.Is(err, ErrPassphraseRequired) {
			t.Error("should not return ErrPassphraseRequired for non-RSA key")
		}
		// The error should mention unsupported key type
		t.Logf("Got expected error: %v", err)
	})
}

func TestParseOpenSSHPrivateKey_InvalidData(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{
			name: "EmptyData",
			data: []byte{},
		},
		{
			name: "RandomBytes",
			data: []byte("not a valid key at all"),
		},
		{
			name: "InvalidPEMHeader",
			data: []byte("-----BEGIN FAKE KEY-----\nnotvalidbase64\n-----END FAKE KEY-----"),
		},
		{
			name: "CorruptedOpenSSHKey",
			data: []byte("-----BEGIN OPENSSH PRIVATE KEY-----\ncorrupted\n-----END OPENSSH PRIVATE KEY-----"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseOpenSSHPrivateKey(tc.data, nil)
			if err == nil {
				t.Error("expected error for invalid data")
			}
			// Should not return ErrPassphraseRequired for invalid data
			if errors.Is(err, ErrPassphraseRequired) {
				t.Error("should not return ErrPassphraseRequired for invalid data")
			}
		})
	}
}

func TestParseOpenSSHPrivateKey_EmptyPassphrase(t *testing.T) {
	// Generate a test key without passphrase
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}

	pemBytes := pem.EncodeToMemory(pemBlock)

	// Parsing with empty passphrase should work for unencrypted key
	parsed, err := parseOpenSSHPrivateKey(pemBytes, []byte{})
	if err != nil {
		t.Fatalf("parseOpenSSHPrivateKey with empty passphrase failed: %v", err)
	}

	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
}

func TestErrPassphraseRequired(t *testing.T) {
	// Ensure the error variable is properly defined
	if ErrPassphraseRequired == nil {
		t.Fatal("ErrPassphraseRequired should not be nil")
	}

	expectedMsg := "private key is passphrase-protected"
	if ErrPassphraseRequired.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, ErrPassphraseRequired.Error())
	}
}

// Tests for ParsePrivateKeyBytes - the main entry point for parsing private keys

func TestParsePrivateKeyBytes_PKCS1(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Encode in PKCS#1 PEM format (traditional "RSA PRIVATE KEY")
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Parse using ParsePrivateKeyBytes
	parsed, err := ParsePrivateKeyBytes(pemBytes)
	if err != nil {
		t.Fatalf("ParsePrivateKeyBytes failed for PKCS#1: %v", err)
	}

	// Verify the parsed key matches the original
	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
	if parsed.E != privateKey.E {
		t.Error("parsed key exponent does not match original")
	}
}

func TestParsePrivateKeyBytes_PKCS8(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Encode in PKCS#8 PEM format ("PRIVATE KEY")
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to marshal PKCS#8 private key: %v", err)
	}
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Parse using ParsePrivateKeyBytes
	parsed, err := ParsePrivateKeyBytes(pemBytes)
	if err != nil {
		t.Fatalf("ParsePrivateKeyBytes failed for PKCS#8: %v", err)
	}

	// Verify the parsed key matches the original
	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
	if parsed.E != privateKey.E {
		t.Error("parsed key exponent does not match original")
	}
}

func TestParsePrivateKeyBytes_OpenSSH(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Marshal to OpenSSH format
	sshPemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("failed to marshal private key to OpenSSH format: %v", err)
	}
	pemBytes := pem.EncodeToMemory(sshPemBlock)

	// Parse using ParsePrivateKeyBytes
	parsed, err := ParsePrivateKeyBytes(pemBytes)
	if err != nil {
		t.Fatalf("ParsePrivateKeyBytes failed for OpenSSH: %v", err)
	}

	// Verify the parsed key matches the original
	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
	if parsed.E != privateKey.E {
		t.Error("parsed key exponent does not match original")
	}
}

func TestParsePrivateKeyBytes_InvalidFormats(t *testing.T) {
	testCases := []struct {
		name        string
		data        []byte
		expectError string
	}{
		{
			name:        "EmptyData",
			data:        []byte{},
			expectError: "failed to decode PEM block",
		},
		{
			name:        "NotPEM",
			data:        []byte("not a PEM encoded key"),
			expectError: "failed to decode PEM block",
		},
		{
			name: "UnsupportedPEMType",
			data: pem.EncodeToMemory(&pem.Block{
				Type:  "UNSUPPORTED KEY TYPE",
				Bytes: []byte("fake key data"),
			}),
			expectError: "unsupported private key format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParsePrivateKeyBytes(tc.data)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.expectError)
			}
			if !containsString(err.Error(), tc.expectError) {
				t.Errorf("expected error containing %q, got %q", tc.expectError, err.Error())
			}
		})
	}
}

// Tests for LoadPrivateKey - loading from file

func TestLoadPrivateKey_PKCS1File(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Create a temp file with PKCS#1 PEM format
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "privkey")

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	if err := os.WriteFile(keyPath, pemBytes, 0600); err != nil {
		t.Fatalf("failed to write test key file: %v", err)
	}

	// Load using LoadPrivateKey
	loaded, err := LoadPrivateKey(keyPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey failed: %v", err)
	}

	// Verify the loaded key matches the original
	if loaded.N.Cmp(privateKey.N) != 0 {
		t.Error("loaded key modulus does not match original")
	}
}

func TestLoadPrivateKey_OpenSSHFile(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Create a temp file with OpenSSH format
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "privkey")

	sshPemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("failed to marshal private key to OpenSSH format: %v", err)
	}
	pemBytes := pem.EncodeToMemory(sshPemBlock)

	if err := os.WriteFile(keyPath, pemBytes, 0600); err != nil {
		t.Fatalf("failed to write test key file: %v", err)
	}

	// Load using LoadPrivateKey
	loaded, err := LoadPrivateKey(keyPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey failed: %v", err)
	}

	// Verify the loaded key matches the original
	if loaded.N.Cmp(privateKey.N) != 0 {
		t.Error("loaded key modulus does not match original")
	}
}

func TestLoadPrivateKey_PKCS8File(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Create a temp file with PKCS#8 PEM format
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "privkey")

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to marshal PKCS#8 private key: %v", err)
	}
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	if err := os.WriteFile(keyPath, pemBytes, 0600); err != nil {
		t.Fatalf("failed to write test key file: %v", err)
	}

	// Load using LoadPrivateKey
	loaded, err := LoadPrivateKey(keyPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey failed: %v", err)
	}

	// Verify the loaded key matches the original
	if loaded.N.Cmp(privateKey.N) != 0 {
		t.Error("loaded key modulus does not match original")
	}
}

func TestLoadPrivateKey_FileNotFound(t *testing.T) {
	_, err := LoadPrivateKey("/nonexistent/path/to/key")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !os.IsNotExist(err) {
		t.Logf("Got error: %v", err)
	}
}

// Tests for ParsePrivateKeyText - parsing from string

func TestParsePrivateKeyText_PKCS1(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Encode in PKCS#1 PEM format
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	pemText := string(pem.EncodeToMemory(pemBlock))

	// Parse using ParsePrivateKeyText
	parsed, err := ParsePrivateKeyText(pemText)
	if err != nil {
		t.Fatalf("ParsePrivateKeyText failed for PKCS#1: %v", err)
	}

	// Verify the parsed key matches the original
	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
	if parsed.E != privateKey.E {
		t.Error("parsed key exponent does not match original")
	}
}

func TestParsePrivateKeyText_PKCS8(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Encode in PKCS#8 PEM format
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to marshal PKCS#8 private key: %v", err)
	}
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	}
	pemText := string(pem.EncodeToMemory(pemBlock))

	// Parse using ParsePrivateKeyText
	parsed, err := ParsePrivateKeyText(pemText)
	if err != nil {
		t.Fatalf("ParsePrivateKeyText failed for PKCS#8: %v", err)
	}

	// Verify the parsed key matches the original
	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
	if parsed.E != privateKey.E {
		t.Error("parsed key exponent does not match original")
	}
}

func TestParsePrivateKeyText_OpenSSH(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Marshal to OpenSSH format
	sshPemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("failed to marshal private key to OpenSSH format: %v", err)
	}
	pemText := string(pem.EncodeToMemory(sshPemBlock))

	// Parse using ParsePrivateKeyText
	parsed, err := ParsePrivateKeyText(pemText)
	if err != nil {
		t.Fatalf("ParsePrivateKeyText failed for OpenSSH: %v", err)
	}

	// Verify the parsed key matches the original
	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
	if parsed.E != privateKey.E {
		t.Error("parsed key exponent does not match original")
	}
}

func TestParsePrivateKeyText_EmptyString(t *testing.T) {
	_, err := ParsePrivateKeyText("")
	if err == nil {
		t.Fatal("expected error for empty string, got nil")
	}
	expectedMsg := "private key text is empty"
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestParsePrivateKeyText_InvalidFormat(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectError string
	}{
		{
			name:        "NotPEMFormat",
			input:       "not a PEM encoded key",
			expectError: "private key text does not appear to be in PEM format",
		},
		{
			name:        "RandomText",
			input:       "ssh-rsa AAAA... this is not a private key",
			expectError: "private key text does not appear to be in PEM format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParsePrivateKeyText(tc.input)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.expectError)
			}
			if err.Error() != tc.expectError {
				t.Errorf("expected error %q, got %q", tc.expectError, err.Error())
			}
		})
	}
}

func TestParsePrivateKeyText_WhitespaceHandling(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Encode in PKCS#1 PEM format
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	pemText := string(pem.EncodeToMemory(pemBlock))

	// Add leading and trailing whitespace
	paddedText := "\n\n  " + pemText + "  \n\n"

	// Parse using ParsePrivateKeyText
	parsed, err := ParsePrivateKeyText(paddedText)
	if err != nil {
		t.Fatalf("ParsePrivateKeyText failed with whitespace padding: %v", err)
	}

	// Verify the parsed key matches the original
	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
	if parsed.E != privateKey.E {
		t.Error("parsed key exponent does not match original")
	}
}

// Helper function for string contains check.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Tests for ParsePrivateKeyBytesWithPassphrase

func TestParsePrivateKeyBytesWithPassphrase_UnencryptedKey(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Marshal to OpenSSH format (unencrypted)
	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Parse without passphrase - should work
	parsed, err := ParsePrivateKeyBytesWithPassphrase(pemBytes, nil)
	if err != nil {
		t.Fatalf("ParsePrivateKeyBytesWithPassphrase failed for unencrypted key: %v", err)
	}

	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
}

func TestParsePrivateKeyBytesWithPassphrase_EncryptedKeyCorrectPassphrase(t *testing.T) {
	passphrase := "test-passphrase-123"

	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Marshal to OpenSSH format with passphrase
	pemBlock, err := ssh.MarshalPrivateKeyWithPassphrase(privateKey, "", []byte(passphrase))
	if err != nil {
		t.Fatalf("failed to marshal private key with passphrase: %v", err)
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Parse with correct passphrase - should work
	parsed, err := ParsePrivateKeyBytesWithPassphrase(pemBytes, []byte(passphrase))
	if err != nil {
		t.Fatalf("ParsePrivateKeyBytesWithPassphrase failed with correct passphrase: %v", err)
	}

	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
}

func TestParsePrivateKeyBytesWithPassphrase_EncryptedKeyNoPassphrase(t *testing.T) {
	passphrase := "test-passphrase-123"

	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Marshal to OpenSSH format with passphrase
	pemBlock, err := ssh.MarshalPrivateKeyWithPassphrase(privateKey, "", []byte(passphrase))
	if err != nil {
		t.Fatalf("failed to marshal private key with passphrase: %v", err)
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Parse without passphrase - should return ErrPassphraseRequired
	_, err = ParsePrivateKeyBytesWithPassphrase(pemBytes, nil)
	if err == nil {
		t.Fatal("expected error when parsing encrypted key without passphrase")
	}
	if !errors.Is(err, ErrPassphraseRequired) {
		t.Errorf("expected ErrPassphraseRequired, got: %v", err)
	}
}

func TestParsePrivateKeyBytesWithPassphrase_EncryptedKeyWrongPassphrase(t *testing.T) {
	passphrase := "correct-passphrase"
	wrongPassphrase := "wrong-passphrase"

	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Marshal to OpenSSH format with passphrase
	pemBlock, err := ssh.MarshalPrivateKeyWithPassphrase(privateKey, "", []byte(passphrase))
	if err != nil {
		t.Fatalf("failed to marshal private key with passphrase: %v", err)
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Parse with wrong passphrase - should return ErrPassphraseRequired
	_, err = ParsePrivateKeyBytesWithPassphrase(pemBytes, []byte(wrongPassphrase))
	if err == nil {
		t.Fatal("expected error when parsing encrypted key with wrong passphrase")
	}
	// Should return ErrPassphraseRequired for wrong passphrase
	if !errors.Is(err, ErrPassphraseRequired) {
		t.Errorf("expected ErrPassphraseRequired for wrong passphrase, got: %v", err)
	}
}

func TestParsePrivateKeyBytesWithPassphrase_PKCS1Unencrypted(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Encode in PKCS#1 PEM format (unencrypted)
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Parse with nil passphrase - should work
	parsed, err := ParsePrivateKeyBytesWithPassphrase(pemBytes, nil)
	if err != nil {
		t.Fatalf("ParsePrivateKeyBytesWithPassphrase failed for unencrypted PKCS#1: %v", err)
	}

	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
}

func TestParsePrivateKeyBytesWithPassphrase_PKCS8Unencrypted(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Encode in PKCS#8 PEM format (unencrypted)
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to marshal PKCS#8 private key: %v", err)
	}
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Parse with nil passphrase - should work
	parsed, err := ParsePrivateKeyBytesWithPassphrase(pemBytes, nil)
	if err != nil {
		t.Fatalf("ParsePrivateKeyBytesWithPassphrase failed for unencrypted PKCS#8: %v", err)
	}

	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
}

// Tests for LoadPrivateKeyFromBytesWithPrompt in non-TTY environment

func TestLoadPrivateKeyFromBytesWithPrompt_UnencryptedKey(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Marshal to OpenSSH format (unencrypted)
	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Should work without prompting since key is unencrypted
	parsed, err := LoadPrivateKeyFromBytesWithPrompt(pemBytes)
	if err != nil {
		t.Fatalf("LoadPrivateKeyFromBytesWithPrompt failed for unencrypted key: %v", err)
	}

	if parsed.N.Cmp(privateKey.N) != 0 {
		t.Error("parsed key modulus does not match original")
	}
}

func TestLoadPrivateKeyFromBytesWithPrompt_EncryptedKeyNonTTY(t *testing.T) {
	passphrase := "test-passphrase"

	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Marshal to OpenSSH format with passphrase
	pemBlock, err := ssh.MarshalPrivateKeyWithPassphrase(privateKey, "", []byte(passphrase))
	if err != nil {
		t.Fatalf("failed to marshal private key with passphrase: %v", err)
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// In test environment (non-TTY), should fail with appropriate error
	_, err = LoadPrivateKeyFromBytesWithPrompt(pemBytes)
	if err == nil {
		t.Fatal("expected error when loading encrypted key in non-TTY environment")
	}

	// Should mention that stdin is not a terminal
	if !containsString(err.Error(), "not a terminal") {
		t.Errorf("expected error about non-terminal, got: %v", err)
	}
}

// Test LoadPrivateKey with file

func TestLoadPrivateKey_EncryptedOpenSSHFile_NonTTY(t *testing.T) {
	passphrase := "test-passphrase"

	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Create temp file with encrypted OpenSSH key
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "privkey")

	pemBlock, err := ssh.MarshalPrivateKeyWithPassphrase(privateKey, "", []byte(passphrase))
	if err != nil {
		t.Fatalf("failed to marshal private key with passphrase: %v", err)
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	if err := os.WriteFile(keyPath, pemBytes, 0600); err != nil {
		t.Fatalf("failed to write test key file: %v", err)
	}

	// In test environment (non-TTY), should fail with appropriate error
	_, err = LoadPrivateKey(keyPath)
	if err == nil {
		t.Fatal("expected error when loading encrypted key in non-TTY environment")
	}

	// Should mention that stdin is not a terminal
	if !containsString(err.Error(), "not a terminal") {
		t.Errorf("expected error about non-terminal, got: %v", err)
	}
}
