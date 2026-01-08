package configs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadTOML(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.toml")

	type TestStruct struct {
		Name  string
		Age   int
		Email string
	}

	originalData := TestStruct{
		Name:  "John Doe",
		Age:   30,
		Email: "john@example.com",
	}

	err := SaveTOML(testFile, originalData)
	if err != nil {
		t.Fatalf("SaveTOML failed: %v", err)
	}

	loadedData := TestStruct{}
	err = LoadTOML(testFile, &loadedData)
	if err != nil {
		t.Fatalf("LoadTOML failed: %v", err)
	}

	if loadedData.Name != originalData.Name {
		t.Errorf("Expected Name %q, got %q", originalData.Name, loadedData.Name)
	}

	if loadedData.Age != originalData.Age {
		t.Errorf("Expected Age %d, got %d", originalData.Age, loadedData.Age)
	}

	if loadedData.Email != originalData.Email {
		t.Errorf("Expected Email %q, got %q", originalData.Email, loadedData.Email)
	}
}

func TestLoadTOMLNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "nonexistent.toml")

	type TestStruct struct {
		Name string
	}

	data := TestStruct{}
	err := LoadTOML(testFile, &data)
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
}

func TestSaveTOMLCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "subdir", "test.toml")

	type TestStruct struct {
		Name string
	}

	data := TestStruct{Name: "Test"}
	err := SaveTOML(testFile, data)
	if err != nil {
		t.Fatalf("SaveTOML failed: %v", err)
	}

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}
}
