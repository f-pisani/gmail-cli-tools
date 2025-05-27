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

func getClientWithTokenSource(ctx context.Context, config *oauth2.Config) (*http.Client, error) {
	accessTokenFile := "token.json"
	tok, err := readAccessTokenFromFile(accessTokenFile)
	if err != nil {
		tok, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, err
		}

		if err := saveToken(accessTokenFile, tok); err != nil {
			return nil, err
		}
	}

	tokenSource := config.TokenSource(ctx, tok)

	newToken, err := tokenSource.Token()
	if err != nil {
		slog.Warn("Error refreshing token", "error", err)
		tok, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, err
		}

		if err := saveToken(accessTokenFile, tok); err != nil {
			return nil, err
		}
		return config.Client(ctx, tok), nil
	}

	if newToken.AccessToken != tok.AccessToken {
		if err := saveToken(accessTokenFile, newToken); err != nil {
			return nil, err
		}
		tok = newToken
	}

	return oauth2.NewClient(ctx, tokenSource), nil
}

func generateStateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	codeCh := make(chan string)
	stateCh := make(chan string)
	errCh := make(chan error)

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
			errCh <- err
		}
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	stateToken, err := generateStateToken()
	if err != nil {
		return nil, err
	}

	config.RedirectURL = "http://localhost:8080"
	authURL := config.AuthCodeURL(stateToken, oauth2.AccessTypeOffline)

	slog.Info("Opening browser for authorization")
	utils.OpenBrowserURL(authURL)

	select {
	case err := <-errCh:
		return nil, err
	case code := <-codeCh:
		receivedState := <-stateCh
		if receivedState != stateToken {
			return nil, errors.New("invalid state token received")
		}

		tok, err := config.Exchange(ctx, code)
		if err != nil {
			return nil, err
		}
		return tok, nil

	case <-time.After(5 * time.Minute):
		return nil, errors.New("authorization timeout - no response received within 5 minutes")
	}
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
		return err
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

	client, err := getClientWithTokenSource(ctx, config)
	if err != nil {
		return nil, err
	}

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return srv, nil
}
