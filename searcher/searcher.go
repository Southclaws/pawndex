package searcher

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

type Searcher interface {
	Search(string) ([]string, error)
}

type GitHubSearcher struct {
	gh *github.Client
}

func (g *GitHubSearcher) Search(query string) (repos []string, err error) {
	page := 0
	for {
		result, err := g.runQueryForPage(query, page+1)
		if err != nil {
			return nil, err
		}
		if len(result) == 0 {
			break
		}

		for _, r := range result {
			repos = append(repos, fmt.Sprintf("%s/%s", r.Owner.GetLogin(), r.GetName()))
		}
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
