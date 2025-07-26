package grove

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// Get project name from directory
	projectName := filepath.Base(currentDir)

	devenvNixContent := fmt.Sprintf(`{ pkgs, inputs, ... }:
let
  # Import additional nixpkgs channels for multi-channel support
  pkgs-stable = import inputs.nixpkgs-stable { system = pkgs.stdenv.system; };
in
{
  name = "%s";
  
  packages = [
    # Sensible defaults - DO NOT MODIFY unless you know what you're doing
    # These packages provide essential shell functionality and development tools
    # Remove any of these at your own risk - basic commands may stop working
    
    # Essential core utilities (Tier 1)
    pkgs.coreutils     # ls, cp, mv, rm, cat, echo, mkdir, etc.
    pkgs.util-linux    # mount, umount, lsblk, etc.
    pkgs.findutils     # find, xargs, locate
    pkgs.which         # locate commands in PATH
    pkgs.ncurses       # clear, tput, terminal handling
    
    # File and text processing (Tier 2)
    pkgs.file          # determine file types
    pkgs.tree          # directory structure display
    pkgs.less          # file pager
    pkgs.gnugrep       # text search
    pkgs.gnused        # stream editor
    
    # Network and downloads (Tier 2)
    pkgs.curl          # HTTP client
    pkgs.wget          # file downloader
    
    # Development tools
    pkgs.git           # version control
    
    # Add your custom packages below this line
    # Example: pkgs.nodejs_18, pkgs-stable.python3, etc.

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
`, projectName)

	devenvNixPath := filepath.Join(currentDir, "devenv.nix")
	err = os.WriteFile(devenvNixPath, []byte(devenvNixContent), 0600)
	if err != nil {
		return fmt.Errorf("failed to write devenv.nix: %w", err)
	}

	return nil
}

// AddNix2ContainerInput adds the nix2container input to devenv.yaml
// This is required for container functionality as per devenv.sh documentation.
func AddNix2ContainerInput() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	devenvYamlPath := filepath.Join(currentDir, "devenv.yaml")

	// Check if devenv.yaml exists
	if _, err := os.Stat(devenvYamlPath); os.IsNotExist(err) {
		return fmt.Errorf("devenv.yaml not found - run 'kanuka grove init' first")
	}

	// Read current devenv.yaml content
	content, err := os.ReadFile(devenvYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read devenv.yaml: %w", err)
	}

	contentStr := string(content)

	// Check if nix2container input already exists
	if strings.Contains(contentStr, "nix2container:") {
		return nil // Already exists, nothing to do
	}

	// Replace the entire devenv.yaml with the correct structure
	// Get the latest stable channel programmatically
	latestStable := GetLatestStableChannel()

	newDevenvYamlContent := fmt.Sprintf(`inputs:
  mk-shell-bin:
    url: github:rrbutani/nix-mk-shell-bin
  # Required for container support - DO NOT MODIFY
  nix2container:
    url: github:nlewo/nix2container
    inputs:
      nixpkgs:
        follows: nixpkgs
  nixpkgs:
    url: github:NixOS/nixpkgs/nixpkgs-unstable
  nixpkgs-stable:
    url: github:NixOS/nixpkgs/%s

allowUnfree: true
backend: nix
`, latestStable)

	// Write the new content
	if err := os.WriteFile(devenvYamlPath, []byte(newDevenvYamlContent), 0600); err != nil {
		return fmt.Errorf("failed to write modified devenv.yaml: %w", err)
	}

	return nil
}

// CreateOrUpdateGitignore creates or updates .gitignore with Grove-specific entries.
func CreateOrUpdateGitignore() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	gitignorePath := filepath.Join(currentDir, ".gitignore")

	// Grove-specific gitignore entries
	groveEntries := `
# Kanuka Grove - devenv cache and build artifacts
.devenv/
.devenv.flake.nix
result
result-*
`

	// Check if .gitignore exists
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		// Create new .gitignore with Grove entries
		gitignoreContent := `# Kanuka Grove .gitignore
# Generated by 'kanuka grove init'
` + groveEntries

		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0600); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
		return nil
	}

	// .gitignore exists, check if it already has Grove entries
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return fmt.Errorf("failed to read existing .gitignore: %w", err)
	}

	contentStr := string(content)

	// Check if Grove entries already exist
	if strings.Contains(contentStr, "# Kanuka Grove") || strings.Contains(contentStr, ".devenv/") {
		return nil // Grove entries already present
	}

	// Append Grove entries to existing .gitignore
	updatedContent := contentStr
	if !strings.HasSuffix(contentStr, "\n") {
		updatedContent += "\n"
	}
	updatedContent += groveEntries

	if err := os.WriteFile(gitignorePath, []byte(updatedContent), 0600); err != nil {
		return fmt.Errorf("failed to update .gitignore: %w", err)
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
