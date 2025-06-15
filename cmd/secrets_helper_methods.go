package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/briandowns/spinner"
)

func printError(message string, err error) {
	Logger.Errorf("%s: %v", message, err)
	if !verbose && !debug {
		log.SetOutput(os.Stdout)
	}
	log.Fatalf("❌ %s: %v", message, err)
}

// startSpinner creates and starts a spinner with the given message when not in verbose or debug mode.
// Returns the spinner and a function that should be deferred to clean up.
func startSpinner(message string, verbose bool) (*spinner.Spinner, func()) {
	Logger.Debugf("Starting spinner with message: %s", message)
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	err := s.Color("cyan")
	if err != nil {
		Logger.Errorf("Failed to create a spinner: %v", err)
		printError("Failed to create a spinner", err)
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
		} else {
			// In verbose or debug mode, manually print the final message if it's set
			if s.FinalMSG != "" {
				Logger.Debugf("Displaying final message in verbose/debug mode")
				fmt.Print(s.FinalMSG)
			}
		}
	}

	return s, cleanup
}
