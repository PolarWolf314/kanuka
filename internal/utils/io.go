package utils

import (
	"fmt"
	"io"
	"os"
)

// ReadStdin reads all content from stdin.
// Returns an error if stdin is empty, is a terminal (no piped data), or cannot be read.
func ReadStdin() ([]byte, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat stdin: %w", err)
	}

	// Check if stdin is a terminal (no piped data).
	// If ModeCharDevice is set, stdin is connected to a terminal.
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, fmt.Errorf("no data provided on stdin (hint: pipe your private key to this command)")
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("stdin is empty")
	}

	return data, nil
}
