package scraper_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Southclaws/pawndex/scraper"
	"github.com/google/go-github/github"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

func TestGitHubScraper_Scrape(t *testing.T) {
	godotenv.Load("../.env")
	token := os.Getenv("PAWNDEX_GITHUBTOKEN")
	if token == "" {
		panic("tests need a token to run.")
	}

	s := scraper.GitHubScraper{
		GitHub: github.NewClient(
			oauth2.NewClient(
				context.Background(),
				oauth2.StaticTokenSource(&oauth2.Token{
					AccessToken: token,
				}),
			)),
	}

	pkg, err := s.Scrape(context.Background(), "Southclaws/samp-logger")
	fmt.Println(err)
	fmt.Printf("%#v\n", pkg)
}
