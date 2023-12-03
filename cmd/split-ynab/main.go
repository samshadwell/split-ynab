package main

import (
	"context"
	"log"
	"os"

	"github.com/samshadwell/split-ynab/internal"
	"github.com/samshadwell/split-ynab/internal/storage"
	"go.uber.org/zap"
)

const configFile = "config.yml"

func main() {
	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() {
		// Ignore any sync errors locally
		_ = logger.Sync()
	}()

	f, err := os.Open(configFile)
	if err != nil {
		logger.Fatal("failed to open config file", zap.Error(err))
	}

	config, err := internal.LoadConfig(f)
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	storageAdapter := storage.NewLocalStorageAdapter()

	err = internal.Run(ctx, logger, config, storageAdapter)
	if err != nil {
		logger.Fatal("program did not run successfully", zap.Error(err))
	}
}
