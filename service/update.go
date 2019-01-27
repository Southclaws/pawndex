package service

import (
	"context"
	"time"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (app *App) updateList(queries []string) {
	zap.L().Debug("updating package list")

	for _, query := range queries {
		err := app.runQuery(query)
		if err != nil {
			zap.L().Error("failed to run query",
				zap.Error(err))
			continue
		}
	}
}

func (app *App) runQuery(query string) (err error) {
	page := 1
	total := 0
	for {
		zap.L().Debug("querying api for repositories",
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

		if total >= results.GetTotal() || total >= 1000 {
			break
		}

		time.Sleep(app.config.ScrapeInterval)

		page++
	}

	zap.L().Debug("done updating package list",
		zap.String("query", query),
		zap.Int("packages", total))

	return
}
