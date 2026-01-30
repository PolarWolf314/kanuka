// Package logger provides structured logging for Kanuka CLI commands.
//
// The logger supports multiple verbosity levels controlled by command-line
// flags. Output is formatted with semantic prefixes and colors from the
// ui package.
//
// # Verbosity Levels
//
// Logging behavior is controlled by two flags:
//
//   - --verbose: Shows info and warning messages
//   - --debug: Shows all messages including debug details
//
// Without flags, only critical warnings and errors are shown.
//
// # Log Methods
//
//	Logger.Infof()       // Shown with --verbose or --debug
//	Logger.Debugf()      // Shown only with --debug
//	Logger.Warnf()       // Shown with --verbose or --debug
//	Logger.WarnfAlways() // Always shown (critical warnings)
//	Logger.WarnfUser()   // User-facing warnings (not debug info)
//	Logger.Errorf()      // Shown with --debug
//	Logger.Fatalf()      // Always shown, then exits
//
// # Usage
//
// Create a logger with the desired verbosity:
//
//	log := Logger{Verbose: verbose, Debug: debug}
//	log.Infof("Processing %d files", count)
//
// Commands typically create a logger in their PersistentPreRun and
// pass it to internal functions.
package logger
