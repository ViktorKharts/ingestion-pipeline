package auth

import (
	"context"
	"net/http"
)

type Authenticator interface {
	GetHTTPClient(ctx context.Context) (*http.Client, error)
}

type Config struct {
	CredentialsPath string
	TokenPath       string
	Scopes          []string
}
