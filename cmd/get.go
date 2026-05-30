package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/heliofernandes404/tg-alerts/internal/auth"
	"github.com/heliofernandes404/tg-alerts/internal/config"
	"github.com/heliofernandes404/tg-alerts/internal/output"
	"github.com/heliofernandes404/tg-alerts/internal/search"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Fetch a single Telegram message by ID",
	RunE:  runGet,
}

func init() {
	f := getCmd.Flags()
	f.String("channel", "", "Telegram channel username (e.g. @canal-alertas)")
	f.Int("message-id", 0, "Telegram message ID to fetch")
	f.Duration("timeout", 30*time.Second, "Operation timeout")

	rootCmd.AddCommand(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		output.WriteError(os.Stdout, "CONFIG_INVALID", err.Error(), "")
		return nil
	}

	channel, _ := cmd.Flags().GetString("channel")
	if channel == "" {
		channel = cfg.Defaults.Channel
	}
	if channel == "" {
		output.WriteError(os.Stdout, "QUERY_INVALID", "--channel is required", "")
		return nil
	}

	messageID, _ := cmd.Flags().GetInt("message-id")
	if messageID <= 0 {
		output.WriteError(os.Stdout, "QUERY_INVALID",
			"--message-id must be a positive integer", "")
		return nil
	}

	if cfg.Telegram.APIID == 0 {
		output.WriteError(os.Stdout, "CONFIG_INVALID",
			"telegram.api_id is not configured",
			"Create ~/.config/tg-alerts/config.yaml with api_id and api_hash_env.")
		return nil
	}
	apiHash := cfg.APIHash()
	if apiHash == "" {
		msg := "api_hash not configured"
		if cfg.Telegram.APIHashEnv != "" {
			msg = fmt.Sprintf("env var %q not set", cfg.Telegram.APIHashEnv)
		}
		output.WriteError(os.Stdout, "CONFIG_INVALID", msg,
			"Create ~/.config/tg-alerts/config.yaml and set api_hash_env.")
		return nil
	}

	session := auth.NewFileSession(config.SessionPath())
	if !session.Exists() {
		output.WriteError(os.Stdout, "AUTH_REQUIRED", "No local session found.",
			"Run tg-alerts auth login.")
		return nil
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ex := search.NewExecutor(cfg.Telegram.APIID, apiHash, session)
	resp, err := ex.ExecuteGetMessage(ctx, channel, messageID)
	if err != nil {
		code, msg := classifyError(err)
		output.WriteError(os.Stdout, code, msg, "")
		return nil
	}
	return output.WriteJSON(os.Stdout, resp)
}
