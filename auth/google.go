package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleAuthenticator struct {
	config    *oauth2.Config
	tokenPath string
}

func NewGoogleAuthenticator(cfg Config) (*GoogleAuthenticator, error) {
	b, err := os.ReadFile(cfg.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials: %w", err)
	}

	config, err := google.ConfigFromJSON(b, cfg.Scopes...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %w", err)
	}

	return &GoogleAuthenticator{
		config:    config,
		tokenPath: cfg.TokenPath,
	}, nil
}

func (g *GoogleAuthenticator) GetHTTPClient(ctx context.Context) (*http.Client, error) {
	tok, err := g.getTokenFromFile()
	if err != nil {
		tok, err = g.getTokenFromWeb()
		if err != nil {
			return nil, err
		}
		g.saveToken(tok)
	}
	return g.config.Client(ctx, tok), nil
}

func (g *GoogleAuthenticator) getTokenFromWeb() (*oauth2.Token, error) {
	authURL := g.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	exec.Command("xdg-open", authURL).Start()

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code %w", err)
	}

	tok, err := g.config.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web %w", err)
	}

	return tok, nil
}

func (g *GoogleAuthenticator) getTokenFromFile() (*oauth2.Token, error) {
	f, err := os.Open(g.tokenPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func (g *GoogleAuthenticator) saveToken(token *oauth2.Token) {
	log.Printf("Saving credential file to: %s\n", g.tokenPath)
	f, err := os.OpenFile(g.tokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
