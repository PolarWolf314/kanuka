package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/PolarWolf314/kanuka/internal/ui"
	"github.com/briandowns/spinner"
)

// startSpinner creates and starts a spinner with the given message when not in verbose or debug mode.
// Returns the spinner and a function that should be deferred to clean up.
// Uses the global debug flag from the secrets command.
//
// IMPORTANT: spinner.FinalMSG values do NOT need trailing newlines. The cleanup function
// automatically calls ui.EnsureNewline() on the final message before printing it.
// This ensures consistent output formatting across all commands.
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

		// Ensure final message ends with a newline.
		finalMsg := ""
		if s.FinalMSG != "" {
			finalMsg = ui.EnsureNewline(s.FinalMSG)
			// Clear FinalMSG so s.Stop() doesn't print it.
			s.FinalMSG = ""
		}

		// Stop the spinner first to clear the spinner line.
		if !verbose && !debug {
			Logger.Debugf("Stopping spinner")
			s.Stop()
		}

		// Print final message to stdout (for tests to capture).
		if finalMsg != "" {
			fmt.Print(finalMsg)
		}
	}

	return s, cleanup
}

// startSpinnerWithFlags creates and starts a spinner with explicit verbose and debug flags.
// This is useful for commands that have their own flag variables (e.g., config commands).
//
// IMPORTANT: spinner.FinalMSG values do NOT need trailing newlines. The cleanup function
// automatically calls ui.EnsureNewline() on the final message before printing it.
// This ensures consistent output formatting across all commands.
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

		// Ensure final message ends with a newline.
		finalMsg := ""
		if s.FinalMSG != "" {
			finalMsg = ui.EnsureNewline(s.FinalMSG)
			// Clear FinalMSG so s.Stop() doesn't print it.
			s.FinalMSG = ""
		}

		// Stop the spinner first to clear the spinner line.
		if !verbose && !debugFlag {
			s.Stop()
		}

		// Print final message to stdout (for tests to capture).
		if finalMsg != "" {
			fmt.Print(finalMsg)
		}
	}

	return s, cleanup
}
