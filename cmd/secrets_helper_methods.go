package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
)

// startSpinner creates and starts a spinner with the given message when not in verbose or debug mode.
// Returns the spinner and a function that should be deferred to clean up.
// Uses the global debug flag from the secrets command.
func startSpinner(message string, verbose bool) (*spinner.Spinner, func()) {
	Logger.Debugf("Starting spinner with message: %s", message)
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message

	err := s.Color("cyan")
	if err != nil {
		// If we can't set spinner color, just continue without it.
		Logger.Warnf("Failed to set spinner color: %v", err)
	}

	if !verbose && !debug {
		Logger.Debugf("Starting spinner in non-verbose mode")
		s.Start()
		// Ensure log output is discarded unless in verbose mode.
		log.SetOutput(io.Discard)
	} else {
		Logger.Infof("Running in verbose or debug mode: %s", message)
	}

	cleanup := func() {
		// Restore log output first.
		if !verbose && !debug {
			Logger.Debugf("Restoring log output")
			log.SetOutput(os.Stdout)
		}

		// Ensure the final message ends with a newline.
		if s.FinalMSG != "" && !strings.HasSuffix(s.FinalMSG, "\n") {
			s.FinalMSG += "\n"
		}

		if !verbose && !debug {
			// Stop the spinner if it was started.
			Logger.Debugf("Stopping spinner")
			s.Stop()
		} else if s.FinalMSG != "" {
			// In verbose/debug mode, the spinner doesn't run, so we need to print the message manually.
			Logger.Debugf("Displaying final message in verbose/debug mode")
			fmt.Print(s.FinalMSG)
		}
	}

	return s, cleanup
}

// startSpinnerWithFlags creates and starts a spinner with explicit verbose and debug flags.
// This is useful for commands that have their own flag variables (e.g., config commands).
func startSpinnerWithFlags(message string, verbose, debugFlag bool) (*spinner.Spinner, func()) {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message

	// Ignore color errors - continue without colored spinner if it fails.
	_ = s.Color("cyan")

	if !verbose && !debugFlag {
		s.Start()
		// Ensure log output is discarded unless in verbose mode.
		log.SetOutput(io.Discard)
	}

	cleanup := func() {
		// Restore log output first.
		if !verbose && !debugFlag {
			log.SetOutput(os.Stdout)
		}

		// Ensure the final message ends with a newline.
		if s.FinalMSG != "" && !strings.HasSuffix(s.FinalMSG, "\n") {
			s.FinalMSG += "\n"
		}

		if !verbose && !debugFlag {
			// Stop the spinner if it was started.
			s.Stop()
		} else if s.FinalMSG != "" {
			// In verbose/debug mode, the spinner doesn't run, so we need to print the message manually.
			fmt.Print(s.FinalMSG)
		}
	}

	return s, cleanup
}
