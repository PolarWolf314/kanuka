package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"errors"
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
