package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/heliofernandes404/tg-alerts/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tg-alerts",
	Short: "Query Telegram alert channels",
	Long:  "tg-alerts queries Telegram alert channels and returns JSON results for LLMs and automation agents.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version.Version
	rootCmd.SetVersionTemplate(versionJSON())
}

func versionJSON() string {
	b, err := json.Marshal(version.Current())
	if err != nil {
		return fmt.Sprintf("{\"version\":%q,\"commit\":%q,\"date\":%q}\n", version.Version, version.Commit, version.Date)
	}
	return string(b) + "\n"
}
