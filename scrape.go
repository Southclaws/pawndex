package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Southclaws/sampctl/types"
	"github.com/Southclaws/sampctl/versioning"
	"github.com/google/go-github/github"
	"go.uber.org/zap"
)

// scrapeRepo is responsible for taking a repo and checking its contents for the qualifying
// properties of a Pawn Package. This includes the presence of one or more .inc files and optionally
// a pawn.json or pawn.yaml file. If one of these files exists, additional information is extracted.
// This function pushes to the `toIndex` channel if the repo is valid.
func (app *App) scrapeRepo(ctx context.Context, repo github.Repository) (err error) {
	meta := versioning.DependencyMeta{
		User: repo.Owner.GetLogin(),
		Repo: repo.GetName(),
	}

	if meta.User == "" || meta.Repo == "" {
		return errors.New("repository details empty")
	}

	pkg, err := types.GetRemotePackage(ctx, app.gh, meta)
	if err != nil {
		var files []*github.RepositoryContent
		_, files, _, err = app.gh.Repositories.GetContents(
			ctx,
			meta.User,
			meta.Repo,
			"/",
			&github.RepositoryContentGetOptions{})
		if err != nil {
			return
		}

		valid := false
		for _, file := range files {
			ext := filepath.Ext(file.GetName())
			if ext == ".inc" || ext == ".pwn" {
				valid = true
				break
			}
		}
		if !valid {
			logger.Debug("package does not contain pawn source at top level",
				zap.String("meta", fmt.Sprint(meta)))
			return
		}

		logger.Debug("scraped non-package pawn repository",
			zap.String("meta", fmt.Sprint(meta)))

		app.toIndex <- &pkg
	} else {
		logger.Debug("scraped valid pawn package",
			zap.String("meta", fmt.Sprint(meta)))

		app.toIndex <- &types.Package{DependencyMeta: meta}
	}

	return
}
