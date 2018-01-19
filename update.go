package main

import (
	"context"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (app *App) updateList() (err error) {
	logger.Debug("updating package list")

	results, _, err := app.gh.Search.Repositories(
		context.Background(),
		"language:pawn&topic:pawn-package",
		&github.SearchOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to search repositories")
	}

	for _, repo := range results.Repositories {
		app.toScrape <- repo
	}

	logger.Debug("done updating package list",
		zap.Int("packages", results.GetTotal()))

	return
}
