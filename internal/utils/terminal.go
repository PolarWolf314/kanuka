package utils

import (
	"fmt"
	"os"
	"runtime"

	"golang.org/x/term"
)

// ReadPassphrase prompts the user for a passphrase without echoing input.
// Returns an error if stdin is not a terminal.
func ReadPassphrase(prompt string) ([]byte, error) {
	fd := int(os.Stdin.Fd())

	if !term.IsTerminal(fd) {
		return nil, fmt.Errorf("cannot read passphrase: stdin is not a terminal")
	}

	fmt.Fprint(os.Stderr, prompt)
	passphrase, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr) // Add newline after hidden input

	if err != nil {
		return nil, fmt.Errorf("failed to read passphrase: %w", err)
	}

	return passphrase, nil
}

// ReadPassphraseFromTTY prompts the user for a passphrase from /dev/tty (or CON on Windows).
// This is useful when stdin is being used for other input (e.g., piping a private key).
// Returns an error if /dev/tty cannot be opened.
func ReadPassphraseFromTTY(prompt string) ([]byte, error) {
	ttyPath := "/dev/tty"
	if runtime.GOOS == "windows" {
		ttyPath = "CON"
	}

	tty, err := os.Open(ttyPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s for passphrase input: %w", ttyPath, err)
	}
	defer tty.Close()

	fd := int(tty.Fd())
	if !term.IsTerminal(fd) {
		return nil, fmt.Errorf("%s is not a terminal", ttyPath)
	}

	fmt.Fprint(os.Stderr, prompt)
	passphrase, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr) // Add newline after hidden input

	if err != nil {
		return nil, fmt.Errorf("failed to read passphrase: %w", err)
	}

	return passphrase, nil
}

// IsTerminal returns true if stdin is a terminal.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// IsTTYAvailable returns true if /dev/tty (or CON on Windows) is available for reading.
func IsTTYAvailable() bool {
	ttyPath := "/dev/tty"
	if runtime.GOOS == "windows" {
		ttyPath = "CON"
	}

	tty, err := os.Open(ttyPath)
	if err != nil {
		return false
	}
	defer tty.Close()

	return term.IsTerminal(int(tty.Fd()))
}

// WriteToTTY writes content directly to the terminal (bypassing stdout/stderr).
// On Unix, writes to /dev/tty. On Windows, writes to CON.
// Returns an error if the TTY cannot be opened.
func WriteToTTY(content string) error {
	ttyPath := "/dev/tty"
	if runtime.GOOS == "windows" {
		ttyPath = "CON"
	}

	tty, err := os.OpenFile(ttyPath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("cannot open %s for writing: %w", ttyPath, err)
	}
	defer tty.Close()

	_, err = tty.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to TTY: %w", err)
	}

	return nil
}

// ClearScreen clears the terminal screen using ANSI escape sequences.
// Writes directly to TTY to ensure it works even when stdout is redirected.
func ClearScreen() error {
	// ANSI escape sequence: clear screen and move cursor to top-left.
	return WriteToTTY("\033[2J\033[H")
}

// WaitForEnterFromTTY waits for the user to press Enter on the TTY.
// This reads from /dev/tty (or CON on Windows) directly.
func WaitForEnterFromTTY() error {
	ttyPath := "/dev/tty"
	if runtime.GOOS == "windows" {
		ttyPath = "CON"
	}

	tty, err := os.Open(ttyPath)
	if err != nil {
		return fmt.Errorf("cannot open %s for reading: %w", ttyPath, err)
	}
	defer tty.Close()

	buf := make([]byte, 1)
	for {
		_, err := tty.Read(buf)
		if err != nil {
			return fmt.Errorf("failed to read from TTY: %w", err)
		}
		if buf[0] == '\n' || buf[0] == '\r' {
			return nil
		}
	}
}
