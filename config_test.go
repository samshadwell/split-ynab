package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/samshadwell/split-ynab/ynab"
)

func TestLoadConfig(t *testing.T) {
	s := `---
ynabToken: "my-fake-token"
budgetId: "00000000-0000-0000-0000-000000000001"
splitCategoryId: "00000000-0000-0000-0000-000000000002"
accounts:
  - id: "00000000-0000-0000-0000-000000000003"
    exceptFlags: ["green"]
  - id: "00000000-0000-0000-0000-000000000004"
    defaultPercentTheirShare: 30
flags:
  - color: "orange"
  - color: "purple"
    percentTheirShare: 30
`

	got, err := LoadConfig(strings.NewReader(s))
	if err != nil {
		t.Fatalf("wanted nil error, got %v", err)
	}

	thirty := 30
	fifty := 50
	want := config{
		YnabToken:       "my-fake-token",
		BudgetId:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		SplitCategoryId: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Accounts: []accountConfig{
			{
				Id:                       uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				ExceptFlags:              []ynab.TransactionFlagColor{ynab.TransactionFlagColorGreen},
				DefaultPercentTheirShare: &fifty,
			},
			{
				Id:                       uuid.MustParse("00000000-0000-0000-0000-000000000004"),
				ExceptFlags:              nil,
				DefaultPercentTheirShare: &thirty,
			},
		},
		Flags: []flagConfig{
			{Color: ynab.TransactionFlagColorOrange, PercentTheirShare: &fifty},
			{Color: ynab.TransactionFlagColorPurple, PercentTheirShare: &thirty},
		},
	}

	if diff := cmp.Diff(&want, got); diff != "" {
		t.Errorf("config did not match expected. Diff (-want +got):\n%s", diff)
	}
}

func TestLoadConfigMissingFields(t *testing.T) {
	s := `---
ynabToken: "my-fake-token"
splitCategoryId: "00000000-0000-0000-0000-000000000002"
`

	_, err := LoadConfig(strings.NewReader(s))
	if err == nil {
		t.Fatalf("wanted error, got nil")
	}

	if !strings.Contains(err.Error(), "budgetId") {
		t.Errorf("wanted error to include missing field 'budgetId', got %v", err)
	}
}

func TestLoadConfigInvalidYml(t *testing.T) {
	s := `I'm just a text file
that contains some random words
`

	_, err := LoadConfig(strings.NewReader(s))
	if err == nil {
		t.Fatalf("wanted error, got nil")
	}
}

func TestLoadConfigInvalidColor(t *testing.T) {
	s := `---
ynabToken: "my-fake-token"
budgetId: "00000000-0000-0000-0000-000000000001"
splitCategoryId: "00000000-0000-0000-0000-000000000002"
accounts:
  - id: "00000000-0000-0000-0000-000000000003"
    exceptFlags: ["maroon"]

`
	_, err := LoadConfig(strings.NewReader(s))
	if err == nil {
		t.Fatalf("wanted error, got nil")
	}

	if !strings.Contains(err.Error(), "maroon") {
		t.Errorf("wanted error to include invalid color 'maroon', got %v", err)
	}
}

func TestLoadConfigInvalidPercentOwed(t *testing.T) {
	s := `---
ynabToken: "my-fake-token"
budgetId: "00000000-0000-0000-0000-000000000001"
splitCategoryId: "00000000-0000-0000-0000-000000000002"
accounts:
  - id: "00000000-0000-0000-0000-000000000003"
    defaultPercentTheirShare: 100

`

	_, err := LoadConfig(strings.NewReader(s))
	if err == nil {
		t.Fatalf("wanted error, got nil")
	}

	if !strings.Contains(err.Error(), "100") {
		t.Errorf("wanted error to include invalid percent their share '100', got %v", err)
	}
}
