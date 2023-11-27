package internal

import (
	"context"
	"math/rand"
	"slices"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/samshadwell/split-ynab/internal/storage"
	"github.com/samshadwell/split-ynab/internal/ynab"
	"go.uber.org/zap"
)

type splitTransaction struct {
	transaction   *ynab.TransactionDetail
	pctTheirShare int
}

func Run(ctx context.Context, logger *zap.Logger, cfg *config, storageAdapter storage.StorageAdapter) error {
	client, err := ynab.NewYnabAdapter(logger, cfg.YnabToken)
	if err != nil {
		return errors.Wrap(err, "failed to construct client")
	}

	// In case of error we'll process more transactions than we need to, but don't need to exit.
	logger.Info("getting last server knowledge")
	serverKnowledge, err := storageAdapter.GetLastServerKnowledge(cfg.BudgetId)
	if err != nil {
		logger.Warn("failed to get last server knowledge", zap.Error(err))
	}

	transactionsResponse, err := client.FetchTransactions(ctx, cfg.BudgetId, serverKnowledge)
	if err != nil {
		return errors.Wrap(err, "failed to fetch transactions from YNAB")
	}

	updatedServerKnowledge := transactionsResponse.JSON200.Data.ServerKnowledge
	filteredTransactions := filterTransactions(transactionsResponse.JSON200.Data.Transactions, cfg)
	logger.Info("finished filtering transactions", zap.Int("count", len(filteredTransactions)))

	if len(filteredTransactions) == 0 {
		logger.Info("no transactions to update, exiting")
		err = storageAdapter.SetLastServerKnowledge(cfg.BudgetId, updatedServerKnowledge)
		if err != nil {
			logger.Warn("failed to set new server knowledge", zap.Error(err))
		}
		return nil
	}

	updatedTransactions := splitTransactions(filteredTransactions, cfg.SplitCategoryId)

	err = client.UpdateTransactions(ctx, cfg.BudgetId, updatedTransactions)
	if err != nil {
		return errors.Wrap(err, "failed to update transactions in YNAB")
	}

	logger.Info("setting server knowledge", zap.Int64("serverKnowledge", updatedServerKnowledge))
	err = storageAdapter.SetLastServerKnowledge(cfg.BudgetId, updatedServerKnowledge)
	if err != nil {
		logger.Warn("failed to set new server knowledge", zap.Error(err))
	}

	logger.Info("run complete, program finished successfully")
	return nil
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
