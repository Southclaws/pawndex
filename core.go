package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/Southclaws/sampctl/types"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

var version string

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

// Classification represents how compatible or easy to use a package is. If a package contains a
// package definition file, it's of a higher classification than one that does not contain one.
type Classification string

var (
	classificationInvalid     Classification = "invalid"
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
	Topics         []string       `json:"topics"`         // GitHub topics
	Tags           []string       `json:"tags"`           // Git tags
}

// Start initialises the app and blocks until fatal error
func Start(config Config) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.GithubToken})
	tc := oauth2.NewClient(context.Background(), ts)

	app := App{
		config:   config,
		gh:       github.NewClient(tc),
		toScrape: make(chan github.Repository, 2000),
		toIndex:  make(chan Package, 2000),
		index:    make(map[string]Package),
		lock:     sync.RWMutex{},
		metrics:  newMetrics(),
	}

	logger.Info("starting pawndex and running initial list update",
		zap.String("version", version))

	err := app.loadCache()
	if err != nil && !strings.HasSuffix(err.Error(), "no such file or directory") {
		logger.Error("failed to load cache",
			zap.Error(err))
	}
	app.metrics.IndexSize.Set(float64(len(app.index)))

	go app.runServer()
	app.updateList([]string{"topic:pawn-package", "language:pawn", "topic:sa-mp"})
	app.Daemon()
}

// Daemon blocks forever and handles the main event loop and message bus
func (app *App) Daemon() {
	search := time.NewTicker(app.config.SearchInterval)
	scrape := time.NewTicker(app.config.ScrapeInterval)

	f := func() (err error) {
		select {

		// handles searching GitHub for all Pawn repositories
		case <-search.C:
			if len(app.toScrape) > 0 {
				logger.Warn("cannot search with items still to scrape, raise search interval",
					zap.Int("toScrape", len(app.toScrape)))
				return
			}
			app.updateList([]string{"topic:pawn-package", "language:pawn", "topic:sa-mp"})

		// consumes repositories discovered by the search loop and investigates them
		case <-scrape.C:
			if len(app.toScrape) == 0 {
				return
			}

			go func() {
				searched := <-app.toScrape
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
				defer cancel()
				err := app.scrapeRepo(ctx, searched)
				if err != nil {
					logger.Error("failed to scrape repository",
						zap.Error(err))
				}
				app.metrics.ScrapeRate.Observe(1)
				app.metrics.ScrapeQueue.Set(float64(len(app.toScrape)))
			}()

		// toIndex consumes repositories that have been confirmed Pawn repos
		case scraped := <-app.toIndex:
			str := fmt.Sprint(scraped)

			app.lock.Lock()
			app.index[str] = scraped
			app.lock.Unlock()

			app.metrics.IndexRate.Observe(1)
			app.metrics.IndexSize.Set(float64(len(app.index)))
			app.metrics.IndexQueue.Set(float64(len(app.toIndex)))

			logger.Debug("discovered repo",
				zap.String("meta", str))

			err := app.dumpCache()
			if err != nil {
				return errors.Wrap(err, "failed to dump cache")
			}
		}
		return
	}

	for {
		err := f()
		if err != nil {
			logger.Error("daemon error", zap.Error(err))
		}
	}
}

func (app *App) getPackageList() (result []Package) {
	app.lock.RLock()
	defer app.lock.RUnlock()

	visited := make(map[string]struct{})
	for key, pkg := range app.index {
		if _, ok := visited[key]; !ok {
			result = append(result, pkg)
			visited[key] = struct{}{}
		}
	}
	return
}

func (app *App) getPackage(user, repo string) (result Package, exists bool) {
	app.lock.RLock()
	defer app.lock.RUnlock()

	result, exists = app.index[fmt.Sprintf("%s/%s", user, repo)]
	return
}

func (app *App) dumpCache() (err error) {
	list := app.getPackageList()
	payload, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(app.config.Cache, payload, 0600)
	return
}

func (app *App) loadCache() (err error) {
	contents, err := ioutil.ReadFile(app.config.Cache)
	if err != nil {
		return
	}
	var list []Package
	err = json.Unmarshal(contents, &list)
	if err != nil {
		return
	}

	for i := range list {
		pkg := list[i]
		str := fmt.Sprint(pkg)
		logger.Debug("loaded from cache", zap.String("name", str))
		app.index[str] = pkg
	}
	logger.Debug("loaded cache", zap.Int("size", len(app.index)))
	return
}
