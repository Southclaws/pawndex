package searcher

import (
	"context"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

type Searcher interface {
	Search(string) ([]github.Repository, error)
}

type GitHubSearcher struct {
	gh *github.Client
}

func (g *GitHubSearcher) Search(query string) (repos []github.Repository, err error) {
	page := 0
	for {
		r, err := g.runQueryForPage(query, page + 1)
		if err != nil {
			return nil, err
		}
		if len(r) == 0{
			break
		}

		repos = append(repos, r...)
	}
	return
}

func (g *GitHubSearcher) runQueryForPage(query string, page int) (repos []github.Repository, err error) {
	results, _, err := g.gh.Search.Repositories(
		context.Background(),
		query,
		&github.SearchOptions{ListOptions: github.ListOptions{
			Page:    page,
			PerPage: 100,
		}})
	if err != nil {
		return nil, errors.Wrap(err, "failed to search repositories")
	}

	return results.Repositories, nil
}