package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarWolf314/kanuka/internal/configs"
	"github.com/PolarWolf314/kanuka/test/integration/shared"
)

// TestSecretsCreateIntegrationWorkflow contains integration workflow tests for the `kanuka secrets create` command.
func TestSecretsCreateIntegrationWorkflow(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get original working directory: %v", err)
	}

	originalUserSettings := configs.UserKanukaSettings

	t.Run("CreateThenRegisterWorkflow", func(t *testing.T) {
		testCreateThenRegisterWorkflow(t, originalWd, originalUserSettings)
	})

	t.Run("CreateThenEncryptWorkflow", func(t *testing.T) {
		testCreateThenEncryptWorkflow(t, originalWd, originalUserSettings)
	})

	t.Run("MultipleUsersWorkflow", func(t *testing.T) {
		testMultipleUsersWorkflow(t, originalWd, originalUserSettings)
	})
}

// Tests create then register workflow - verify created keys work with register command.
func testCreateThenRegisterWorkflow(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-workflow-*")
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
	
	// Initialize project structure only
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Clean up any existing keys first
	username := configs.UserKanukaSettings.Username
	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	publicKeyPath := filepath.Join(tempUserDir, "keys", projectName+".pub")
	projectPublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", username+".pub")
	
	os.Remove(privateKeyPath)
	os.Remove(publicKeyPath)
	os.Remove(projectPublicKeyPath)

	// Step 1: Create keys
	createOutput, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Create command failed: %v", err)
	}

	// Verify create was successful
	if !strings.Contains(createOutput, "✓") {
		t.Errorf("Create command didn't show success: %s", createOutput)
	}

	publicKeyPathCheck := filepath.Join(tempDir, ".kanuka", "public_keys", username+".pub")
	
	// Verify public key was created
	if _, err := os.Stat(publicKeyPathCheck); os.IsNotExist(err) {
		t.Fatalf("Public key was not created by create command")
	}

	// Step 2: Try to register the user (simulate someone with permissions granting access)
	registerOutput, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", username})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Register command failed: %v", err)
		t.Errorf("Register output: %s", registerOutput)
	}

	// Verify register was successful
	if !strings.Contains(registerOutput, "✓") && !strings.Contains(registerOutput, "success") {
		t.Errorf("Register command didn't show success: %s", registerOutput)
	}

	// Verify .kanuka file was created
	kanukaFilePath := filepath.Join(tempDir, ".kanuka", "secrets", username+".kanuka")
	if _, err := os.Stat(kanukaFilePath); os.IsNotExist(err) {
		t.Errorf("Kanuka file was not created by register command")
	}

	// Verify the workflow instructions were shown in create output
	if !strings.Contains(createOutput, "kanuka secrets add "+username) {
		t.Errorf("Create output didn't show register instructions: %s", createOutput)
	}
}

// Tests create then encrypt workflow - verify workflow after gaining access.
func testCreateThenEncryptWorkflow(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-encrypt-workflow-*")
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
	
	// Initialize project structure only
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Clean up any existing keys first
	username := configs.UserKanukaSettings.Username
	projectName := filepath.Base(tempDir)
	privateKeyPath := filepath.Join(tempUserDir, "keys", projectName)
	publicKeyPath := filepath.Join(tempUserDir, "keys", projectName+".pub")
	projectPublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", username+".pub")
	
	os.Remove(privateKeyPath)
	os.Remove(publicKeyPath)
	os.Remove(projectPublicKeyPath)

	// Step 1: Create keys
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Create command failed: %v", err)
	}

	// Step 2: Register user (grant access)
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", username})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Register command failed: %v", err)
	}

	// Step 3: Create a test .env file to encrypt
	envFilePath := filepath.Join(tempDir, "test.env")
	envContent := "DATABASE_URL=postgres://localhost:5432/test\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFilePath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Step 4: Try to encrypt the file
	encryptOutput, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("encrypt", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "encrypt", envFilePath})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Encrypt command failed: %v", err)
		t.Errorf("Encrypt output: %s", encryptOutput)
	}

	// Verify encryption was successful
	encryptedFilePath := envFilePath + ".kanuka"
	if _, err := os.Stat(encryptedFilePath); os.IsNotExist(err) {
		t.Errorf("Encrypted file was not created")
	}

	// Verify the encrypted file contains encrypted data
	encryptedData, err := os.ReadFile(encryptedFilePath)
	if err != nil {
		t.Errorf("Failed to read encrypted file: %v", err)
	} else {
		// Encrypted data should not contain the original plaintext
		if strings.Contains(string(encryptedData), "DATABASE_URL") {
			t.Errorf("Encrypted file appears to contain plaintext data")
		}
		// Should contain some binary data
		if len(encryptedData) == 0 {
			t.Errorf("Encrypted file is empty")
		}
	}

	// Step 5: Test decryption to verify the full workflow
	// Remove original file first
	if err := os.Remove(envFilePath); err != nil {
		t.Errorf("Failed to remove original env file: %v", err)
	}

	decryptOutput, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("decrypt", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "decrypt", encryptedFilePath})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Decrypt command failed: %v", err)
		t.Errorf("Decrypt output: %s", decryptOutput)
	}

	// Verify decryption restored the original file
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		t.Errorf("Decrypted file was not created")
	}

	// Verify content matches original
	decryptedData, err := os.ReadFile(envFilePath)
	if err != nil {
		t.Errorf("Failed to read decrypted file: %v", err)
	} else if string(decryptedData) != envContent {
		t.Errorf("Decrypted content doesn't match original.\nExpected: %s\nGot: %s", envContent, string(decryptedData))
	}
}

