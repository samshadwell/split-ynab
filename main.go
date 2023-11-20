package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"slices"

	"github.com/google/uuid"
	"github.com/samshadwell/split-ynab/storage"
	"github.com/samshadwell/split-ynab/ynab"
	"go.uber.org/zap"
)

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

	client, err := ynab.NewYnabAdapter(logger, config.YnabToken)
	if err != nil {
		logger.Error("failed to construct client", zap.Error(err))
		os.Exit(1)
	}

	ctx := context.Background()

	storage := storage.NewLocalStorageAdapter()
	// In case of error we'll process more transactions than we need to, but don't need to exit.
	serverKnowledge, _ := storage.GetLastServerKnowledge(config.BudgetId)

	transactionsResponse, err := client.FetchTransactions(ctx, config.BudgetId, serverKnowledge)
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

	updatedTransactions := splitTransactions(filteredTransactions, config.SplitCategoryId)

	err = client.UpdateTransactions(ctx, config.BudgetId, updatedTransactions)
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

func filterTransactions(transactions []ynab.TransactionDetail, cfg *config) []ynab.TransactionDetail {
	var filtered []ynab.TransactionDetail
OUTER:
	for _, t := range transactions {
		if t.Deleted ||
			t.Amount == 0 ||
			t.CategoryId == nil || // Example: credit card payments
			*t.CategoryId == cfg.SplitCategoryId || // Don't re-split already-split transactions
			len(t.Subtransactions) != 0 || // Don't re-split already-split transactions
			t.Cleared == ynab.Reconciled {
			continue
		}

		var flagColor ynab.TransactionFlagColor
		if t.FlagColor == nil {
			flagColor = ynab.TransactionFlagColorNil
		} else {
			flagColor = *t.FlagColor
		}

		for _, splitAcct := range cfg.SplitAccounts {
			if splitAcct.Id != t.AccountId {
				continue
			}

			if len(splitAcct.ExceptFlags) == 0 || !slices.Contains(splitAcct.ExceptFlags, flagColor) {
				filtered = append(filtered, t)
				continue OUTER // Avoid appending again if another condition is satisfied
			}
		}

		if len(cfg.SplitFlags) != 0 && slices.Contains(cfg.SplitFlags, flagColor) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func splitTransactions(transactions []ynab.TransactionDetail, splitCategoryId uuid.UUID) []ynab.SaveTransactionWithId {
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
					CategoryId: &splitCategoryId,
				},
			},
		}
	}

	return split
}
