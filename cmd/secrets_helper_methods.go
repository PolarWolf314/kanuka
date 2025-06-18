package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/briandowns/spinner"
)

// startSpinner creates and starts a spinner with the given message when not in verbose or debug mode.
// Returns the spinner and a function that should be deferred to clean up.
func startSpinner(message string, verbose bool) (*spinner.Spinner, func()) {
	Logger.Debugf("Starting spinner with message: %s", message)
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	err := s.Color("cyan")
	if err != nil {
		// If we can't set spinner color, just continue without it
		Logger.Warnf("Failed to set spinner color: %v", err)
	}

	if !verbose && !debug {
		Logger.Debugf("Starting spinner in non-verbose mode")
		s.Start()
		// Ensure log output is discarded unless in verbose mode
		log.SetOutput(io.Discard)
	} else {
		Logger.Infof("Running in verbose or debug mode: %s", message)
	}

	cleanup := func() {
		if !verbose && !debug {
			Logger.Debugf("Stopping spinner and restoring log output")
			log.SetOutput(os.Stdout)
			s.Stop()
		}
		// Always print the final message if it's set, regardless of verbose mode
		// This ensures the message is captured by tests
		if s.FinalMSG != "" {
			Logger.Debugf("Displaying final message")
			fmt.Print(s.FinalMSG)
		}
	}

	return s, cleanup
}
