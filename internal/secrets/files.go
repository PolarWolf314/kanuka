package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// ResolveFiles takes user-provided paths/globs and returns matching files.
// If patterns is empty, returns nil (caller should use default behavior).
// forEncryption=true finds .env* files, forEncryption=false finds *.kanuka files.
func ResolveFiles(patterns []string, projectPath string, forEncryption bool) ([]string, error) {
	if len(patterns) == 0 {
		// No patterns provided, caller should use default behavior.
		return nil, nil
	}

	var files []string
	seen := make(map[string]bool) // Deduplicate.

	for _, pattern := range patterns {
		resolved, err := resolvePattern(pattern, projectPath, forEncryption)
		if err != nil {
			return nil, err
		}

		for _, f := range resolved {
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no matching files found")
	}

	return files, nil
}

func resolvePattern(pattern string, projectPath string, forEncryption bool) ([]string, error) {
	// Convert relative patterns to absolute paths based on project path.
	absPattern := pattern
	if !filepath.IsAbs(pattern) {
		absPattern = filepath.Join(projectPath, pattern)
	}

	// Check if it's a directory.
	info, err := os.Stat(absPattern)
	if err == nil && info.IsDir() {
		return findFilesInDir(absPattern, forEncryption)
	}

	// Check if it contains glob characters.
	if strings.ContainsAny(pattern, "*?[") {
		return expandGlob(pattern, projectPath, forEncryption)
	}

	// Treat as literal file path.
	if _, err := os.Stat(absPattern); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", pattern)
	}

	// Validate that the file matches the expected type.
	if forEncryption && !isEnvFile(absPattern) {
		return nil, fmt.Errorf("file is not a .env file: %s", pattern)
	}
	if !forEncryption && !isKanukaFile(absPattern) {
		return nil, fmt.Errorf("file is not a .kanuka file: %s", pattern)
	}

	return []string{absPattern}, nil
}

func expandGlob(pattern string, projectPath string, forEncryption bool) ([]string, error) {
	// Use doublestar for ** support.
	// We need to use the fsys version with os.DirFS for proper ** handling.
	absPattern := pattern
	if !filepath.IsAbs(pattern) {
		absPattern = filepath.Join(projectPath, pattern)
	}

	matches, err := doublestar.FilepathGlob(absPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}

	// Filter to only include appropriate file types.
	var filtered []string
	for _, m := range matches {
		// Skip directories.
		info, err := os.Stat(m)
		if err != nil || info.IsDir() {
			continue
		}

		// Skip files inside .kanuka directory.
		if isInKanukaDir(m) {
			continue
		}

		if forEncryption && isEnvFile(m) {
			filtered = append(filtered, m)
		} else if !forEncryption && isKanukaFile(m) {
			filtered = append(filtered, m)
		}
	}

	return filtered, nil
}

func findFilesInDir(dir string, forEncryption bool) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip .kanuka directory.
			if d.Name() == ".kanuka" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip irregular files.
		if !d.Type().IsRegular() {
			return nil
		}

		if forEncryption && isEnvFile(path) {
			files = append(files, path)
		} else if !forEncryption && isKanukaFile(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func isEnvFile(path string) bool {
	base := filepath.Base(path)
	return strings.Contains(base, ".env") && !strings.HasSuffix(base, ".kanuka")
}

func isKanukaFile(path string) bool {
	base := filepath.Base(path)
	return strings.Contains(base, ".env") && strings.HasSuffix(base, ".kanuka")
}

func isInKanukaDir(path string) bool {
	// Check if any component of the path is .kanuka.
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if part == ".kanuka" {
			return true
		}
	}
	return false
}
