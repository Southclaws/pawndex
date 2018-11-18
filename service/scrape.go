package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/Southclaws/sampctl/types"
	"github.com/Southclaws/sampctl/versioning"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v1"
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
	pkg, err := packageFromRepo(repo, meta)
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

	if processedPackage.User == "" || processedPackage.Repo == "" {
		return errors.Errorf("processed %s package details empty for %s/%s", processedPackage.Classification, meta.User, meta.Repo)
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

// packageFromRepo attempts to get a package from the given package definition's public repo
func packageFromRepo(
	repo github.Repository,
	meta versioning.DependencyMeta,
) (pkg types.Package, err error) {
	var resp *http.Response

	resp, err = http.Get(fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/%s/pawn.json",
		meta.User, meta.Repo, *repo.DefaultBranch,
	))
	if err != nil {
		return
	}
	if resp.StatusCode == 200 {
		var contents []byte
		contents, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		err = json.Unmarshal(contents, &pkg)
		return
	}

	zap.L().Debug("repo does not contain a pawn.json",
		zap.String("meta", meta.String()))

	resp, err = http.Get(fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/%s/pawn.yaml",
		meta.User, meta.Repo, *repo.DefaultBranch,
	))
	if err != nil {
		return
	}
	if resp.StatusCode == 200 {
		var contents []byte
		contents, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		err = yaml.Unmarshal(contents, &pkg)
		return
	}

	zap.L().Debug("repo does not contain a pawn.yaml",
		zap.String("meta", meta.String()))

	return pkg, errors.New("package does not point to a valid remote package")
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
