package gmail

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"google.golang.org/api/gmail/v1"
)

type Attachment struct {
	ID       string
	Filename string
	MimeType string
	Size     int64
}

type Email struct {
	ID           string
	From         string
	To           string
	Subject      string
	Date         time.Time
	Body         string
	HTMLBody     string
	MarkdownBody string
	Labels       []string
	Attachments  []Attachment
}

func ParseMessage(msg *gmail.Message) (*Email, error) {
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

	if err := convertToMarkdown(email, false, false); err != nil {
		return nil, fmt.Errorf("failed to convert to markdown: %w", err)
	}

	return email, nil
}

func extractContent(payload *gmail.MessagePart, email *Email) {
	if payload.Body != nil && payload.Body.Data != "" {
		if payload.MimeType == "text/plain" {
			decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
			if err != nil {
				slog.Warn("Failed to decode plain text body", "error", err)
			} else {
				email.Body = string(decoded)
			}
		} else if payload.MimeType == "text/html" {
			decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
			if err != nil {
				slog.Warn("Failed to decode HTML body", "error", err)
			} else {
				email.HTMLBody = string(decoded)
			}
		}
	}

	for _, part := range payload.Parts {
		if part.Filename != "" {
			attachment := Attachment{
				ID:       part.Body.AttachmentId,
				Filename: part.Filename,
				MimeType: part.MimeType,
				Size:     part.Body.Size,
			}
			email.Attachments = append(email.Attachments, attachment)
			continue
		}

		if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
			decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err != nil {
				slog.Warn("Failed to decode plain text part", "error", err)
			} else if email.Body == "" {
				email.Body = string(decoded)
			}
		} else if part.MimeType == "text/html" && part.Body != nil && part.Body.Data != "" {
			decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err != nil {
				slog.Warn("Failed to decode HTML part", "error", err)
			} else if email.HTMLBody == "" {
				email.HTMLBody = string(decoded)
			}
		} else if strings.HasPrefix(part.MimeType, "multipart/") {
			extractContent(part, email)
		}
	}

	if email.Body == "" && email.HTMLBody != "" {
		email.Body = "[Email contains HTML content only]"
	}
}

func (e *Email) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ID: %s\n", e.ID))
	sb.WriteString(fmt.Sprintf("From: %s\n", e.From))
	sb.WriteString(fmt.Sprintf("To: %s\n", e.To))
	sb.WriteString(fmt.Sprintf("Subject: %s\n", e.Subject))
	sb.WriteString(fmt.Sprintf("Date: %s\n", e.Date.Format("Mon, 02 Jan 2006 15:04:05 -0700")))
	sb.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(e.Labels, ", ")))

	if len(e.Attachments) > 0 {
		sb.WriteString(fmt.Sprintf("\nAttachments (%d):\n", len(e.Attachments)))
		for _, att := range e.Attachments {
			sizeKB := float64(att.Size) / 1024
			sb.WriteString(fmt.Sprintf("  - %s (%.1f KB, %s)\n", att.Filename, sizeKB, att.MimeType))
		}
	}

	sb.WriteString(fmt.Sprintf("\nBody:\n%s\n", e.Body))

	if e.HTMLBody != "" && e.Body != "[Email contains HTML content only]" {
		sb.WriteString("\n[Note: Email also contains HTML version]\n")
	}

	return sb.String()
}

func convertToMarkdown(email *Email, removeImg, removeLink bool) error {
	if email.HTMLBody != "" {
		conv := converter.NewConverter(
			converter.WithPlugins(
				base.NewBasePlugin(),
				commonmark.NewCommonmarkPlugin(),
			),
		)

		if removeImg {
			conv.Register.TagType("img", converter.TagTypeRemove, converter.PriorityEarly)
		}
		if removeLink {
			conv.Register.RendererFor("a", converter.TagTypeInline, base.RenderAsPlaintextWrapper, converter.PriorityEarly)
		}
		markdown, err := conv.ConvertString(email.HTMLBody)
		if err != nil {
			return fmt.Errorf("failed to convert HTML to markdown: %w", err)
		}
		email.MarkdownBody = strings.TrimSpace(markdown)
	} else if email.Body != "" {
		email.MarkdownBody = email.Body
	}
	return nil
}
