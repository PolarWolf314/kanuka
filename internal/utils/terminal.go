package utils

import (
	"fmt"
	"os"

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

// IsTerminal returns true if stdin is a terminal.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
