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

const configFile = "config.yml"

type splitTransaction struct {
	transaction   *ynab.TransactionDetail
	pctTheirShare int
}

func main() {
	os.Exit(mainReturnWithCode())
}

func mainReturnWithCode() int {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while creating logger: %v\n", err)
		return 1
	}
	defer func() {
		err = errors.Join(err, logger.Sync())
	}()

	f, err := os.Open(configFile)
	if err != nil {
		logger.Error("error opening config file. Did you create a config.yml?", zap.Error(err))
		return 1
	}
	defer f.Close()

	config, err := LoadConfig(f)
	if err != nil {
		logger.Error("failed to load config", zap.Error(err))
		return 1
	}

	client, err := ynab.NewYnabAdapter(logger, config.YnabToken)
	if err != nil {
		logger.Error("failed to construct client", zap.Error(err))
		return 1
	}

	ctx := context.Background()

	storage := storage.NewLocalStorageAdapter()
	// In case of error we'll process more transactions than we need to, but don't need to exit.
	serverKnowledge, _ := storage.GetLastServerKnowledge(config.BudgetId)

	transactionsResponse, err := client.FetchTransactions(ctx, config.BudgetId, serverKnowledge)
	if err != nil {
		logger.Error("failed to fetch transactions from YNAB", zap.Error(err))
		return 1
	}

	updatedServerKnowledge := transactionsResponse.JSON200.Data.ServerKnowledge
	filteredTransactions := filterTransactions(transactionsResponse.JSON200.Data.Transactions, config)
	logger.Info("finished filtering transactions", zap.Int("count", len(filteredTransactions)))

	if len(filteredTransactions) == 0 {
		logger.Info("no transactions to update, exiting")
		// Ignore errors since we're exiting anyway
		_ = storage.SetLastServerKnowledge(config.BudgetId, updatedServerKnowledge)
		return 0
	}

	updatedTransactions := splitTransactions(filteredTransactions, config.SplitCategoryId)

	err = client.UpdateTransactions(ctx, config.BudgetId, updatedTransactions)
	if err != nil {
		logger.Error("failed to update transactions in YNAB", zap.Error(err))
		return 1
	}

	logger.Info("setting server knowledge", zap.Int64("serverKnowledge", updatedServerKnowledge))
	err = storage.SetLastServerKnowledge(config.BudgetId, updatedServerKnowledge)
	if err != nil {
		logger.Error("failed to set new server knowledge", zap.Error(err))
	}

	logger.Info("run complete, program finished successfully")
	return 0
}

func filterTransactions(transactions []ynab.TransactionDetail, cfg *config) []splitTransaction {
	acctConfigs := make(map[uuid.UUID]*accountConfig, len(cfg.Accounts))
	for _, acct := range cfg.Accounts {
		copy := acct
		acctConfigs[acct.Id] = &copy
	}

	splitFlags := make(map[ynab.TransactionFlagColor]*flagConfig, len(cfg.Flags))
	for _, f := range cfg.Flags {
		copy := f
		splitFlags[f.Color] = &copy
	}

	filtered := make([]splitTransaction, 0)
	for _, t := range transactions {
		if t.Deleted ||
			t.Amount == 0 ||
			t.CategoryId == nil || // Example: credit card payments
			*t.CategoryId == cfg.SplitCategoryId || // Don't re-split already-split transactions
			len(t.Subtransactions) != 0 || // Don't re-split already-split transactions
			t.Cleared == ynab.Reconciled {
			continue
		}

		shouldAdd := false
		theirShare := 0

		var flagColor ynab.TransactionFlagColor
		if t.FlagColor == nil {
			flagColor = ynab.TransactionFlagColorNil
		} else {
			flagColor = *t.FlagColor
		}

		acctConfig := acctConfigs[t.AccountId]
		if acctConfig != nil {
			if len(acctConfig.ExceptFlags) == 0 || !slices.Contains(acctConfig.ExceptFlags, flagColor) {
				shouldAdd = true
				theirShare = *acctConfig.DefaultPercentTheirShare
			}
		}

		flagConfig := splitFlags[flagColor]
		if flagConfig != nil {
			shouldAdd = true
			theirShare = *flagConfig.PercentTheirShare
		}

		if shouldAdd {
			if theirShare == 0 {
				panic("programmer error, theirShare should never be 0")
			}

			transactionCopy := t
			filtered = append(filtered, splitTransaction{
				transaction:   &transactionCopy,
				pctTheirShare: theirShare,
			})
		}
	}

	return filtered
}

func splitTransactions(transactions []splitTransaction, splitCategoryId uuid.UUID) []ynab.SaveTransactionWithId {
	split := make([]ynab.SaveTransactionWithId, len(transactions))

	for i, splitTransaction := range transactions {
		t := splitTransaction.transaction

		// Copy to avoid pointing to the loop variable
		id := t.Id

		// Use cents to avoid assigning sub-cent amounts
		totalCents := t.Amount / 10
		ourShare := totalCents * (100 - int64(splitTransaction.pctTheirShare)) / 100
		theirShare := totalCents * int64(splitTransaction.pctTheirShare) / 100

		// Turn back into milli-dollars
		ourShare *= 10
		theirShare *= 10

		extra := t.Amount - (ourShare + theirShare)
		if extra != 0 {
			// Randomly assign the remainder to one of the two people
			if rand.Intn(2) == 0 {
				ourShare += extra
			} else {
				theirShare += extra
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
					Amount:     ourShare,
					CategoryId: t.CategoryId,
				},
				{
					Amount:     theirShare,
					CategoryId: &splitCategoryId,
				},
			},
		}
	}

	return split
}
