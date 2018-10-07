package main

import (
	"context"
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (app *App) updateList(queries []string) {
	logger.Debug("updating package list")

	for _, query := range queries {
		err := app.runQuery(query)
		if err != nil {
			logger.Error("failed to run query",
				zap.Error(err))
			continue
		}
	}
}

func (app *App) runQuery(query string) (err error) {
	page := 1
	total := 0
	for {
		logger.Debug("querying api for repositories",
			zap.Int("page", page),
			zap.Int("total", total))

		results, _, err := app.gh.Search.Repositories(
			context.Background(),
			query,
			&github.SearchOptions{ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			}})
		if err != nil {
			return errors.Wrap(err, "failed to search repositories")
		}
		app.metrics.SearchRate.Observe(1)

		for _, repo := range results.Repositories {
			app.toScrape <- repo
			total++
		}

		if total >= results.GetTotal() {
			break
		}

		time.Sleep(app.config.ScrapeInterval)

		page++
	}

	logger.Debug("done updating package list",
		zap.String("query", query),
		zap.Int("packages", total))

	return
}
