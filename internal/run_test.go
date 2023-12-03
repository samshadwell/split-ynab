package internal

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/samshadwell/split-ynab/internal/ynab"
)

func int64Less(a, b int64) bool {
	return a < b
}

func uuidLess(a, b uuid.UUID) bool {
	return a.String() < b.String()
}

func TestFilterTransactions(t *testing.T) {
	categoryId := uuid.New()
	splitAcctId1 := uuid.New()
	splitAcctId2 := uuid.New()

	splitCategory := uuid.New()
	twenty := 20
	thirty := 30
	fifty := 50
	cfg := Config{
		SplitCategoryId: splitCategory,
		Accounts: []accountConfig{
			{Id: splitAcctId1, DefaultPercentTheirShare: &twenty},
			{Id: splitAcctId2, DefaultPercentTheirShare: &thirty, ExceptFlags: []ynab.TransactionFlagColor{ynab.TransactionFlagColorRed}},
		},
		Flags: []flagConfig{
			{Color: ynab.TransactionFlagColorBlue, PercentTheirShare: &fifty},
			{Color: ynab.TransactionFlagColorPurple, PercentTheirShare: &thirty},
		},
	}

	blueFlag := ynab.TransactionFlagColorBlue
	greenFlag := ynab.TransactionFlagColorGreen
	purpleFlag := ynab.TransactionFlagColorPurple
	redFlag := ynab.TransactionFlagColorRed

	type testCase struct {
		shouldKeep     bool
		wantTheirShare int
		transaction    ynab.TransactionDetail
	}
	testCases := []testCase{
		// In a split account
		{
			shouldKeep:     true,
			wantTheirShare: 20,
			transaction: ynab.TransactionDetail{
				Id:         "00000000-0000-0000-0000-000000000001",
				AccountId:  splitAcctId1,
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// In a split account, with amount override flag
		{
			shouldKeep:     true,
			wantTheirShare: 50,
			transaction: ynab.TransactionDetail{
				Id:         "00000000-0000-0000-0000-000000000002",
				AccountId:  splitAcctId1,
				FlagColor:  &blueFlag,
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// In split account, with excluded flag
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         "00000000-0000-0000-0000-000000000003",
				AccountId:  splitAcctId2,
				FlagColor:  &redFlag,
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// In split account, does not have excluded flag
		{
			shouldKeep:     true,
			wantTheirShare: 30,
			transaction: ynab.TransactionDetail{
				Id:         "00000000-0000-0000-0000-000000000004",
				AccountId:  splitAcctId2,
				FlagColor:  &greenFlag,
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// Not in split account, no included flag
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         "00000000-0000-0000-0000-000000000005",
				AccountId:  uuid.New(),
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// Not in split account, but has included flag
		{
			shouldKeep:     true,
			wantTheirShare: 50,
			transaction: ynab.TransactionDetail{
				Id:         "00000000-0000-0000-0000-000000000006",
				AccountId:  uuid.New(),
				FlagColor:  &blueFlag,
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// Not in split account, other included flag
		{
			shouldKeep:     true,
			wantTheirShare: 30,
			transaction: ynab.TransactionDetail{
				Id:         "00000000-0000-0000-0000-000000000007",
				AccountId:  uuid.New(),
				FlagColor:  &purpleFlag,
				Amount:     -10_000,
				CategoryId: &categoryId,
			},
		},
		// Zero-value transaction
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         "00000000-0000-0000-0000-000000000008",
				AccountId:  splitAcctId1,
				Amount:     0,
				CategoryId: &categoryId,
			},
		},
		// Missing category (like credit card payment)
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         "00000000-0000-0000-0000-000000000009",
				AccountId:  splitAcctId1,
				Amount:     -10_000,
				CategoryId: nil,
			},
		},
		// Category is the split category
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         "00000000-0000-0000-0000-00000000000a",
				AccountId:  splitAcctId1,
				Amount:     -10_000,
				CategoryId: &splitCategory,
			},
		},
		// Already has subtransactions
		{
			shouldKeep: false,
			transaction: ynab.TransactionDetail{
				Id:         "00000000-0000-0000-0000-00000000000b",
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
				Id:         "00000000-0000-0000-0000-00000000000c",
				AccountId:  splitAcctId1,
				Amount:     -10_000,
				CategoryId: &categoryId,
				Cleared:    ynab.Reconciled,
			},
		},
	}

	type idTheirSharePairs struct {
		Id            string
		PctTheirShare int
	}
	want := make([]idTheirSharePairs, 0)
	for _, tc := range testCases {
		if tc.shouldKeep {
			want = append(want, idTheirSharePairs{tc.transaction.Id, tc.wantTheirShare})
		}
	}

	transactions := make([]ynab.TransactionDetail, len(testCases))
	for i, tc := range testCases {
		transactions[i] = tc.transaction
	}

	got := filterTransactions(transactions, &cfg)
	gotPairs := make([]idTheirSharePairs, len(got))
	for i, t := range got {
		gotPairs[i] = idTheirSharePairs{t.transaction.Id, t.pctTheirShare}
	}

	if diff := cmp.Diff(want, gotPairs); diff != "" {
		t.Fatalf("filtered transactions did not match expected. Diff (-want +got):\n%s", diff)
	}
}

func TestSplitTransactionsEvenSplit(t *testing.T) {
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
		originalTransactions := []splitTransaction{
			{
				transaction: &ynab.TransactionDetail{
					Id:         id,
					Amount:     tc.amount,
					CategoryId: &originalCategory,
				},
				pctTheirShare: 50,
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

func TestSplitTransactionsUnevenSplit(t *testing.T) {
	id := uuid.New().String()
	splitCategory := uuid.New()
	originalCategory := uuid.New()
	originalTransactions := []splitTransaction{
		{
			transaction: &ynab.TransactionDetail{
				Id:         id,
				Amount:     -10_000,
				CategoryId: &originalCategory,
			},
			pctTheirShare: 30,
		},
	}

	got := splitTransactions(originalTransactions, splitCategory)

	var gotTheirAmount, gotOurAmount int64
	for i, sub := range *got[0].Subtransactions {
		if i > 1 {
			t.Fatalf("want 2 subtransactions, got %d", len(*got[0].Subtransactions))
		}

		if *sub.CategoryId == splitCategory {
			gotTheirAmount = sub.Amount
		} else {
			gotOurAmount = sub.Amount
		}
	}

	if gotTheirAmount != -3_000 {
		t.Fatalf("want their amount to be -3_000, got %d", gotTheirAmount)
	}

	if gotOurAmount != -7_000 {
		t.Fatalf("want our amount to be -7_000, got %d", gotOurAmount)
	}
}

func TestSplitTransactionsUnevenSplitWithRemainder(t *testing.T) {
	id := uuid.New().String()
	splitCategory := uuid.New()
	originalCategory := uuid.New()
	originalTransactions := []splitTransaction{
		{
			transaction: &ynab.TransactionDetail{
				Id:         id,
				Amount:     -10_010, // $10.01, ideal split is $7.007 and $3.003
				CategoryId: &originalCategory,
			},
			pctTheirShare: 30,
		},
	}

	got := splitTransactions(originalTransactions, splitCategory)

	var gotTheirAmount, gotOurAmount int64
	for i, sub := range *got[0].Subtransactions {
		if i > 1 {
			t.Fatalf("want 2 subtransactions, got %d", len(*got[0].Subtransactions))
		}

		if *sub.CategoryId == splitCategory {
			gotTheirAmount = sub.Amount
		} else {
			gotOurAmount = sub.Amount
		}
	}

	if gotTheirAmount != -3_000 && gotTheirAmount != -3_010 {
		t.Fatalf("want their amount to be either -3_000 or -3_010, got %d", gotTheirAmount)
	}

	if gotOurAmount != -7_000 && gotOurAmount != -7_010 {
		t.Fatalf("want our amount to be either -7_000 or -7_010, got %d", gotOurAmount)
	}

	if gotTheirAmount+gotOurAmount != -10_010 {
		t.Fatalf("want total amount to be -10_010, got %d", gotTheirAmount+gotOurAmount)
	}
}
