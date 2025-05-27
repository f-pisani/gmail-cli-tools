package gmail

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/api/gmail/v1"
)

// ExportOptions contains all options for exporting emails
type ExportOptions struct {
	OutputFile         string
	IncludeAttachments bool
	AttachmentsDir     string
	StripImages        bool
	StripLinks         bool
}

// ExportToJSONL exports emails to JSONL format with all options using a context
func ExportToJSONL(ctx context.Context, client *Client, messages []*gmail.Message, options ExportOptions) error {
	outputDir := filepath.Dir(options.OutputFile)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	file, err := os.Create(options.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Store parsed emails if we need to download attachments later
	var parsedEmails []*Email
	if options.IncludeAttachments {
		parsedEmails = make([]*Email, 0, len(messages))
	}

	for i, msg := range messages {
		if i%10 == 0 {
			slog.Info("Processing emails", "progress", fmt.Sprintf("%d/%d", i+1, len(messages)))
		}

		// Add a small delay every 50 messages to avoid rate limits
		if i > 0 && i%50 == 0 {
			time.Sleep(1 * time.Second)
		}

		// Messages already have full details from GetMessagesByQuery
		email, err := ParseMessageWithOptions(msg, options.StripImages, options.StripLinks)
		if err != nil {
			slog.Warn("Failed to parse message", "id", msg.Id, "error", err)
			continue
		}

		jsonlEmail := convertToJSONL(msg, email)

		data, err := json.Marshal(jsonlEmail)
		if err != nil {
			slog.Warn("Failed to marshal email", "id", msg.Id, "error", err)
			continue
		}

		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("failed to write JSON line: %w", err)
		}
		if _, err := writer.Write([]byte("\n")); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}

		// Store parsed email for attachment download if needed
		if options.IncludeAttachments && len(email.Attachments) > 0 {
			parsedEmails = append(parsedEmails, email)
		}
	}

	slog.Info("Export completed", "total", len(messages), "output", options.OutputFile)

	// Download attachments after all emails are exported
	if options.IncludeAttachments && len(parsedEmails) > 0 {
		slog.Info("Downloading attachments", "directory", options.AttachmentsDir)

		if err := os.MkdirAll(options.AttachmentsDir, 0755); err != nil {
			return fmt.Errorf("failed to create attachments directory: %w", err)
		}

		for i, email := range parsedEmails {
			if i%10 == 0 {
				slog.Info("Downloading attachments", "progress", fmt.Sprintf("%d/%d", i+1, len(parsedEmails)))
			}

			emailAttachDir := filepath.Join(options.AttachmentsDir, email.ID)
			if err := os.MkdirAll(emailAttachDir, 0755); err != nil {
				slog.Warn("Failed to create email attachment directory", "email_id", email.ID, "error", err)
				continue
			}

			for _, att := range email.Attachments {
				if err := client.DownloadAttachment(ctx, email.ID, att.ID, att.Filename, emailAttachDir); err != nil {
					slog.Warn("Failed to download attachment",
						"filename", att.Filename, "email_id", email.ID, "error", err)
				}
			}
		}
	}

	return nil
}

// ParseMessageWithOptions parses a message with strip options for markdown
func ParseMessageWithOptions(msg *gmail.Message, stripImages, stripLinks bool) (*Email, error) {
	email := &Email{
		ID:          msg.Id,
		Labels:      msg.LabelIds,
		Attachments: []Attachment{},
	}

	headers := msg.Payload.Headers
	for _, header := range headers {
		switch header.Name {
		case "From":
			email.From = header.Value
		case "To":
			email.To = header.Value
		case "Subject":
			email.Subject = header.Value
		case "Date":
			parsedTime, err := mail.ParseDate(header.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse date '%s': %w", header.Value, err)
			}
			email.Date = parsedTime
		}
	}

	extractContent(msg.Payload, email)

	if err := convertToMarkdown(email, stripImages, stripLinks); err != nil {
		return nil, fmt.Errorf("failed to convert to markdown: %w", err)
	}

	return email, nil
}
