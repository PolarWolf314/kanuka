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
	// Create two separate user directories to simulate two different users
	tempDir, err := os.MkdirTemp("", "kanuka-test-register-workflow-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// User 1 (admin) - will initialize project and grant access
	tempUserDir1, err := os.MkdirTemp("", "kanuka-user1-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory 1: %v", err)
	}
	defer os.RemoveAll(tempUserDir1)

	// User 2 (new user) - will create keys and request access
	tempUserDir2, err := os.MkdirTemp("", "kanuka-user2-*")
	if err != nil {
		t.Fatalf("Failed to create temp user directory 2: %v", err)
	}
	defer os.RemoveAll(tempUserDir2)

	// Step 1: User 1 initializes the project (gets admin access)
	shared.SetupTestEnvironmentWithUUID(t, tempDir, tempUserDir1, originalWd, originalUserSettings,
		shared.TestUserUUID, "testuser1", "testuser1@example.com")

	initOutput, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Verify init was successful by checking if kanuka file exists
	// (init command may only show warnings in test output)
	t.Logf("Init output: %s", initOutput)

	user1UUID := shared.TestUserUUID
	user1KanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", user1UUID+".kanuka")
	if _, err := os.Stat(user1KanukaFile); os.IsNotExist(err) {
		t.Fatalf("User 1 kanuka file was not created by init command")
	}

	// Step 2: User 2 creates keys (joins project)
	// Use a different UUID for user 2
	shared.SetupTestEnvironmentWithUUID(t, tempDir, tempUserDir2, originalWd, originalUserSettings,
		shared.TestUser2UUID, "testuser2", "testuser2@example.com")

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

	user2UUID := shared.TestUser2UUID
	user2PublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", user2UUID+".pub")

	// Verify user 2's public key was created
	if _, err := os.Stat(user2PublicKeyPath); os.IsNotExist(err) {
		t.Fatalf("User 2 public key was not created by create command")
	}

	// Step 3: User 1 (admin) grants access to User 2
	shared.SetupTestEnvironmentWithUUID(t, tempDir, tempUserDir1, originalWd, originalUserSettings,
		shared.TestUserUUID, "testuser1", "testuser1@example.com")

	registerOutput, err := shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", user2UUID})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Register command failed: %v", err)
		t.Errorf("Register output: %s", registerOutput)
	}

	// Verify register was successful
	if !strings.Contains(registerOutput, "✓") {
		t.Errorf("Register command didn't show success: %s", registerOutput)
	}

	// Verify user 2's .kanuka file was created
	user2KanukaFile := filepath.Join(tempDir, ".kanuka", "secrets", user2UUID+".kanuka")
	if _, err := os.Stat(user2KanukaFile); os.IsNotExist(err) {
		t.Errorf("User 2 kanuka file was not created by register command")
	}

	// Verify the workflow instructions were shown in create output
	if !strings.Contains(createOutput, "kanuka secrets register") {
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

	// Step 1: Initialize project (this gives the user access automatically)
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	// Verify user has access after init
	userUUID := shared.GetUserUUID(t)
	kanukaFilePath := filepath.Join(tempDir, ".kanuka", "secrets", userUUID+".kanuka")
	if _, err := os.Stat(kanukaFilePath); os.IsNotExist(err) {
		t.Fatalf("Kanuka file was not created by init command")
	}

	// Step 2: Create a test .env file to encrypt
	envFilePath := filepath.Join(tempDir, "test.env")
	envContent := "DATABASE_URL=postgres://localhost:5432/test\nAPI_KEY=secret123\n"
	if err := os.WriteFile(envFilePath, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Step 3: Try to encrypt the file
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

	// Step 4: Test decryption to verify the full workflow
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

	// User 1: Initialize project (gets admin access automatically)
	shared.SetupTestEnvironmentWithUUID(t, tempDir, tempUserDir1, originalWd, originalUserSettings,
		shared.TestUserUUID, "testuser1", "testuser1@example.com")

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("init", nil, nil, false, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	user1UUID := shared.TestUserUUID
	user1PublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", user1UUID+".pub")

	// Verify user 1's public key was created
	if _, err := os.Stat(user1PublicKeyPath); os.IsNotExist(err) {
		t.Errorf("User 1 public key was not created")
	}

	// User 2: Setup environment with a DIFFERENT UUID and create keys
	shared.SetupTestEnvironmentWithUUID(t, tempDir, tempUserDir2, originalWd, originalUserSettings,
		shared.TestUser2UUID, "testuser2", "testuser2@example.com")

	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("create", nil, nil, true, false)
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("User 2 create failed: %v", err)
	}

	user2UUID := shared.TestUser2UUID
	user2PublicKeyPath := filepath.Join(tempDir, ".kanuka", "public_keys", user2UUID+".pub")

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
			userUUID := strings.TrimSuffix(entry.Name(), ".pub")
			foundUsers[userUUID] = true
		}
	}

	if !foundUsers[user1UUID] {
		t.Errorf("User 1 (%s) public key not found in project", user1UUID)
	}
	if !foundUsers[user2UUID] {
		t.Errorf("User 2 (%s) public key not found in project", user2UUID)
	}

	// Test that user 1 (admin) can register user 2
	// User 1 already has access from init, so they can register user 2
	shared.SetupTestEnvironmentWithUUID(t, tempDir, tempUserDir1, originalWd, originalUserSettings,
		shared.TestUserUUID, "testuser1", "testuser1@example.com")
	_, err = shared.CaptureOutput(func() error {
		cmd := shared.CreateTestCLI("register", nil, nil, true, false)
		cmd.SetArgs([]string{"secrets", "register", "--user", user2UUID})
		return cmd.Execute()
	})
	if err != nil {
		t.Errorf("Failed to register user 2: %v", err)
	}

	// Verify both users have .kanuka files
	user1KanukaPath := filepath.Join(tempDir, ".kanuka", "secrets", user1UUID+".kanuka")
	user2KanukaPath := filepath.Join(tempDir, ".kanuka", "secrets", user2UUID+".kanuka")

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
