package daemon

import (
	"context"
	"time"

	"github.com/Southclaws/pawndex/scraper"
	"github.com/Southclaws/pawndex/searcher"
	"github.com/Southclaws/pawndex/storage"
	"go.uber.org/zap"
)

type Daemon struct {
	Searcher       searcher.Searcher
	Scraper        scraper.Scraper
	Storer         storage.Storer
	SearchInterval time.Duration
	ScrapeInterval time.Duration
}

func (d *Daemon) Run(ctx context.Context) {
	search := time.NewTicker(d.SearchInterval)
	scrape := time.NewTicker(d.ScrapeInterval)

	f := func() error {
		select {
		case <-search.C:
			zap.L().Debug("starting search job")
			repos, err := d.Searcher.Search("topic:pawn-package", "language:pawn", "topic:sa-mp")
			if err != nil {
				return err
			}
			zap.L().Debug("finished search", zap.Int("repos", len(repos)))
			for _, r := range repos {
				zap.L().Debug("marking for scrape job", zap.String("repo", r))
				if err := d.Storer.MarkForScrape(r); err != nil {
					zap.L().Error("failed to mark repo for scraping", zap.String("name", r), zap.Error(err))
				}
			}
			zap.L().Debug("finished marking scrape jobs")

		case <-scrape.C:
			marked, err := d.Storer.GetMarked()
			if err != nil {
				return err
			}

			zap.L().Debug("starting scrape jobs", zap.Int("repos", len(marked)))

			for _, r := range marked {
				zap.L().Debug("scraping repository", zap.String("repo", r))
				pkg, err := d.Scraper.Scrape(ctx, r)
				if err != nil {
					zap.L().Error("failed to scrape repo",
						zap.String("name", r), zap.Error(err))
					continue
				}
				if err := d.Storer.Set(*pkg); err != nil {
					zap.L().Error("failed to store scraped package data",
						zap.String("name", r), zap.Error(err))
					continue
				}
			}

		case <-ctx.Done():
			return context.Canceled
		}
		return nil
	}

	for {
		if err := f(); err != nil {
			if err == context.Canceled {
				return
			}
			zap.L().Error("daemon error", zap.Error(err))
		}
	}
}
