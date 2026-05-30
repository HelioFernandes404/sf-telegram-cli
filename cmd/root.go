package cmd

import (
	"fmt"
	"os"

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
