package cmd

import (
	"log"
	"os"
)

func verboseLog(message string) {
	if verbose {
		log.Println(message)
	}
}

func printError(message string, err error) {
	if !verbose {
		log.SetOutput(os.Stdout)
	}
	log.Fatalf("❌ %s: %v", message, err)
}
