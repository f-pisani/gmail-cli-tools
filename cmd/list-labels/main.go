package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/spf13/pflag"

	"github.com/yourusername/gmail-cli-tools/internal/auth"
	"github.com/yourusername/gmail-cli-tools/internal/utils"
)

func main() {
	utils.InitLogger()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var credentialsPath string

	pflag.StringVar(&credentialsPath, "credentials-file", utils.GetEnvWithDefault("GMAIL_CREDENTIALS_FILE", "credentials.json"), "Path to credentials.json file (env: GMAIL_CREDENTIALS_FILE)")
	pflag.Parse()

	service, err := auth.GetGmailService(ctx, credentialsPath)
	if err != nil {
		slog.Error("Failed to get Gmail service", "error", err)
		os.Exit(1)
	}

	user := "me"
	r, err := service.Users.Labels.List(user).Context(ctx).Do()
	if err != nil {
		slog.Error("Unable to retrieve labels", "error", err)
		os.Exit(1)
	}

	if len(r.Labels) == 0 {
		slog.Info("No labels found")
		return
	}

	sort.Slice(r.Labels, func(i, j int) bool {
		return r.Labels[i].Name < r.Labels[j].Name
	})

	for _, l := range r.Labels {
		slog.Info("Label found", "id", l.Id, "name", l.Name, "type", l.Type)
	}
	slog.Info("Listed labels successfully", "count", len(r.Labels))
}
