package main

import (
	"context"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
)

type GarageClient struct {
	Client  *garage.APIClient
	Token   string
	Scheme  string
	Host    string
}

func NewGarageClient(scheme, host, token string) (*GarageClient, error) {
	cfg := garage.NewConfiguration()
	cfg.Scheme = scheme
	cfg.Host = host

	client := garage.NewAPIClient(cfg)
	return &GarageClient{
		Client: client,
		Token:  token,
		Scheme: scheme,
		Host:   host,
	}, nil
}

func (c *GarageClient) WithAuth(ctx context.Context) context.Context {
	return context.WithValue(ctx, garage.ContextAccessToken, c.Token)
}