// Tests multiple users workflow - test multiple users creating keys in same project.
func testMultipleUsersWorkflow(t *testing.T, originalWd string, originalUserSettings *configs.UserSettings) {
	tempDir, err := os.MkdirTemp("", "kanuka-test-multiuser-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create separate user directories for each user
	tempUserDir1, err := os.MkdirTemp("", "kanuka-user1-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory 1: %v", err)
	}
	defer os.RemoveAll(tempUserDir1)

	tempUserDir2, err := os.MkdirTemp("", "kanuka-user2-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory 2: %v", err)
	}
	defer os.RemoveAll(tempUserDir2)

	// Initialize project once
	shared.SetupTestEnvironment(t, tempDir, tempUserDir1, originalWd, originalUserSettings)
	
	// Initialize project structure only
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// User 1: Create keys
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("User 1 create failed: %v", err)
	}

	user1Name := configs.UserKanukaSettings.Username
	user1PublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", user1Name+".pub")

	// Verify user 1's public key was created
	if _, err := os.Stat(user1PublicKeyPath); os.IsNotExist(err) {
		t.Errorf("User 1 public key was not created")
	}

	// User 2: Setup environment and create keys
	shared.SetupTestEnvironment(t, tempDir, tempUserDir2, originalWd, originalUserSettings)
	
	// Override username for user 2
	configs.UserKanukaSettings.Username = "testuser2"

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("User 2 create failed: %v", err)
	}

	user2Name := configs.UserKanukaSettings.Username
	user2PublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", user2Name+".pub")

	// Verify user 2's public key was created
	if _, err := os.Stat(user2PublicKeyPath); os.IsNotExist(err) {
		t.Errorf("User 2 public key was not created")
	}

	// Verify both users have different keys
	user1KeyData, err := os.ReadFile(user1PublicKeyPath)
	if err != nil {
		t.Errorf("Failed to read user 1 public key: %v", err)
	}

	user2KeyData, err := os.ReadFile(user2PublicKeyPath)
	if err != nil {
		t.Errorf("Failed to read user 2 public key: %v", err)
	}

	if string(user1KeyData) == string(user2KeyData) {
		t.Errorf("User 1 and User 2 have identical public keys (should be different)")
	}

	// Verify project structure contains both users
	publicKeysDir := filepath.Join(tempDir, ".kanuka", "public_keys")
	entries, err := os.ReadDir(publicKeysDir)
	if err != nil {
		t.Errorf("Failed to read public keys directory: %v", err)
	}

	foundUsers := make(map[string]bool)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".pub") {
			username := strings.TrimSuffix(entry.Name(), ".pub")
			foundUsers[username] = true
		}
	}

	if !foundUsers[user1Name] {
		t.Errorf("User 1 (%s) public key not found in project", user1Name)
	}
	if !foundUsers[user2Name] {
		t.Errorf("User 2 (%s) public key not found in project", user2Name)
	}

	// Test that each user can be registered independently
	// Register user 1
	shared.SetupTestEnvironment(t, tempDir, tempUserDir1, originalWd, originalUserSettings)
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", user1Name})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Failed to register user 1: %v", err)
	}

	// Register user 2
	shared.SetupTestEnvironment(t, tempDir, tempUserDir2, originalWd, originalUserSettings)
	configs.UserKanukaSettings.Username = "testuser2"
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", user2Name})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Failed to register user 2: %v", err)
	}

	// Verify both users have .kanuka files
	user1KanukaPath := filepath.Join(tempDir, ".kanuka", "secrets", user1Name+".kanuka")
	user2KanukaPath := filepath.Join(tempDir, ".kanuka", "secrets", user2Name+".kanuka")

	if _, err := os.Stat(user1KanukaPath); os.IsNotExist(err) {
		t.Errorf("User 1 kanuka file was not created")
	}
	if _, err := os.Stat(user2KanukaPath); os.IsNotExist(err) {
		t.Errorf("User 2 kanuka file was not created")
	}

	// Verify kanuka files are different (different encrypted symmetric keys)
	user1KanukaData, err := os.ReadFile(user1KanukaPath)
	if err != nil {
		t.Errorf("Failed to read user 1 kanuka file: %v", err)
	}

	user2KanukaData, err := os.ReadFile(user2KanukaPath)
	if err != nil {
		t.Errorf("Failed to read user 2 kanuka file: %v", err)
	}

	if string(user1KanukaData) == string(user2KanukaData) {
		t.Errorf("User 1 and User 2 have identical kanuka files (should be different)")
	}
}