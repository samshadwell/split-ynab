package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"github.com/samshadwell/split-ynab/storage"
	"github.com/samshadwell/split-ynab/ynab"
	"go.uber.org/zap"
)

const ynabServer = "https://api.ynab.com/v1"

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while creating logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		err = errors.Join(err, logger.Sync())
	}()

	config, err := LoadConfig()
	if err != nil {
		logger.Error("failed to load config", zap.Error(err))
		os.Exit(1)
	}

	client, err := constructYnabClient(config.YnabToken)
	if err != nil {
		logger.Error("failed to construct client", zap.Error(err))
		os.Exit(1)
	}

	storage := storage.NewLocalStorageAdapter()
	// In case of error we'll process more transactions than we need to, but don't need to exit.
	serverKnowledge, _ := storage.GetLastServerKnowledge(config.BudgetId)

	transactionsResponse, err := fetchTransactions(logger, config.BudgetId, serverKnowledge, client)
	if err != nil {
		logger.Error("failed to fetch transactions from YNAB", zap.Error(err))
		os.Exit(1)
	}

	updatedServerKnowledge := transactionsResponse.JSON200.Data.ServerKnowledge
	filteredTransactions := filterTransactions(transactionsResponse.JSON200.Data.Transactions, config)
	logger.Info("finished filtering transactions", zap.Int("count", len(filteredTransactions)))

	if len(filteredTransactions) == 0 {
		logger.Info("no transactions to update, exiting")
		// Ignore errors since we're exiting anyway
		_ = storage.SetLastServerKnowledge(config.BudgetId, updatedServerKnowledge)
		os.Exit(0)
	}

	updatedTransactions := splitTransactions(filteredTransactions, config)

	err = updateTransactions(logger, config.BudgetId, updatedTransactions, client)
	if err != nil {
		logger.Error("failed to update transactions in YNAB", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("setting server knowledge", zap.Int64("serverKnowledge", updatedServerKnowledge))
	err = storage.SetLastServerKnowledge(config.BudgetId, updatedServerKnowledge)
	if err != nil {
		logger.Error("failed to set new server knowledge", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("run complete, program finished successfully")
}

func constructYnabClient(authToken string) (*ynab.ClientWithResponses, error) {
	authRequestEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))
		return nil
	}
	return ynab.NewClientWithResponses(ynabServer, ynab.WithRequestEditorFn(authRequestEditor))
}

func fetchTransactions(
	logger *zap.Logger,
	budgetId uuid.UUID,
	serverKnowledge int64,
	client *ynab.ClientWithResponses,
) (*ynab.GetTransactionsResponse, error) {
	logger.Info("fetching transactions from YNAB",
		zap.String("budgetId", budgetId.String()),
		zap.Int64("lastKnowledgeOfServer", serverKnowledge),
	)

	transactionParams := ynab.GetTransactionsParams{}
	if serverKnowledge == 0 {
		// If we don't have any server knowledge, only update transactions from the last 30 days
		today := time.Now()
		thirtyDaysAgo := today.AddDate(0, 0, -30)
		transactionParams.SinceDate = &types.Date{Time: thirtyDaysAgo}
	} else {
		transactionParams.LastKnowledgeOfServer = &serverKnowledge
	}

	transactionsResponse, err := client.GetTransactionsWithResponse(
		context.TODO(),
		budgetId.String(),
		&transactionParams,
	)
	if err != nil {
		return nil, err
	}
	statusCode := transactionsResponse.StatusCode()
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response from YNAB when fetching transactions: %v", statusCode)
	}

	logger.Info("successfully fetched transactions from YNAB",
		zap.Int("count", len(transactionsResponse.JSON200.Data.Transactions)),
	)

	return transactionsResponse, err
}

func filterTransactions(transactions []ynab.TransactionDetail, cfg *config) []ynab.TransactionDetail {
	var filtered []ynab.TransactionDetail
	for _, t := range transactions {
		if t.Deleted || t.Amount == 0 || t.Cleared == ynab.Reconciled || len(t.Subtransactions) != 0 {
			// Skip if zero amount, reconciled, or already split
			continue
		}

		if slices.Contains(cfg.SplitAccountIds, t.AccountId) {
			filtered = append(filtered, t)
			continue
		}
	}
	return filtered
}

func splitTransactions(transactions []ynab.TransactionDetail, cfg *config) []ynab.SaveTransactionWithId {
	split := make([]ynab.SaveTransactionWithId, len(transactions))
	for i, t := range transactions {
		// Copy to avoid pointing to the loop variable
		id := t.Id

		paidAmount := ((t.Amount / 2) / 10) * 10 // Divide then multiply to truncate to nearest cent
		owedAmount := paidAmount
		extra := t.Amount - (paidAmount + owedAmount)
		if extra != 0 {
			// Randomly assign the remainder to one of the two people
			if rand.Intn(2) == 0 {
				paidAmount += extra
			} else {
				owedAmount += extra
			}
		}

		split[i] = ynab.SaveTransactionWithId{
			Id:         &id,
			PayeeId:    t.PayeeId,
			CategoryId: nil,
			Memo:       t.Memo,
			FlagColor:  t.FlagColor,
			ImportId:   t.ImportId,
			Subtransactions: &[]ynab.SaveSubTransaction{
				{
					Amount:     paidAmount,
					CategoryId: t.CategoryId,
				},
				{
					Amount:     owedAmount,
					CategoryId: &cfg.SplitCategoryId,
				},
			},
		}
	}

	return split
}

func updateTransactions(
	logger *zap.Logger,
	budgetId uuid.UUID,
	updatedTransactions []ynab.SaveTransactionWithId,
	client *ynab.ClientWithResponses,
) error {
	logger.Info("updating transactions in YNAB")
	resp, err := client.UpdateTransactionsWithResponse(
		context.TODO(),
		budgetId.String(),
		ynab.UpdateTransactionsJSONRequestBody{
			Transactions: updatedTransactions,
		},
	)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("non-200 status code %v from YNAB when updating transactions: %v", resp.StatusCode(), resp.JSON400.Error.Detail)
	}
	logger.Info("successfully updated transactions in YNAB")
	return nil
}
