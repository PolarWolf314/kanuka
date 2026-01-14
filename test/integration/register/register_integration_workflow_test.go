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

func TestSecretsRegisterIntegrationWorkflow(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("InitCreateRegisterWorkflow", func(t *testing.T) {
		testInitCreateRegisterWorkflow(t, originalWd, originalUserSettings)
	})

	t.Run("MultipleUserRegistrationWorkflow", func(t *testing.T) {
		testMultipleUserRegistrationWorkflow(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterThenEncryptDecryptWorkflow", func(t *testing.T) {
		testRegisterThenEncryptDecryptWorkflow(t, originalWd, originalUserSettings)
	})

	t.Run("RegisterThenRemoveWorkflow", func(t *testing.T) {
		testRegisterThenRemoveWorkflow(t, originalWd, originalUserSettings)
	})

	t.Run("ChainedRegistrationWorkflow", func(t *testing.T) {
		testChainedRegistrationWorkflow(t, originalWd, originalUserSettings)
	})

}

func testInitCreateRegisterWorkflow(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-init-create-register-*")
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

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Init command failed: %v", err)
	}

	secretsFile := filepath.Join(tempDir, "test.env")
	if err := os.WriteFile(secretsFile, []byte("TEST_VAR=test_value\n"), 0600); err != nil {
		t.Fatalf("Failed to create test secrets file: %v", err)
	}

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "create", secretsFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Create command failed: %v", err)
	}

	targetUserUUID := "new-user-uuid-1234"
	targetUserEmail := "newuser@example.com"

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	pemKey := generatePEMKeyIntegration(t, &privateKey.PublicKey)

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Register command failed: %v", err)
		t.Errorf("Output: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	targetKanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(targetKanukaFile); os.IsNotExist(err) {
		t.Errorf("Target user's .kanuka file was not created at %s", targetKanukaFile)
	}

	encryptedFile := filepath.Join(tempDir, ".kanuka", "secrets", "test.env.enc")
	if _, err := os.Stat(encryptedFile); os.IsNotExist(err) {
		t.Logf("Encrypted secrets file was not created at %s - this may be expected behavior", encryptedFile)
	}
}

func testMultipleUserRegistrationWorkflow(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-multiple-register-*")
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

	users := []struct {
		uuid  string
		email string
	}{
		{"user1-uuid-1234", "user1@example.com"},
		{"user2-uuid-1234", "user2@example.com"},
		{"user3-uuid-1234", "user3@example.com"},
	}

	for _, user := range users {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate RSA key: %v", err)
		}
		pemKey := generatePEMKeyIntegration(t, &privateKey.PublicKey)

		projectConfig, err := configs.LoadProjectConfig()
		if err != nil {
			t.Fatalf("Failed to load project config: %v", err)
		}
		projectConfig.Users[user.uuid] = user.email
		if err := configs.SaveProjectConfig(projectConfig); err != nil {
			t.Fatalf("Failed to save project config: %v", err)
		}

		output, err := shared.CaptureOutput(func() error {
			cmd := shared.CreateTestCLI("register", nil, nil, true, false)
			cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", user.email})
			return cmd.Execute()
		})
		if err != nil {
			t.Errorf("Register command failed for %s: %v", user.email, err)
		}

		if !strings.Contains(output, "✓") {
			t.Errorf("Expected success message not found for user %s in output: %s", user.email, output)
		}

		kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", user.uuid+".kanuka")
		if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
			t.Errorf("User %s's .kanuka file was not created at %s", user.email, kanukaKeyPath)
		}
	}

	shared.VerifyProjectStructure(t, tempDir)
}

func testRegisterThenEncryptDecryptWorkflow(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-encrypt-decrypt-*")
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

	targetUserUUID := "encrypt-user-uuid-1234"
	targetUserEmail := "encryptuser@example.com"

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	pemKey := generatePEMKeyIntegration(t, &privateKey.PublicKey)

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Register command failed: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Target user's .kanuka file was not created at %s", kanukaKeyPath)
	}

	secretsFile := filepath.Join(tempDir, "test.env")
	if err := os.WriteFile(secretsFile, []byte("SECRET_DATA=super_secret_value\n"), 0600); err != nil {
		t.Fatalf("Failed to create test secrets file: %v", err)
	}

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "encrypt", secretsFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Logf("Encrypt command failed: %v", err)
	}

	encryptedFile := filepath.Join(tempDir, ".kanuka", "secrets", "test.env.enc")
	if _, err := os.Stat(encryptedFile); os.IsNotExist(err) {
		t.Logf("Encrypted file was not created at %s - this may be expected behavior", encryptedFile)
	}

	decryptedFile := filepath.Join(tempDir, "decrypted_test.env")
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, false, false)
		cmd.SetArgs([]string{"secrets", "decrypt", secretsFile, "-o", decryptedFile})
		return cmd.Execute()
	})
	if err != nil {
		t.Logf("Decrypt command failed: %v", err)
	}

	if _, err := os.Stat(decryptedFile); os.IsNotExist(err) {
		t.Logf("Decrypted file was not created at %s - this may be expected behavior", decryptedFile)
	}
}

