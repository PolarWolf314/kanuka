package cmd

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/briandowns/spinner"
)

func printError(message string, err error) {
	if !verbose {
		log.SetOutput(os.Stdout)
	}
	log.Fatalf("❌ %s: %v", message, err)
}

// startSpinner creates and starts a spinner with the given message when not in verbose mode.
// Returns the spinner and a function that should be deferred to clean up.
func startSpinner(message string, verbose bool) (*spinner.Spinner, func()) {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	err := s.Color("cyan")
	if err != nil {
		printError("Failed to create a spinner", err)
	}

	if !verbose {
		s.Start()
		// Ensure log output is discarded unless in verbose mode
		log.SetOutput(io.Discard)
	}

	cleanup := func() {
		if !verbose {
			log.SetOutput(os.Stdout)
			s.Stop()
		}
	}

	return s, cleanup
}
