package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
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
	toIndex  chan *Package
	index    map[string]*Package
	lock     sync.RWMutex
}

// Classification represents how compatible or easy to use a package is. If a package contains a
// package definition file, it's of a higher classification than one that does not contain one.
type Classification string

var (
	classificationPawnPackage Classification = "full"
	classificationBarebones   Classification = "basic"
	classificationBuried      Classification = "buried"
)

// Package wraps types.Package and adds extra fields
type Package struct {
	types.Package
	Classification Classification `json:"classification"` // classification represents how conformative the package is
	Stars          int            `json:"stars"`          // GitHub stars
	Updated        time.Time      `json:"updated"`        // last updated
}

// Start initialises the app and blocks until fatal error
func Start(config Config) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.GithubToken})
	tc := oauth2.NewClient(context.Background(), ts)

	app := App{
		config:   config,
		gh:       github.NewClient(tc),
		toScrape: make(chan github.Repository, 1000),
		toIndex:  make(chan *Package),
		index:    make(map[string]*Package),
		lock:     sync.RWMutex{},
	}

	logger.Info("starting pawndex and running initial list update",
		zap.String("version", version))

	err := app.loadCache()
	if err != nil && err.Error() != "open cache.json: no such file or directory" {
		logger.Fatal("failed to load cache",
			zap.Error(err))
	}
	app.updateList([]string{"topic:pawn-package", "language:pawn", "topic:sa-mp"})

	go app.runServer()
	app.Daemon()
}

// Daemon blocks forever and handles the main event loop and message bus
func (app *App) Daemon() {
	search := time.NewTicker(app.config.SearchInterval)
	scrape := time.NewTicker(app.config.ScrapeInterval)

	var err error
	var scraped *Package

	for {
		select {

		// handles searching GitHub for all Pawn repositories
		case <-search.C:
			app.updateList([]string{"topic:pawn-package", "language:pawn", "topic:sa-mp"})

		// consumes repositories discovered by the search loop and investigates them
		case <-scrape.C:
			go func() {
				searched := <-app.toScrape
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
				err = app.scrapeRepo(ctx, searched)
				if err != nil {
					logger.Error("failed to scrape repository",
						zap.Error(err))
				}
				cancel()
			}()

		// toIndex consumes repositories that have been confirmed Pawn repos bu scrapeRepo
		case scraped = <-app.toIndex:
			str := fmt.Sprint(scraped)

			app.lock.Lock()
			app.index[str] = scraped
			app.lock.Unlock()

			logger.Debug("discovered repo",
				zap.String("meta", str))

			err = app.dumpCache()
			if err != nil {
				logger.Error("failed to dump cache",
					zap.Error(err))
			}
		}
	}
}

func (app *App) getPackageList() (result []*Package) {
	app.lock.RLock()
	for _, pkg := range app.index {
		result = append(result, pkg)
	}
	app.lock.RUnlock()
	return
}

func (app *App) dumpCache() (err error) {
	list := app.getPackageList()
	payload, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile("cache.json", payload, 0600)
	return
}

func (app *App) loadCache() (err error) {
	contents, err := ioutil.ReadFile("cache.json")
	if err != nil {
		return
	}
	var list []Package
	err = json.Unmarshal(contents, &list)
	if err != nil {
		return
	}
	for _, pkg := range list {
		str := fmt.Sprint(pkg)
		logger.Debug("loaded from cache", zap.String("name", str))
		app.index[str] = &pkg
	}
	return
}
