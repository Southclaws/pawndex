package service

import (
	"context"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/Southclaws/pawndex/api"
	"github.com/Southclaws/pawndex/daemon"
	"github.com/Southclaws/pawndex/scraper"
	"github.com/Southclaws/pawndex/searcher"
	"github.com/Southclaws/pawndex/storage"
)

// App stores the app state
type App struct {
	config Config
	gh     *github.Client
	server api.Server
	daemon daemon.Daemon
}

// Config stores static configuration
type Config struct {
	Bind           string        `required:"true"` // bind interface
	GithubToken    string        `required:"true"` // GitHub API token
	SearchInterval time.Duration `required:"true"` // interval between checks
	ScrapeInterval time.Duration `required:"true"` // interval between scrapes
	DatabasePath   string        `required:"true"` // cache for persistence
}

// Initialise prepres the service for starting
func Initialise(ctx context.Context, config Config) (app *App, err error) {
	gh := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.GithubToken})))
	search := searcher.GitHubSearcher{GitHub: gh}
	scrape := scraper.GitHubScraper{GitHub: gh}
	store, err := storage.New(config.DatabasePath)
	if err != nil {
		return nil, err
	}

	return &App{
		config: config,
		gh:     gh,
		server: api.New(config.Bind, store),
		daemon: daemon.Daemon{
			Searcher:       &search,
			Scraper:        &scrape,
			Storer:         store,
			SearchInterval: config.SearchInterval,
			ScrapeInterval: config.ScrapeInterval,
		},
	}, nil
}

// Start initialises the app and blocks until fatal error
func (app *App) Start(ctx context.Context) (err error) {
	errs := make(chan error)

	go func() {
		errs <- app.server.Run()
	}()

	go func() {
		app.daemon.Run(ctx)
	}()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return context.Canceled
	}
}
