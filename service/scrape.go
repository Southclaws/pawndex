package service

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Southclaws/sampctl/types"
	"github.com/Southclaws/sampctl/versioning"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
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

	var processedPackage Package // the result - a package with some additional metadata
	pkg, err := types.GetRemotePackage(ctx, app.gh, meta)
	if err != nil {
		processedPackage, err = app.findPawnSource(ctx, repo, meta)
		if err != nil {
			return
		}
		err = nil
	} else {
		processedPackage = Package{
			Package:        pkg,
			Classification: classificationPawnPackage,
		}
	}

	if processedPackage.Classification == classificationInvalid {
		return nil
	}

	// add some generic info
	processedPackage.Stars = repo.GetStargazersCount()
	processedPackage.Updated = repo.GetUpdatedAt().Time
	processedPackage.Topics = repo.Topics

	tags, _, err := app.gh.Repositories.ListTags(ctx, meta.User, meta.Repo, nil)
	if err != nil {
		return errors.Wrap(err, "failed to list repo tags")
	}
	for _, tag := range tags {
		processedPackage.Tags = append(processedPackage.Tags, tag.GetName())
	}

	app.toIndex <- processedPackage

	return
}

func (app *App) findPawnSource(ctx context.Context, repo github.Repository, meta versioning.DependencyMeta) (pkg Package, err error) {
	ref, _, err := app.gh.Git.GetRef(ctx, meta.User, meta.Repo, fmt.Sprintf("heads/%s", repo.GetDefaultBranch()))
	if err != nil {
		err = errors.Wrap(err, "failed to get HEAD ref from default branch")
		return
	}

	sha := ref.GetObject().GetSHA()
	tree, _, err := app.gh.Git.GetTree(ctx, meta.User, meta.Repo, sha, true)
	if err != nil {
		err = errors.Wrap(err, "failed to get git tree")
		return
	}

	pkg = Package{Package: types.Package{DependencyMeta: meta}}

	for _, file := range tree.Entries {
		ext := filepath.Ext(file.GetPath())
		if ext == ".inc" || ext == ".pwn" {
			if filepath.Dir(file.GetPath()) == "." {
				pkg.Classification = classificationBarebones
				break
			} else {
				pkg.Classification = classificationBuried
				// no break, keep searching
			}
		}
	}

	return
}
