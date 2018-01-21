package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Southclaws/sampctl/types"
	"github.com/Southclaws/sampctl/versioning"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
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
		var ref *github.Reference

		ref, _, err = app.gh.Git.GetRef(ctx, meta.User, meta.Repo, "heads/"+repo.GetDefaultBranch())
		if err != nil {
			return errors.Wrap(err, "failed to get HEAD ref from default branch")
		}

		sha := ref.GetObject().GetSHA()

		var tree *github.Tree
		tree, _, err = app.gh.Git.GetTree(ctx, meta.User, meta.Repo, sha, true)
		if err != nil {
			return errors.Wrap(err, "failed to get git tree")
		}

		valid := false
		for _, file := range tree.Entries {
			ext := filepath.Ext(file.GetPath())
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

		app.toIndex <- &types.Package{DependencyMeta: meta}
	} else {
		pkg.User = repo.GetOwner().GetLogin()
		pkg.Repo = repo.GetName()

		logger.Debug("scraped valid pawn package",
			zap.String("meta", fmt.Sprint(meta)))

		app.toIndex <- &pkg
	}

	return
}
