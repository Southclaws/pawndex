package main

import (
	"time"

	"github.com/google/go-github/github"
	"go.uber.org/zap"
)

// App stores the app state
type App struct {
	gh *github.Client
}

// Start initialises the app and blocks until fatal error
func Start(config Config) {
	app := App{
		gh: github.NewClient(nil),
	}

	ticker := time.NewTicker(config.SearchInterval)

	for range ticker.C {
		err := app.updateList()
		if err != nil {
			logger.Error("error encountered while updating",
				zap.Error(err))
		}
	}
}
