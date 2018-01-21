package main

import (
	"context"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (app *App) updateList() (err error) {
	logger.Debug("updating package list")

	page := 1
	total := 0
	for {
		logger.Debug("querying api for repositories",
			zap.Int("page", page),
			zap.Int("total", total))

		results, _, err := app.gh.Search.Repositories(
			context.Background(),
			"language:pawn",
			&github.SearchOptions{ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			}})
		if err != nil {
			return errors.Wrap(err, "failed to search repositories")
		}

		for _, repo := range results.Repositories {
			app.toScrape <- repo
			total++
		}

		if total >= results.GetTotal() {
			break
		}

		page++
	}

	logger.Debug("done updating package list",
		zap.Int("packages", total))

	return
}
