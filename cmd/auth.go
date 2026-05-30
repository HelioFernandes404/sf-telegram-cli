package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/heliofernandes404/tg-alerts/internal/auth"
	"github.com/heliofernandes404/tg-alerts/internal/config"
	"github.com/heliofernandes404/tg-alerts/internal/output"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Telegram authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate your Telegram account locally",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			output.WriteError(os.Stdout, "CONFIG_INVALID", err.Error(), "")
			return nil
		}
		apiHash := cfg.APIHash()
		if apiHash == "" {
			output.WriteError(os.Stdout, "CONFIG_INVALID",
				fmt.Sprintf("environment variable %q is not set", cfg.Telegram.APIHashEnv),
				"Set the variable and try again.")
			return nil
		}
		phone, _ := cmd.Flags().GetString("phone")
		svc := auth.NewService(auth.ServiceConfig{
			SessionPath: config.SessionPath(),
			APIID:       cfg.Telegram.APIID,
			APIHash:     apiHash,
			Phone:       phone,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := svc.Login(ctx); err != nil {
			output.WriteError(os.Stdout, "AUTH_REQUIRED", err.Error(), "")
			return nil
		}
		fmt.Fprintln(os.Stderr, "Login successful.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report local session status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		sessionPath := config.SessionPath()
		svc := auth.NewService(auth.ServiceConfig{
			SessionPath: sessionPath,
			APIID:       cfg.Telegram.APIID,
			APIHash:     cfg.APIHash(),
		})
		local := svc.LocalStatus()
		resp := output.AuthStatusResponse{
			OK:            true,
			Authenticated: local.Authenticated,
			Session: output.AuthStatusSession{
				Exists: local.Session.Exists,
				Path:   local.Session.Path,
			},
		}
		if local.Authenticated && cfg.Telegram.APIID != 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			username, phone, err := svc.GetAccountInfo(ctx)
			if err == nil {
				resp.Account = &output.AuthStatusAccount{
					PhoneRedacted: phone,
					Username:      username,
				}
			}
		}
		return output.WriteJSON(os.Stdout, resp)
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove the local Telegram session",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := auth.NewService(auth.ServiceConfig{
			SessionPath: config.SessionPath(),
		})
		if err := svc.Logout(); err != nil {
			output.WriteError(os.Stdout, "INTERNAL", err.Error(), "")
			return nil
		}
		fmt.Fprintln(os.Stderr, "Logged out. Session file removed.")
		return nil
	},
}

func init() {
	authLoginCmd.Flags().String("phone", "", "Phone number (e.g. +5511999999999); skips interactive prompt")
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
	rootCmd.AddCommand(authCmd)
}
