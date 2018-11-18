package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"

	_ "github.com/joho/godotenv/autoload" // loads environment variables from .env
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Southclaws/pawndex/service"
)

var version string

func init() {
	// constructs a logger and replaces the default global logger
	var config zap.Config
	if d, e := strconv.ParseBool(os.Getenv("DEVELOPMENT")); d && e == nil {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.DisableStacktrace = true
	if d, e := strconv.ParseBool(os.Getenv("DEBUG")); d && e == nil {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
}

func main() {
	config := service.Config{}
	err := envconfig.Process("PAWNDEX", &config)
	if err != nil {
		zap.L().Fatal("failed to load configuration",
			zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, err := service.Initialise(ctx, config)
	if err != nil {
		zap.L().Fatal("failed to initialise", zap.Error(err))
	}

	zap.L().Info("service initialised", zap.String("version", version))

	errs := make(chan error, 1)
	go func() { errs <- svc.Start() }()

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case sig := <-s:
		err = errors.New(sig.String())
	case err = <-errs:
	}

	zap.L().Fatal("exit", zap.Error(err))
}
