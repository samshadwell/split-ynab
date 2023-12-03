package storage

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type dynamoDbStorageAdapter struct {
	client    dynamodb.Client
	logger    *zap.Logger
	tableName string
}

type ServerKnowledgeDocument struct {
	LastServerKnowledge int `dynamodbav:"lastServerKnowledge"`
}

func NewDynamoDbStorageAdapter(logger *zap.Logger, awsConfig *aws.Config, tableName string) (StorageAdapter, error) {
	return &dynamoDbStorageAdapter{
		client:    *dynamodb.NewFromConfig(*awsConfig),
		logger:    logger,
		tableName: tableName,
	}, nil
}

func (d *dynamoDbStorageAdapter) GetLastServerKnowledge(ctx context.Context, budgetId uuid.UUID) (int64, error) {
	d.logger.Info("getting last server knowledge from DynamoDB",
		zap.String("budgetId", budgetId.String()))
	key := serverKnowledgeKey(budgetId)
	response, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &d.tableName,
		Key:       *key,
	})

	if err != nil {
		return 0, fmt.Errorf("failed to get last server knowledge: %w", err)
	}

	responseDoc := ServerKnowledgeDocument{}
	err = attributevalue.UnmarshalMap(response.Item, &responseDoc)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	d.logger.Info("successfully retrieved last server knowledge from DynamoDB")
	return int64(responseDoc.LastServerKnowledge), nil
}

func (d *dynamoDbStorageAdapter) SetLastServerKnowledge(ctx context.Context, budgetId uuid.UUID, serverKnowledge int64) error {
	d.logger.Info("setting last server knowledge in DynamoDB",
		zap.String("budgetId", budgetId.String()),
		zap.Int64("lastServerKnowledge", serverKnowledge))

	item := *serverKnowledgeKey(budgetId)
	item["lastServerKnowledge"] = &types.AttributeValueMemberN{
		Value: fmt.Sprintf("%d", serverKnowledge),
	}

	_, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &d.tableName,
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	d.logger.Info("successfully set last server knowledge in DynamoDB")
	return nil
}

func serverKnowledgeKey(budgetId uuid.UUID) *map[string]types.AttributeValue {
	key := fmt.Sprintf("%v#SERVER_KNOWLEDGE", budgetId)
	return &map[string]types.AttributeValue{
		"key": &types.AttributeValueMemberS{
			Value: key,
		},
	}
}
