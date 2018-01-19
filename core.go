package main

import (
	"fmt"
	"time"

	"github.com/google/go-github/github"
	"go.uber.org/zap"
)

// App stores the app state
type App struct {
	config   Config
	gh       *github.Client
	toScrape chan github.Repository
	toIndex  chan github.Repository
}

// Start initialises the app and blocks until fatal error
func Start(config Config) {
	app := App{
		config:   config,
		gh:       github.NewClient(nil),
		toScrape: make(chan github.Repository, 1000),
		toIndex:  make(chan github.Repository),
	}

	app.Daemon()
}

// Daemon blocks forever and handles the main event loop and message bus
func (app *App) Daemon() {
	search := time.NewTicker(app.config.SearchInterval)
	scrape := time.NewTicker(time.Second)

loop:
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
				repo := <-app.toScrape
				err := app.scrapeRepo(repo)
				if err != nil {
					logger.Error("failed to scrape repository",
						zap.Error(err))
				}
			}()

		// toIndex consumes repositories that have been confirmed Pawn repos bu scrapeRepo
		case repo := <-app.toIndex:
			fmt.Println(repo)
			break loop
		}
	}
}