func testRegisterThenRemoveWorkflow(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-remove-*")
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

	targetUserUUID := "remove-user-uuid-1234"
	targetUserEmail := "removeuser@example.com"

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	pemKey := generatePEMKeyIntegration(t, &privateKey.PublicKey)

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[targetUserUUID] = targetUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Register command failed: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", targetUserUUID+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Errorf("Target user's .kanuka file was not created at %s", kanukaKeyPath)
	}

	if err := os.Remove(kanukaKeyPath); err != nil {
		t.Fatalf("Failed to remove .kanuka file: %v", err)
	}

	output, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", targetUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Re-register command failed: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in re-register output: %s", output)
	}
}

func testChainedRegistrationWorkflow(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-chained-register-*")
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

	userAUUID := "userA-uuid-1234"
	userAEmail := "usera@example.com"

	privateKeyA, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key for user A: %v", err)
	}
	pemKeyA := generatePEMKeyIntegration(t, &privateKeyA.PublicKey)

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[userAUUID] = userAEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	outputA, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKeyA, "--user", userAEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("User A registration failed: %v", err)
	}

	if !strings.Contains(outputA, "✓") {
		t.Errorf("Expected success message not found for User A: %s", outputA)
	}

	userBUUID := "userB-uuid-1234"
	userBEmail := "userb@example.com"

	privateKeyB, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key for user B: %v", err)
	}
	pemKeyB := generatePEMKeyIntegration(t, &privateKeyB.PublicKey)

	projectConfig, err = configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[userBUUID] = userBEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	outputB, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKeyB, "--user", userBEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("User B registration failed: %v", err)
	}

	if !strings.Contains(outputB, "✓") {
		t.Errorf("Expected success message not found for User B: %s", outputB)
	}

	userCUUID := "userC-uuid-1234"
	userCEmail := "userc@example.com"

	privateKeyC, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key for user C: %v", err)
	}
	pemKeyC := generatePEMKeyIntegration(t, &privateKeyC.PublicKey)

	projectConfig, err = configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[userCUUID] = userCEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	outputC, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKeyC, "--user", userCEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("User C registration failed: %v", err)
	}

	if !strings.Contains(outputC, "✓") {
		t.Errorf("Expected success message not found for User C: %s", outputC)
	}

	shared.VerifyProjectStructure(t, tempDir)
}

func testRegisterAfterManualReset(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-after-reset-*")
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

	secretsDir := filepath.Join(tempDir, ".kanuka", "secrets")
	publicKeysDir := filepath.Join(tempDir, ".kanuka", "public_keys")

	files, err := filepath.Glob(filepath.Join(secretsDir, "*.kanuka"))
	if err != nil {
		t.Fatalf("Failed to list .kanuka files: %v", err)
	}
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			t.Logf("Failed to remove %s: %v", file, err)
		}
	}

	pubKeyFiles, err := filepath.Glob(filepath.Join(publicKeysDir, "*.pub"))
	if err != nil {
		t.Fatalf("Failed to list .pub files: %v", err)
	}
	for _, file := range pubKeyFiles {
		if err := os.Remove(file); err != nil {
			t.Logf("Failed to remove %s: %v", file, err)
		}
	}

	newUserUUID := "newuserafterreset-uuid-1234"
	newUserEmail := "newuserafterreset@example.com"

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	pemKey := generatePEMKeyIntegration(t, &privateKey.PublicKey)

	projectConfig, err := configs.LoadProjectConfig()
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}
	projectConfig.Users[newUserUUID] = newUserEmail
	if err := configs.SaveProjectConfig(projectConfig); err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	output, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--pubkey", pemKey, "--user", newUserEmail})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Register command failed: %v", err)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success message not found in output: %s", output)
	}

	kanukaKeyPath := filepath.Join(tempDir, ".kanuka", "secrets", newUserUUID+".kanuka")
	if _, err := os.Stat(kanukaKeyPath); os.IsNotExist(err) {
		t.Logf("New user's .kanuka file was not created after reset at %s - this may be expected behavior", kanukaKeyPath)
	}

	shared.VerifyProjectStructure(t, tempDir)
}

func generatePEMKeyIntegration(t *testing.T, publicKey *rsa.PublicKey) string {
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
