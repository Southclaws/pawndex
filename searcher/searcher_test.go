package searcher_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-github/github"
	"github.com/joho/godotenv" // loads environment variables from .env
	"golang.org/x/oauth2"

	"github.com/Southclaws/pawndex/searcher"
)

func TestGitHubSearcher_Search(t *testing.T) {
	godotenv.Load("../.env")
	token := os.Getenv("PAWNDEX_GITHUBTOKEN")
	if token == "" {
		panic("tests need a token to run.")
	}

	s := searcher.GitHubSearcher{
		GitHub: github.NewClient(
			oauth2.NewClient(
				context.Background(),
				oauth2.StaticTokenSource(&oauth2.Token{
					AccessToken: token,
				}),
			)),
	}

	results, err := s.Search("topic:pawn-package")
	fmt.Println(err)
	fmt.Println(results)
}
