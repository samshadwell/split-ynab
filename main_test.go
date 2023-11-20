package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/samshadwell/split-ynab/ynab"
)

func int64Less(a, b int64) bool {
	return a < b
}

func uuidLess(a, b uuid.UUID) bool {
	return a.String() < b.String()
}

func TestFilterTransactions(t *testing.T) {
	t.Parallel()

	type testCase struct {
		shouldKeep  bool
		transaction ynab.TransactionDetail
	}

	categoryId := uuid.New()
	splitAcctId1 := uuid.New()
	splitAcctId2 := uuid.New()

	splitCategory := uuid.New()
	cfg := config{
		SplitCategoryId: splitCategory,
		SplitAccounts: []splitAccount{
			{Id: splitAcctId1},
			{Id: splitAcctId2, ExceptFlags: []ynab.TransactionFlagColor{ynab.TransactionFlagColorRed}},
		},
		SplitFlags: []ynab.TransactionFlagColor{ynab.TransactionFlagColorBlue},
	}

	redFlag := ynab.TransactionFlagColorRed
	blueFlag := ynab.TransactionFlagColorBlue

	testCases := []testCase{
		// In a split account
		{
			shouldKeep: true,
			transaction: ynab.TransactionDetail{
				Id:         uuid.New().String(),
				AccountId:  splitAcctId1,
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// In split account, with excluded flag
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         uuid.New().String(),
				AccountId:  splitAcctId2,
				FlagColor:  &redFlag,
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// In split account, does not have excluded flag
		{
			shouldKeep: true,
			transaction: ynab.TransactionDetail{
				Id:         uuid.New().String(),
				AccountId:  splitAcctId2,
				FlagColor:  &blueFlag,
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// Not in split account, no included flag
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         uuid.New().String(),
				AccountId:  uuid.New(),
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// Not in split account, but has included flag
		{
			shouldKeep: true,
			transaction: ynab.TransactionDetail{
				Id:         uuid.New().String(),
				AccountId:  uuid.New(),
				FlagColor:  &blueFlag,
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// Zero-value transaction
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         uuid.New().String(),
				AccountId:  splitAcctId1,
				Amount:     0,
				CategoryId: &categoryId,
			},
		},
		// Missing category (like credit card payment)
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         uuid.New().String(),
				AccountId:  splitAcctId1,
				Amount:     -10_000,
				CategoryId: nil,
			},
		},
		// Category is the split category
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         uuid.New().String(),
				AccountId:  splitAcctId1,
				Amount:     -10_000,
				CategoryId: &splitCategory,
			},
		},
		// Already has subtransactions
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         uuid.New().String(),
				AccountId:  splitAcctId1,
				Amount:     -10_000,
				CategoryId: &categoryId,
				Subtransactions: []ynab.SubTransaction{
					{CategoryId: &categoryId, Amount: -5_000},
					{CategoryId: &splitCategory, Amount: -5_000},
				},
			},
		},
		// Reconciled
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         uuid.New().String(),
				AccountId:  splitAcctId1,
				Amount:     -10_000,
				CategoryId: &categoryId,
				Cleared:    ynab.Reconciled,
			},
		},
	}

	wantIds := make([]string, 0)
	for _, tc := range testCases {
		if tc.shouldKeep {
			wantIds = append(wantIds, tc.transaction.Id)
		}
	}

	transactions := make([]ynab.TransactionDetail, len(testCases))
	for i, tc := range testCases {
		transactions[i] = tc.transaction
	}

	got := filterTransactions(transactions, &cfg)
	gotIds := make([]string, len(got))
	for i, t := range got {
		gotIds[i] = t.Id
	}

	if !cmp.Equal(wantIds, gotIds) {
		diff := cmp.Diff(wantIds, gotIds)
		t.Fatalf("want filtered transactions to be %v, got %v\n%s", wantIds, gotIds, diff)
	}
}

func TestSplitTransactions(t *testing.T) {
	t.Parallel()

	type testCase struct {
		amount      int64
		wantAmounts []int64
	}
	testCases := []testCase{
		// $10, splits evenly
		{amount: -10_000, wantAmounts: []int64{-5_000, -5_000}},
		// $10.01, splits into $5 and $5.01
		{amount: -10_010, wantAmounts: []int64{-5_000, -5_010}},
	}

	for _, tc := range testCases {
		id := uuid.New().String()
		splitCategory := uuid.New()
		originalCategory := uuid.New()
		originalTransactions := []ynab.TransactionDetail{
			{
				Id:         id,
				Amount:     tc.amount,
				CategoryId: &originalCategory,
			},
		}

		got := splitTransactions(originalTransactions, splitCategory)
		if len(got) != 1 {
			t.Fatalf("want 1 transaction, got %d", len(got))
		}

		split := got[0]
		if *split.Id != id {
			t.Fatalf("want Id to be %q, got %q", id, *split.Id)
		}

		if split.CategoryId != nil {
			t.Fatalf("want CategoryId to be nil, got %q", *split.CategoryId)
		}

		if len(*split.Subtransactions) != 2 {
			t.Fatalf("want 2 subtransactions, got %d", len(*split.Subtransactions))
		}

		first := (*split.Subtransactions)[0]
		second := (*split.Subtransactions)[1]

		gotAmounts := []int64{first.Amount, second.Amount}
		if !cmp.Equal(tc.wantAmounts, gotAmounts, cmpopts.SortSlices(int64Less)) {
			t.Fatalf("want subtransaction amounts to be %v, got %v", tc.wantAmounts, gotAmounts)
		}

		wantCategories := []uuid.UUID{originalCategory, splitCategory}
		gotCategories := []uuid.UUID{*first.CategoryId, *second.CategoryId}
		if !cmp.Equal(wantCategories, gotCategories, cmpopts.SortSlices(uuidLess)) {
			t.Fatalf("want categories to be %v, got %v", wantCategories, gotCategories)
		}
	}
}
