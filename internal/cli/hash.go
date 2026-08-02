package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

// NewHashCmd returns the hashsecret subcommand, which prints a bcrypt hash
// suitable for clients.json. The operator pastes the output into a
// client_secret_hash field.
func NewHashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hashsecret <secret>",
		Short: "Print a bcrypt hash of a client secret for clients.json",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hash, err := bcrypt.GenerateFromPassword([]byte(args[0]), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("hash secret: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(hash))
			return nil
		},
	}
}
