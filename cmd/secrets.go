package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var verbose bool

var SecretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secrets stored in the repository",
	Long:  `Provides encryption, decryption, addition, removal, initialization, and purging of secrets.`,
}

func init() {
	SecretsCmd.AddCommand(encryptCmd)
	SecretsCmd.AddCommand(decryptCmd)
	SecretsCmd.AddCommand(createCmd)
	SecretsCmd.AddCommand(addCmd)
	SecretsCmd.AddCommand(removeCmd)
	SecretsCmd.AddCommand(initCmd)
	SecretsCmd.AddCommand(purgeCmd)
}

func verboseLog(message string) {
	if verbose {
		log.Println(message)
	}
}
