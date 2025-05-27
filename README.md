# Gmail CLI Tools

A suite of command-line tools for exporting and managing Gmail messages with OAuth authentication. Export emails to JSONL format for easy processing and analysis.

## Features

- OAuth2 authentication with Gmail API
- Export emails to JSONL format
- List Gmail labels
- Export email metadata including attachments
- Support for large mailboxes (>500 emails)
- Secure token storage

## Installation

```bash
go get github.com/f-pisani/gmail-cli-tools
```

Or build from source:

```bash
make build
```

## Setup

1. **Get Gmail API credentials:**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing
   - Enable Gmail API
   - Create OAuth 2.0 credentials (Desktop application)
   - Download credentials as `credentials.json`

2. **Authenticate:**
   ```bash
   # First time authentication
   go run cmd/auth/main.go -credentials=credentials.json
   
   # Or with make
   make auth
   ```

## Usage

### List Labels
```bash
# List all available Gmail labels
go run cmd/list-labels/main.go

# Or with make
make list-labels
```

### Export Emails
```bash
# Export emails from INBOX (default)
go run cmd/export/main.go --label=INBOX --output=emails.jsonl

# Export with attachments
go run cmd/export/main.go --label=INBOX --download-attachments

# Export from custom label with limit
go run cmd/export/main.go --label="MyLabel" --limit=1000

# Strip markdown images and links
go run cmd/export/main.go --markdown-strip-link --markdown-strip-img

# Use environment variables
export GMAIL_LABEL="Important"
export GMAIL_LIMIT=1000
export GMAIL_STRIP_LINK=true
go run cmd/export/main.go

# Or with make
make export
```

### Command Options

#### auth
- `--credentials-file` - Path to OAuth2 credentials file (default: `credentials.json`, env: `GMAIL_CREDENTIALS_FILE`)

#### list-labels
- `--credentials-file` - Path to OAuth2 credentials file (default: `credentials.json`, env: `GMAIL_CREDENTIALS_FILE`)

#### export
- `--credentials-file` - Path to OAuth2 credentials file (default: `credentials.json`, env: `GMAIL_CREDENTIALS_FILE`)
- `--label` - Gmail label to filter emails (default: `INBOX`, env: `GMAIL_LABEL`)
- `--limit` - Maximum number of emails to retrieve (default: `500`, env: `GMAIL_LIMIT`)
- `--output` - Output JSONL file path (default: `emails.jsonl`, env: `GMAIL_OUTPUT_FILE`)
- `--download-attachments` - Download attachment files (default: `false`, env: `GMAIL_DOWNLOAD_ATTACHMENTS`)
- `--attachments-dir` - Directory to save attachments (default: `attachments`, env: `GMAIL_ATTACHMENTS_DIR`)
- `--markdown-strip-img` - Remove `<img>` tags from markdown (default: `false`, env: `GMAIL_STRIP_IMG`)
- `--markdown-strip-link` - Remove links from markdown, keep text (default: `false`, env: `GMAIL_STRIP_LINK`)
- `--include-raw` - Include raw RFC822 message in base64 (default: `false`, env: `GMAIL_INCLUDE_RAW`)

## Output Format

Emails are exported in JSONL (JSON Lines) format with the following structure:

```json
{
  "id": "message_id",
  "thread_id": "thread_id",
  "label_ids": ["INBOX", "UNREAD"],
  "subject": "Email subject",
  "from": "sender@example.com",
  "to": ["recipient@example.com"],
  "cc": ["cc@example.com"],
  "bcc": ["bcc@example.com"],
  "date": "2024-01-15T10:30:00Z",
  "body": {
    "text": "Plain text content",
    "html": "<html>HTML content</html>",
    "markdown": "Markdown content"
  },
  "attachments": [
    {
      "id": "attachment_id",
      "filename": "document.pdf",
      "mime_type": "application/pdf",
      "size": 12345
    }
  ],
  "headers": {
    "Message-ID": "<123@example.com>",
    "References": "...",
    "In-Reply-To": "..."
  },
  "raw": "base64_encoded_raw_message"
}
```

## Development

### Building
```bash
# Build all commands
make build

# Build for multiple platforms
make build-all
```

### Code Quality
```bash
# Format code
make fmt

# Run linter
make vet

# Run all checks (format and vet)
make check
```

## Security

- OAuth tokens are stored locally in `token.json` with 0600 permissions
- Uses secure random state tokens for OAuth CSRF protection
- Credentials are never logged or exposed
- Token refresh happens automatically

## License

MIT