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
