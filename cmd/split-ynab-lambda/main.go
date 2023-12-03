package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/samshadwell/split-ynab/internal"
	"github.com/samshadwell/split-ynab/internal/storage"
	"go.uber.org/zap"
)

type handler struct {
	logger         *zap.Logger
	config         *internal.Config
	storageAdapter storage.StorageAdapter
}

func (h *handler) HandleLambdaEvent(ctx context.Context) error {
	return internal.Run(ctx, h.logger, h.config, h.storageAdapter)
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() {
		loggerErr := logger.Sync()
		if loggerErr != nil {
			fmt.Printf("failed to sync logger: %v", loggerErr)
		}
	}()

	// See deployments/aws/aws.go for how these are set
	configBucket := os.Getenv("CONFIG_BUCKET")
	configKey := os.Getenv("CONFIG_KEY")
	tableName := os.Getenv("TABLE_NAME")

	if configBucket == "" || configKey == "" || tableName == "" {
		logger.Fatal("missing required environment variable. All of CONFIG_BUCKET, CONFIG_KEY, and TABLE_NAME must be set",
			zap.String("CONFIG_BUCKET", configBucket),
			zap.String("CONFIG_KEY", configKey),
			zap.String("TABLE_NAME", tableName))
	}

	initContext := context.Background()

	sdkConfig, err := config.LoadDefaultConfig(initContext)
	if err != nil {
		logger.Fatal("failed to load AWS SDK config", zap.Error(err))
	}

	s3c := s3.NewFromConfig(sdkConfig)
	configFile, err := s3c.GetObject(initContext, &s3.GetObjectInput{
		Bucket: &configBucket,
		Key:    &configKey,
	})
	if err != nil {
		logger.Fatal("failed to get config file from S3", zap.Error(err))
	}

	cfg, err := internal.LoadConfig(configFile.Body)
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	storageAdapter, err := storage.NewDynamoDbStorageAdapter(logger, &sdkConfig, tableName)
	if err != nil {
		logger.Fatal("failed to initialize storage adapter", zap.Error(err))
	}

	h := &handler{
		logger:         logger,
		config:         cfg,
		storageAdapter: storageAdapter,
	}

	lambda.Start(h.HandleLambdaEvent)
}
