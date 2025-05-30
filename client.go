package main

import (
	"context"
	"github.com/google/go-github/v28/github"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

func NewClient() *github.Client {
	token := viper.GetString("gh_token")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}
