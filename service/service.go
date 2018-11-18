package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// App stores the app state
type App struct {
	config   Config
	gh       *github.Client
	toScrape chan github.Repository
	toIndex  chan Package
	index    map[string]Package
	lock     sync.RWMutex
	metrics  Metrics
}

// Config stores static configuration
type Config struct {
	Bind           string        `required:"true"` // bind interface
	Domain         string        `required:"true"` // public domain
	GithubToken    string        `required:"true"` // GitHub API token
	SearchInterval time.Duration `required:"true"` // interval between checks
	ScrapeInterval time.Duration `required:"true"` // interval between scrapes
	Cache          string        `required:"true"` // cache for persistence
}

// Initialise prepres the service for starting
func Initialise(ctx context.Context, config Config) (app *App, err error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.GithubToken})
	tc := oauth2.NewClient(ctx, ts)

	app = &App{
		config:   config,
		gh:       github.NewClient(tc),
		toScrape: make(chan github.Repository, 2000),
		toIndex:  make(chan Package, 2000),
		index:    make(map[string]Package),
		lock:     sync.RWMutex{},
		metrics:  newMetrics(),
	}

	err = app.loadCache()
	if err != nil && !strings.HasSuffix(err.Error(), "no such file or directory") {
		return
	}
	app.metrics.IndexSize.Set(float64(len(app.index)))

	return
}

// Start initialises the app and blocks until fatal error
func (app *App) Start() (err error) {
	go app.runServer()
	app.updateList([]string{"topic:pawn-package", "language:pawn", "topic:sa-mp"})
	app.Daemon()
	return
}
