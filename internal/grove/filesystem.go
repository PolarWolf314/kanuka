package grove

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

// DoesKanukaTomlExist checks if kanuka.toml exists in the current directory.
func DoesKanukaTomlExist() (bool, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("failed to get current directory: %w", err)
	}

	kanukaTomlPath := filepath.Join(currentDir, "kanuka.toml")
	_, err = os.Stat(kanukaTomlPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("error checking kanuka.toml: %w", err)
}

// DoesDevenvNixExist checks if devenv.nix exists in the current directory.
func DoesDevenvNixExist() (bool, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvNixPath := filepath.Join(currentDir, "devenv.nix")
	_, err = os.Stat(devenvNixPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("error checking devenv.nix: %w", err)
}

// CreateKanukaToml creates a new kanuka.toml file in the current directory.
func CreateKanukaToml() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Generate a project ID
	projectID, err := generateProjectID()
	if err != nil {
		return fmt.Errorf("failed to generate project ID: %w", err)
	}

	// Get project name from directory
	projectName := filepath.Base(currentDir)

	kanukaTomlContent := fmt.Sprintf(`[project]
id = "%s"
name = "%s"

[grove]
# Grove-specific configuration
`, projectID, projectName)

	kanukaTomlPath := filepath.Join(currentDir, "kanuka.toml")
	err = os.WriteFile(kanukaTomlPath, []byte(kanukaTomlContent), 0600)
	if err != nil {
		return fmt.Errorf("failed to write kanuka.toml: %w", err)
	}

	return nil
}

// CreateDevenvYaml creates a new devenv.yaml file in the current directory.
func CreateDevenvYaml() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get the latest stable channel programmatically
	latestStable := GetLatestStableChannel()

	devenvYamlContent := fmt.Sprintf(`inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixpkgs-unstable
  nixpkgs-stable:
    url: github:NixOS/nixpkgs/%s

allowUnfree: true
`, latestStable)

	devenvYamlPath := filepath.Join(currentDir, "devenv.yaml")
	err = os.WriteFile(devenvYamlPath, []byte(devenvYamlContent), 0600)
	if err != nil {
		return fmt.Errorf("failed to write devenv.yaml: %w", err)
	}

	return nil
}

// DoesDevenvYamlExist checks if devenv.yaml exists in the current directory.
func DoesDevenvYamlExist() (bool, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvYamlPath := filepath.Join(currentDir, "devenv.yaml")
	_, err = os.Stat(devenvYamlPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("error checking devenv.yaml: %w", err)
}

// CreateDevenvNix creates a new devenv.nix file in the current directory.
func CreateDevenvNix() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvNixContent := `{ pkgs, inputs, ... }: {
  packages = [
    # Add your packages here
    pkgs.git

    # Kanuka-managed packages - DO NOT EDIT MANUALLY
    # End Kanuka-managed packages
  ];

  # Kanuka Grove environment configuration
  env = {
    # Add your custom environment variables here
  };

  # Enable dotenv integration
  dotenv.enable = true;

  enterShell = ''
    # ============== EDIT ABOVE THIS LINE ==============
    # Kanuka-managed shell configuration - DO NOT MODIFY
    export TERM="xterm-256color"
    
    # Set custom Kanuka prompt
    # (Kanuka) in green at start, path in blue, $ in blue
    export PS1='\[\033[32m\](Kanuka)\[\033[00m\] \[\033[01;34m\]\w\[\033[00m\] $ '
    
    echo "Welcome to your development environment!"
    echo "Managed by Kānuka Grove"
    # ============== EDIT BELOW THIS LINE ==============
  '';
}
`

	devenvNixPath := filepath.Join(currentDir, "devenv.nix")
	err = os.WriteFile(devenvNixPath, []byte(devenvNixContent), 0600)
	if err != nil {
		return fmt.Errorf("failed to write devenv.nix: %w", err)
	}

	return nil
}

// generateProjectID generates a random project ID.
func generateProjectID() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hex string
	return fmt.Sprintf("%x", bytes), nil
}
