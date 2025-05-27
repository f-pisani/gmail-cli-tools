package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"

	"github.com/f-pisani/gmail-cli-tools/internal/auth"
	"github.com/f-pisani/gmail-cli-tools/internal/utils"
)

func main() {
	utils.InitLogger()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var credentialsPath string
	pflag.StringVar(&credentialsPath, "credentials-file", utils.GetEnvWithDefault("GMAIL_CREDENTIALS_FILE", "credentials.json"), "Path to OAuth2 credentials file (env: GMAIL_CREDENTIALS_FILE)")
	pflag.Parse()

	slog.Info("Starting authentication process", "credentials", credentialsPath)
	_, err := auth.GetGmailService(ctx, credentialsPath)
	if err != nil {
		slog.Error("Authentication failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Authentication successful! Token saved to token.json you can now use other commands.")
}
