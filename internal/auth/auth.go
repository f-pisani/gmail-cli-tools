package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/f-pisani/gmail-cli-tools/internal/utils"
)

func getClientWithTokenSource(ctx context.Context, config *oauth2.Config) *http.Client {
	accessTokenFile := "token.json"
	tok, err := readAccessTokenFromFile(accessTokenFile)
	if err != nil {
		tok = getTokenFromWeb(ctx, config)

		// TODO: unhandled error, should propagate
		saveToken(accessTokenFile, tok)
	}

	tokenSource := config.TokenSource(ctx, tok)

	newToken, err := tokenSource.Token()
	if err != nil {
		slog.Warn("Error refreshing token", "error", err)
		tok = getTokenFromWeb(ctx, config)

		// TODO: unhandled error, should propagate
		saveToken(accessTokenFile, tok)
		return config.Client(ctx, tok)
	}

	if newToken.AccessToken != tok.AccessToken {
		slog.Info("Token refreshed automatically")

		// TODO: unhandled error, should propagate
		saveToken(accessTokenFile, newToken)
		tok = newToken
	}

	return oauth2.NewClient(ctx, tokenSource)
}

func generateStateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	codeCh := make(chan string)
	stateCh := make(chan string)

	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
			<html>
			<head><title>Gmail Reader - Authorization Successful</title></head>
			<body>
				<h1>Authorization Successful!</h1>
				<p>You can now close this window and return to the terminal.</p>
				<script>window.close();</script>
			</body>
			</html>
		`))

		codeCh <- code
		stateCh <- state
	})

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed to start", "error", err)

			// TODO: why are we calling os.Exit, should propagate error
			os.Exit(1)
		}
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	stateToken, err := generateStateToken()
	if err != nil {
		slog.Error("Failed to generate state token", "error", err)
		os.Exit(1)
	}

	config.RedirectURL = "http://localhost:8080"
	authURL := config.AuthCodeURL(stateToken, oauth2.AccessTypeOffline)

	slog.Info("Opening browser for authorization")
	utils.OpenBrowserURL(authURL)

	select {
	case code := <-codeCh:
		receivedState := <-stateCh
		if receivedState != stateToken {
			slog.Error("Invalid state token received", "expected", stateToken, "received", receivedState)

			// TODO: why are we calling os.Exit, should return err and propagate properly
			os.Exit(1)
		}

		tok, err := config.Exchange(ctx, code)
		if err != nil {
			slog.Error("Unable to retrieve token from authorization code", "error", err)

			// TODO: why are we calling os.Exit, should return err and propagate properly
			os.Exit(1)
		}
		return tok

	case <-time.After(5 * time.Minute):
		slog.Error("Authorization timeout - no response received within 5 minutes")

		// TODO: why are we calling os.Exit, should return err and propagate properly
		os.Exit(1)
	}

	return nil
}

func readAccessTokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	tok := &oauth2.Token{}
	if err = json.NewDecoder(f).Decode(tok); err != nil {
		return nil, err
	}

	return tok, nil
}

func saveToken(path string, token *oauth2.Token) error {
	slog.Info("Saving credential file", "path", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		slog.Error("Unable to cache oauth token", "error", err)

		// TODO: why are we calling os.Exit, should return err and propagate properly
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	return json.NewEncoder(f).Encode(token)
}

func GetGmailService(ctx context.Context, credentialsFile string) (*gmail.Service, error) {
	jsonKey, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(jsonKey, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, err
	}

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(getClientWithTokenSource(ctx, config)))
	if err != nil {
		return nil, err
	}

	return srv, nil
}
