package gmail

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/api/gmail/v1"
)

type Client struct {
	service *gmail.Service
}

func NewClient(service *gmail.Service) *Client {
	return &Client{
		service: service,
	}
}

func (c *Client) GetLabelID(ctx context.Context, labelName string) (string, error) {
	user := "me"
	labelsCall := c.service.Users.Labels.List(user)
	labels, err := labelsCall.Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("unable to retrieve labels: %v", err)
	}

	for _, label := range labels.Labels {
		if label.Name == labelName {
			return label.Id, nil
		}
	}

	return "", fmt.Errorf("label '%s' not found", labelName)
}

// GetMessagesByQuery fetches messages with full details using batch requests
func (c *Client) GetMessagesByQuery(ctx context.Context, query string, limit int64) ([]*gmail.Message, error) {
	user := "me"
	var allMessages []*gmail.Message
	var pageToken string

	// Gmail API max is 500 per page
	const maxPageSize int64 = 500

	for {
		remaining := limit - int64(len(allMessages))
		if remaining <= 0 {
			break
		}

		pageSize := remaining
		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}

		call := c.service.Users.Messages.List(user).Q(query).MaxResults(pageSize)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		response, err := call.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve messages: %v", err)
		}

		// Fetch full message details for all messages in this page
		for _, msg := range response.Messages {
			fullMsg, err := c.service.Users.Messages.Get(user, msg.Id).Format("full").Context(ctx).Do()
			if err != nil {
				slog.Warn("Error retrieving message", "message_id", msg.Id, "error", err)
				continue
			}
			allMessages = append(allMessages, fullMsg)

			if int64(len(allMessages)) >= limit {
				return allMessages[:limit], nil
			}
		}

		pageToken = response.NextPageToken
		if pageToken == "" {
			break
		}

		slog.Info("Fetching messages", "fetched_count", len(allMessages), "limit", limit)
	}

	return allMessages, nil
}

func (c *Client) GetEmailsByLabelName(ctx context.Context, labelName string, limit int64) ([]*gmail.Message, error) {
	// This supports both standard labels (INBOX, SENT) and custom labels
	query := fmt.Sprintf("label:%s", labelName)
	messages, err := c.GetMessagesByQuery(ctx, query, limit)

	if err == nil && len(messages) > 0 {
		return messages, nil
	}

	firstErr := err

	labelID, labelErr := c.GetLabelID(ctx, labelName)
	if labelErr == nil {
		query = fmt.Sprintf("label:%s", labelID)
		messages, err = c.GetMessagesByQuery(ctx, query, limit)
		if err == nil {
			return messages, nil
		}
	}

	if firstErr != nil {
		return nil, fmt.Errorf("unable to find emails with label '%s': direct query failed: %v, label ID query failed: %v", labelName, firstErr, err)
	}

	return messages, nil
}

func (c *Client) ListLabels(ctx context.Context) ([]*gmail.Label, error) {
	user := "me"
	labelsCall := c.service.Users.Labels.List(user)
	labels, err := labelsCall.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve labels: %v", err)
	}
	return labels.Labels, nil
}
