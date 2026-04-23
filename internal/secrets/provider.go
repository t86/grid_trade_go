package secrets

import "context"

type AccountSecret struct {
	APIKey    string
	SecretKey string
}

type Provider interface {
	Load(ctx context.Context, ref string) (AccountSecret, error)
}
