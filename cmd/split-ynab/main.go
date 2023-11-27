package main

import (
	"context"
	"fmt"
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
		fmt.Printf("failed to initialize logger: %v", err)
		os.Exit(1)
	}
	defer func() {
		// If this fails there's not much to do about it. This is mostly here to appease the linter.
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
