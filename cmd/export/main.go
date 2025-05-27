package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"

	"github.com/yourusername/gmail-cli-tools/internal/auth"
	"github.com/yourusername/gmail-cli-tools/internal/gmail"
	"github.com/yourusername/gmail-cli-tools/internal/utils"
)

func main() {
	utils.InitLogger()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var (
		labelName           string
		limit               int64
		credentialsPath     string
		downloadAttachments bool
		attachmentsDir      string
		outputFile          string
		removeImg           bool
		removeLink          bool
	)

	pflag.StringVar(&labelName, "label", utils.GetEnvWithDefault("GMAIL_LABEL", "INBOX"), "Gmail label name to filter emails (env: GMAIL_LABEL)")
	pflag.Int64Var(&limit, "limit", utils.GetEnvWithDefault("GMAIL_LIMIT", int64(500)), "Maximum number of emails to retrieve (env: GMAIL_LIMIT)")
	pflag.StringVar(&credentialsPath, "credentials-file", utils.GetEnvWithDefault("GMAIL_CREDENTIALS_FILE", "credentials.json"), "Path to credentials.json file (env: GMAIL_CREDENTIALS_FILE)")
	pflag.BoolVar(&downloadAttachments, "download-attachments", utils.GetEnvWithDefault("GMAIL_DOWNLOAD_ATTACHMENTS", false), "Download all attachments from retrieved emails (env: GMAIL_DOWNLOAD_ATTACHMENTS)")
	pflag.StringVar(&attachmentsDir, "attachments-dir", utils.GetEnvWithDefault("GMAIL_ATTACHMENTS_DIR", "attachments"), "Directory path to save attachments (env: GMAIL_ATTACHMENTS_DIR)")
	pflag.StringVar(&outputFile, "output", utils.GetEnvWithDefault("GMAIL_OUTPUT_FILE", "emails.jsonl"), "Output JSONL file path (env: GMAIL_OUTPUT_FILE)")
	pflag.BoolVar(&removeImg, "markdown-strip-img", utils.GetEnvWithDefault("GMAIL_STRIP_IMG", false), "Remove <img> tags from markdown output (env: GMAIL_STRIP_IMG)")
	pflag.BoolVar(&removeLink, "markdown-strip-link", utils.GetEnvWithDefault("GMAIL_STRIP_LINK", false), "Remove links from markdown output, keeping only the label (env: GMAIL_STRIP_LINK)")
	pflag.Parse()

	service, err := auth.GetGmailService(ctx, credentialsPath)
	if err != nil {
		slog.Error("Failed to get Gmail service", "error", err)
		os.Exit(1)
	}

	client := gmail.NewClient(service)

	slog.Info("Fetching emails",
		"label", labelName,
		"limit", limit,
		"markdown_strip_img", removeImg,
		"markdown_strip_link", removeLink)

	query := ""
	if labelName != "" {
		labelID, err := client.GetLabelID(ctx, labelName)
		if err != nil {
			query = "label:" + labelName
		} else {
			query = "label:" + labelID
		}
	}

	messages, err := client.GetMessagesByQuery(ctx, query, limit)
	if err != nil {
		slog.Error("Failed to get emails", "error", err)
		os.Exit(1)
	}

	if len(messages) == 0 {
		slog.Info("No emails found with the specified criteria", "label", labelName)
		return
	}

	slog.Info("Found emails", "count", len(messages))

	exportOptions := gmail.ExportOptions{
		OutputFile:         outputFile,
		IncludeAttachments: downloadAttachments,
		AttachmentsDir:     attachmentsDir,
		StripImages:        removeImg,
		StripLinks:         removeLink,
	}

	if err := gmail.ExportToJSONL(ctx, client, messages, exportOptions); err != nil {
		slog.Error("Failed to export emails", "error", err)
		os.Exit(1)
	}

	slog.Info("Successfully exported emails",
		"count", len(messages),
		"output", outputFile,
		"attachments_downloaded", downloadAttachments,
	)
}
