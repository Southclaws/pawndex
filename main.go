package main

import (
	"os"
	"time"

	// loads environment variables from .env
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func init() {
	var config zap.Config
	debug := os.Getenv("DEBUG")

	if os.Getenv("TESTING") != "" {
		config = zap.NewDevelopmentConfig()
		config.DisableCaller = true
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.MessageKey = "@message"
		config.EncoderConfig.TimeKey = "@timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

		if debug != "0" && debug != "" {
			dyn := zap.NewAtomicLevel()
			dyn.SetLevel(zap.DebugLevel)
			config.Level = dyn
		}
	}
	_logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	logger = _logger.With(
		zap.String("@version", os.Getenv("GIT_HASH")),
		zap.Namespace("@fields"),
	)
}

// Config stores static configuration
type Config struct {
	GithubToken    string        // GitHub API token
	SearchInterval time.Duration `required:"true"` // interval between checks
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
