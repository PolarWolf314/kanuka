package cmd

import (
	"github.com/spf13/cobra"
)

var (
	verbose bool

	SecretsCmd = &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets stored in the repository",
		Long:  `Provides encryption, decryption, registration, removal, initialization, and purging of secrets.`,
	}
)

func init() {
	SecretsCmd.AddCommand(encryptCmd)
	SecretsCmd.AddCommand(decryptCmd)
	SecretsCmd.AddCommand(createCmd)
	SecretsCmd.AddCommand(registerCmd)
	SecretsCmd.AddCommand(removeCmd)
	SecretsCmd.AddCommand(initCmd)
	SecretsCmd.AddCommand(purgeCmd)
}
