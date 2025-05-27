package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

func (c *Client) DownloadAttachment(ctx context.Context, messageID, attachmentID, filename, outputDir string) error {
	user := "me"

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	attachment, err := c.service.Users.Messages.Attachments.Get(user, messageID, attachmentID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get attachment: %v", err)
	}

	data, err := base64.URLEncoding.DecodeString(attachment.Data)
	if err != nil {
		return fmt.Errorf("failed to decode attachment: %v", err)
	}

	outputPath := filepath.Join(outputDir, filename)
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write attachment to file: %v", err)
	}

	return nil
}
