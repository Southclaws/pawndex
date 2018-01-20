package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Southclaws/sampctl/types"
	"github.com/google/go-github/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

var version string

// App stores the app state
type App struct {
	config   Config
	gh       *github.Client
	toScrape chan github.Repository
	toIndex  chan *types.Package
	index    map[string]*types.Package
}

// Start initialises the app and blocks until fatal error
func Start(config Config) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.GithubToken})
	tc := oauth2.NewClient(context.Background(), ts)

	app := App{
		config:   config,
		gh:       github.NewClient(tc),
		toScrape: make(chan github.Repository, 1000),
		toIndex:  make(chan *types.Package),
		index:    make(map[string]*types.Package),
	}

	logger.Info("starting pawndex and running initial list update",
		zap.String("version", version))

	err := app.updateList()
	if err != nil {
		logger.Error("error encountered while updating",
			zap.Error(err))
	}

	go app.runServer()
	app.Daemon()
}

// Daemon blocks forever and handles the main event loop and message bus
func (app *App) Daemon() {
	search := time.NewTicker(app.config.SearchInterval)
	scrape := time.NewTicker(time.Second)

	var scraped *types.Package

	for {
		select {

		// handles searching GitHub for all Pawn repositories
		case <-search.C:
			err := app.updateList()
			if err != nil {
				logger.Error("error encountered while updating",
					zap.Error(err))
			}

		// consumes repositories discovered by the search loop and investigates them
		case <-scrape.C:
			go func() {
				searched := <-app.toScrape
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
				err := app.scrapeRepo(ctx, searched)
				if err != nil {
					logger.Error("failed to scrape repository",
						zap.Error(err))
				}
				cancel()
			}()

		// toIndex consumes repositories that have been confirmed Pawn repos bu scrapeRepo
		case scraped = <-app.toIndex:
			fmt.Printf("SCRAPED: %p", scraped)
			str := fmt.Sprint(scraped)
			app.index[str] = scraped

			logger.Debug("discovered repo",
				zap.String("meta", str))
		}
	}
}

func (app *App) getPackageList() (result []*types.Package) {
	for _, pkg := range app.index {
		result = append(result, pkg)
	}
	return
}
