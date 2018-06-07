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

	var processedPackage *Package // the result - a package with some additional metadata

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

		pawnAtRoot := false // contains pawn files at root level (/)
		pawnAtAny := false  // contains pawn files anywhere
		for _, file := range tree.Entries {
			ext := filepath.Ext(file.GetPath())
			if ext == ".inc" || ext == ".pwn" {
				if filepath.Dir(file.GetPath()) == "." {
					pawnAtRoot = true
				} else {
					pawnAtAny = true
				}
			}
		}

		if pawnAtRoot {
			processedPackage = &Package{
				Package:        types.Package{DependencyMeta: meta},
				Classification: classificationBarebones,
			}
		} else if pawnAtAny {
			processedPackage = &Package{
				Package:        types.Package{DependencyMeta: meta},
				Classification: classificationBuried,
			}
		} else {
			logger.Debug("package does not contain pawn source",
				zap.String("meta", fmt.Sprint(meta)))
		}
	} else {
		pkg.User = repo.GetOwner().GetLogin()
		pkg.Repo = repo.GetName()

		repo.GetStargazersCount()

		processedPackage = &Package{
			Package:        pkg,
			Classification: classificationPawnPackage,
		}
	}

	processedPackage.Topics = repo.Topics

	tags, _, err := app.gh.Repositories.ListTags(ctx, meta.User, meta.Repo, &github.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list repo tags")
	}
	for _, tag := range tags {
		processedPackage.Tags = append(processedPackage.Tags, tag.GetName())
	}

	if processedPackage != nil {
		// add some generic info
		processedPackage.Stars = repo.GetStargazersCount()
		processedPackage.Updated = repo.GetUpdatedAt().Time

		app.toIndex <- processedPackage
	}

	return
}
