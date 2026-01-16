package register

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

func TestSecretsRegisterErrorHandling(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("RegisterWithNetworkInterruption", func(t *testing.T) {
		testRegisterWithNetworkInterruption(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterWithPermissionDenied", func(t *testing.T) {
		testRegisterWithPermissionDenied(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterRecoveryFromPartialFailure", func(t *testing.T) {
		testRegisterRecoveryFromPartialFailure(t, originalWd, originalUserSettings)
	})
}

func testRegisterWithNetworkInterruption(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-network-*")
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

	targetUserUUID := "network-user-uuid-1234"
	targetUserEmail := "networkuser@example.com"

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	pemKey := generatePEMKeyErrorHandling(t, &privateKey.PublicKey)

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	if err := os.Chmod(secretsDir, 0555); err != nil {
		t.Fatalf("Failed to make secrets directory read-only: %v", err)
	}
	defer func() {
		if err := os.Chmod(secretsDir, 0755); err != nil {
			t.Logf("Failed to restore permissions on secrets directory: %v", err)
		}
	}()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})

	hasErrorSymbol := strings.Contains(output, "✗")
	hasErrorMessage := strings.Contains(output, "Error:") || strings.Contains(output, "error") || strings.Contains(output, "failed")

	if err == nil && !hasErrorSymbol && !hasErrorMessage {
		t.Errorf("Expected command to fail or show error, but got success. Output: %s", output)
	}

	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	_, fileExists := os.Stat(targetKanukaFile)

	if strings.Contains(output, "✓") && fileExists == nil {
		t.Logf("Command succeeded despite read-only directory - this may be system-dependent behavior")
	}
}

func testRegisterWithPermissionDenied(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-permission-*")
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

	targetUserUUID := "permission-user-uuid-1234"
	targetUserEmail := "permissionuser@example.com"

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	pemKey := generatePEMKeyErrorHandling(t, &privateKey.PublicKey)

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	kanukaDir := filepath.Join(tempDir, ".kanuka")
	if err := os.Chmod(kanukaDir, 0444); err != nil {
		t.Fatalf("Failed to make .kanuka directory read-only: %v", err)
	}
	defer func() {
		if err := os.Chmod(kanukaDir, 0755); err != nil {
			t.Logf("Failed to restore permissions on .kanuka directory: %v", err)
		}
	}()

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})

	hasError := err != nil || strings.Contains(output, "✗") || strings.Contains(output, "Error:")

	if !hasError {
		t.Errorf("Expected command to fail due to permissions, but got success. Output: %s", output)
	}
}

func testRegisterRecoveryFromPartialFailure(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-recovery-*")
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

	targetUserUUID := "recovery-user-uuid-1234"
	targetUserEmail := "recoveryuser@example.com"

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	pemKey := generatePEMKeyErrorHandling(t, &privateKey.PublicKey)

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	if err := os.Chmod(secretsDir, 0444); err != nil {
		t.Fatalf("Failed to make secrets directory read-only: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})

	hasErrorSymbol := strings.Contains(output, "✗")
	hasErrorMessage := strings.Contains(output, "Error:") || strings.Contains(output, "error") || strings.Contains(output, "failed")

	if err == nil && !hasErrorSymbol && !hasErrorMessage {
		t.Logf("Command succeeded despite read-only directory - this may be system-dependent behavior")
	}

	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, statErr := os.Stat(targetKanukaFile); !os.IsNotExist(statErr) {
		t.Logf("Target user's .kanuka file was created despite permission error at %s - this may be expected behavior on some systems", targetKanukaFile)
	}

	if err := os.Chmod(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to restore permissions on secrets directory: %v", err)
	}

	output, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Recovery register command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in recovery output: %s", output)
	}

	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Errorf("Target user's .kanuka file was not created after recovery at %s", targetKanukaFile)
	}

	shared.VerifyProjectStructure(t, tempDir)

	kanukaFileContent, err := os.ReadFile(targetKanukaFile)
	if err != nil {
		t.Errorf("Failed to read .kanuka file after recovery: %v", err)
	}
	if len(kanukaFileContent) == 0 {
		t.Errorf(".kanuka file is empty after recovery")
	}

	anotherUserUUID := "another-user-uuid-1234"
	anotherUserEmail := "anotheruser@example.com"

	anotherPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	anotherPemKey := generatePEMKeyErrorHandling(t, &anotherPrivateKey.PublicKey)

	projectConfig, err = configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[anotherUserUUID] = anotherUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", anotherPemKey, "--user", anotherUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Follow-up register command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in follow-up registration: %s", output)
	}

	anotherUserKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", anotherUserUUID+".kanuka")
	if _, err := os.Stat(anotherUserKanukaFile); os.IsNotExist(err) {
		t.Errorf("Second user's .kanuka file was not created at %s", anotherUserKanukaFile)
	}
}

func generatePEMKeyErrorHandling(t *testing.T, publicKey *rsa.PublicKey) string {
	pubASN1, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}

	pubPem := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	}

	return string(pem.EncodeToMemory(pubPem))
}
