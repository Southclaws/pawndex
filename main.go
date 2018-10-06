package main

import (
	"time"

	// loads environment variables from .env
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
)

// Config stores static configuration
type Config struct {
	Bind           string        `required:"true"` // bind interface
	Domain         string        `required:"true"` // public domain
	GithubToken    string        `required:"true"` // GitHub API token
	SearchInterval time.Duration `required:"true"` // interval between checks
	ScrapeInterval time.Duration `required:"true"` // interval between scrapes
	Cache          string        `required:"true"` // cache for persistence
}

func main() {
	config := Config{}
	err := envconfig.Process("PAWNDEX", &config)
	if err != nil {
		logger.Fatal("failed to load configuration",
			zap.Error(err))
	}

	Start(config)
}
