package gmail

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/gmail/v1"
)

// JSONLEmail represents the email structure for JSONL export
type JSONLEmail struct {
	ID          string               `json:"id"`
	ThreadID    string               `json:"thread_id"`
	LabelIDs    []string             `json:"label_ids"`
	Subject     string               `json:"subject"`
	From        string               `json:"from"`
	To          []string             `json:"to"`
	Cc          []string             `json:"cc,omitempty"`
	Bcc         []string             `json:"bcc,omitempty"`
	Date        string               `json:"date"`
	Body        BodyFormats          `json:"body"`
	Attachments []AttachmentMetadata `json:"attachments,omitempty"`
	Headers     map[string]string    `json:"headers"`
}

// BodyFormats contains all body format variations
type BodyFormats struct {
	Text     string `json:"text"`
	HTML     string `json:"html"`
	Markdown string `json:"markdown"`
}

// AttachmentMetadata contains detailed attachment information
type AttachmentMetadata struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

func convertToJSONL(msg *gmail.Message, email *Email) JSONLEmail {
	headers := make(map[string]string)
	for _, header := range msg.Payload.Headers {
		headers[header.Name] = header.Value
	}

	to := parseRecipients(headers["To"])
	cc := parseRecipients(headers["Cc"])
	bcc := parseRecipients(headers["Bcc"])

	var attachments []AttachmentMetadata
	for _, att := range email.Attachments {
		attachments = append(attachments, AttachmentMetadata{
			ID:       att.ID,
			Filename: att.Filename,
			MimeType: att.MimeType,
			Size:     att.Size,
		})
	}

	return JSONLEmail{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		LabelIDs: msg.LabelIds,
		Subject:  email.Subject,
		From:     email.From,
		To:       to,
		Cc:       cc,
		Bcc:      bcc,
		Date:     email.Date.Format("2006-01-02T15:04:05Z07:00"),
		Body: BodyFormats{
			Text:     email.Body,
			HTML:     email.HTMLBody,
			Markdown: email.MarkdownBody,
		},
		Attachments: attachments,
		Headers:     headers,
	}
}

func parseRecipients(recipients string) []string {
	if recipients == "" {
		return nil
	}

	// Use the standard library mail parser
	addressList, err := mail.ParseAddressList(recipients)
	if err != nil {
		// Fallback to simple comma split if parsing fails
		// This handles malformed addresses gracefully
		var result []string
		for _, addr := range strings.Split(recipients, ",") {
			if trimmed := strings.TrimSpace(addr); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	var result []string
	for _, addr := range addressList {
		// Return the full address string (includes name if present)
		result = append(result, addr.String())
	}
	return result
}

func DownloadAttachments(ctx context.Context, client *Client, emails []*Email, outputDir string) error {
	attachDir := filepath.Join(outputDir, "attachments")
	if err := os.MkdirAll(attachDir, 0755); err != nil {
		return fmt.Errorf("failed to create attachments directory: %w", err)
	}

	for _, email := range emails {
		if len(email.Attachments) == 0 {
			continue
		}

		emailAttachDir := filepath.Join(attachDir, email.ID)
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

	return nil
}
