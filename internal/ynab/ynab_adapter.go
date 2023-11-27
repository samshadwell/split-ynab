package ynab

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"go.uber.org/zap"
)

const ynabServer = "https://api.ynab.com/v1"
const requestTimeout = 30 * time.Second

type ynabAdapter struct {
	client ClientWithResponsesInterface
	logger *zap.Logger
}

func NewYnabAdapter(logger *zap.Logger, authToken string) (*ynabAdapter, error) {
	authHeader := fmt.Sprintf("Bearer %s", authToken)
	authRequestEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Add("Authorization", authHeader)
		return nil
	}

	client, err := NewClientWithResponses(ynabServer,
		WithHTTPClient(&http.Client{Timeout: requestTimeout}),
		WithRequestEditorFn(authRequestEditor))
	if err != nil {
		return nil, err
	}

	return &ynabAdapter{
		client: client,
		logger: logger,
	}, nil
}

func (y *ynabAdapter) FetchTransactions(
	ctx context.Context,
	budgetId uuid.UUID,
	serverKnowledge int64,
) (*GetTransactionsResponse, error) {
	y.logger.Info("fetching transactions from YNAB",
		zap.String("budgetId", budgetId.String()),
		zap.Int64("lastKnowledgeOfServer", serverKnowledge),
	)

	transactionParams := GetTransactionsParams{}
	if serverKnowledge == 0 {
		// If we don't have any server knowledge, only update transactions from the last 30 days
		today := time.Now()
		thirtyDaysAgo := today.AddDate(0, 0, -30)
		transactionParams.SinceDate = &types.Date{Time: thirtyDaysAgo}
	} else {
		transactionParams.LastKnowledgeOfServer = &serverKnowledge
	}

	resp, err := y.client.GetTransactionsWithResponse(ctx, budgetId.String(), &transactionParams)
	if err != nil {
		return nil, err
	}

	statusCode := resp.StatusCode()
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code %v from YNAB when fetching transactions: %v",
			statusCode, resp.JSON400.Error.Detail)
	}

	y.logger.Info("successfully fetched transactions from YNAB",
		zap.Int("count", len(resp.JSON200.Data.Transactions)),
	)
	return resp, err
}

func (y *ynabAdapter) UpdateTransactions(
	ctx context.Context,
	budgetId uuid.UUID,
	updatedTransactions []SaveTransactionWithId,
) error {
	y.logger.Info("updating transactions in YNAB",
		zap.Int("count", len(updatedTransactions)))

	resp, err := y.client.UpdateTransactionsWithResponse(
		ctx,
		budgetId.String(),
		UpdateTransactionsJSONRequestBody{
			Transactions: updatedTransactions,
		},
	)
	if err != nil {
		return err
	}

	statusCode := resp.StatusCode()
	if statusCode != http.StatusOK {
		return fmt.Errorf("non-200 status code %v from YNAB when updating transactions: %v",
			statusCode, resp.JSON400.Error.Detail)
	}

	y.logger.Info("successfully updated transactions in YNAB")
	return nil
}
