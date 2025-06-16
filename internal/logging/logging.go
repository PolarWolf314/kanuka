package logger

import (
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
)

type Logger struct {
	Verbose bool
	Debug   bool
}

func (l Logger) Infof(msg string, args ...any) {
	if l.Verbose || l.Debug {
		fmt.Fprintf(os.Stdout, color.GreenString("[info] ")+msg+"\n", args...)
	}
}

func (l Logger) Debugf(msg string, args ...any) {
	if l.Debug {
		fmt.Fprintf(os.Stdout, color.CyanString("[debug] ")+msg+"\n", args...)
	}
}

func (l Logger) Warnf(msg string, args ...any) {
	// Show in verbose or debug mode
	if l.Verbose || l.Debug {
		fmt.Fprintf(os.Stderr, color.YellowString("[warn] ")+msg+"\n", args...)
	}
}

func (l Logger) WarnfAlways(msg string, args ...any) {
	// Always show critical warnings
	fmt.Fprintf(os.Stderr, color.YellowString("⚠️  ")+msg+"\n", args...)
}

func (l Logger) WarnfUser(msg string, args ...any) {
	// Show user-facing warnings (not just debug info)
	if !l.Debug { // Don't duplicate with debug logs
		fmt.Fprintf(os.Stderr, color.YellowString("Warning: ")+msg+"\n", args...)
	} else {
		fmt.Fprintf(os.Stderr, color.YellowString("[warn] ")+msg+"\n", args...)
	}
}

func (l Logger) Errorf(msg string, args ...any) {
	if l.Debug {
		fmt.Fprintf(os.Stderr, color.RedString("[error] ")+msg+"\n", args...)
	}
}

func (l Logger) Fatalf(msg string, args ...any) {
	// First log the error using our custom error logging
	l.Errorf(msg, args...)

	// Set log output to stdout if not in verbose or debug mode
	if !l.Verbose && !l.Debug {
		log.SetOutput(os.Stdout)
	}

	// Print fatal error and exit
	log.Fatalf("❌ "+msg, args...)
}

func (l Logger) ErrorfAndReturn(msg string, args ...any) error {
	// Log the error using our custom error logging
	l.Errorf(msg, args...)

	// Print error message without exiting
	if !l.Verbose && !l.Debug {
		fmt.Fprintf(os.Stdout, "❌ "+msg+"\n", args...)
	}

	// Return the error for the caller to handle
	return fmt.Errorf(msg, args...)
}
