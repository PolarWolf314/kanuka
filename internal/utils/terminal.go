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
