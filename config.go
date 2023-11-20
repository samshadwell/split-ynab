package main

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/samshadwell/split-ynab/ynab"
	"gopkg.in/yaml.v3"
)

type splitAccount struct {
	Id          uuid.UUID                   `yaml:"id"`
	ExceptFlags []ynab.TransactionFlagColor `yaml:"exceptFlags"`
}

type config struct {
	YnabToken       string                      `yaml:"ynabToken"`
	BudgetId        uuid.UUID                   `yaml:"budgetId"`
	SplitCategoryId uuid.UUID                   `yaml:"splitCategoryId"`
	SplitAccounts   []splitAccount              `yaml:"splitAccounts"`
	SplitFlags      []ynab.TransactionFlagColor `yaml:"splitFlags"`
}

func LoadConfig() (*config, error) {
	f, err := os.Open("config.yml")
	if err != nil {
		return nil, fmt.Errorf("error opening config file. Did you create a config.yml?\n\t%w\n", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	var cfg config
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding config file. Make sure it has the correct format. See config.yml.example for an example\n\t%w\n",
			err,
		)
	}

	missingFields := make([]string, 0)
	fmt.Println(cfg.SplitCategoryId)
	if len(cfg.YnabToken) == 0 {
		missingFields = append(missingFields, "ynabToken")
	}
	if cfg.BudgetId == uuid.Nil {
		missingFields = append(missingFields, "budgetId")
	}
	if cfg.SplitCategoryId == uuid.Nil {
		missingFields = append(missingFields, "splitCategoryId")
	}

	if len(missingFields) > 0 {
		return nil, fmt.Errorf("missing required fields in config file: %v", missingFields)
	}

	// Doesn't seem like there's a better way than enumerating these by hand
	validColors := map[ynab.TransactionFlagColor]bool{
		ynab.TransactionFlagColorBlue:   true,
		ynab.TransactionFlagColorGreen:  true,
		ynab.TransactionFlagColorNil:    true,
		ynab.TransactionFlagColorOrange: true,
		ynab.TransactionFlagColorPurple: true,
		ynab.TransactionFlagColorRed:    true,
		ynab.TransactionFlagColorYellow: true,
	}

	for _, acct := range cfg.SplitAccounts {
		if acct.Id == uuid.Nil {
			return nil, fmt.Errorf("invalid or mal-formatted `id` in splitAccounts config")
		}
		for _, flag := range acct.ExceptFlags {
			if !validColors[flag] {
				return nil, fmt.Errorf("invalid or flag color in `exceptFlags` of splitAccounts: %v", flag)
			}
		}
	}

	for _, flag := range cfg.SplitFlags {
		if !validColors[flag] {
			return nil, fmt.Errorf("invalid flag color in `splitFlags`: %v", flag)
		}
	}

	return &cfg, nil
}
