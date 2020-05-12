package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload" // loads environment variables from .env
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Southclaws/pawndex/service"
)

var version string

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
	go func() { errs <- svc.Start(ctx) }()

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

func init() {
	godotenv.Load(".env")

	prod, err := strconv.ParseBool(os.Getenv("PRODUCTION"))
	if _, ok := err.(*strconv.NumError); !ok {
		fmt.Println(err)
		os.Exit(1)
	}

	var config zap.Config
	if prod {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(os.Getenv("LOG_LEVEL"))); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	config.Level.SetLevel(level)
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)

	if !prod {
		zap.L().Info("logger configured in development mode", zap.String("level", level.String()))
	}
}
