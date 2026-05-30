package cmd

import (
	"fmt"
	"os"

	"github.com/heliofernandes404/tg-alerts/internal/config"
	"github.com/heliofernandes404/tg-alerts/internal/output"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local tg-alerts configuration",
}

var configSetHashCmd = &cobra.Command{
	Use:   "set-hash <api_hash>",
	Short: "Save api_hash to local secrets file (avoids export env var)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hash := args[0]
		if len(hash) < 8 {
			output.WriteError(os.Stdout, "QUERY_INVALID", "api_hash too short", "")
			return nil
		}
		if err := config.SaveAPIHash(hash); err != nil {
			output.WriteError(os.Stdout, "CONFIG_INVALID", err.Error(), "")
			return nil
		}
		fmt.Fprintf(os.Stderr, "api_hash saved to %s (mode 0600)\n", config.APIHashPath())
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetHashCmd)
	rootCmd.AddCommand(configCmd)
}
