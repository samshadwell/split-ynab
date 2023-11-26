package main

import (
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/samshadwell/split-ynab/ynab"
	"gopkg.in/yaml.v3"
)

type accountConfig struct {
	Id                       uuid.UUID                   `yaml:"id"`
	ExceptFlags              []ynab.TransactionFlagColor `yaml:"exceptFlags"`
	DefaultPercentTheirShare *int                        `yaml:"defaultPercentTheirShare"`
}

type flagConfig struct {
	Color             ynab.TransactionFlagColor `yaml:"color"`
	PercentTheirShare *int                      `yaml:"percentTheirShare"`
}

type config struct {
	YnabToken       string          `yaml:"ynabToken"`
	BudgetId        uuid.UUID       `yaml:"budgetId"`
	SplitCategoryId uuid.UUID       `yaml:"splitCategoryId"`
	Accounts        []accountConfig `yaml:"accounts"`
	Flags           []flagConfig    `yaml:"flags"`
}

func LoadConfig(reader io.Reader) (*config, error) {
	decoder := yaml.NewDecoder(reader)
	var cfg config
	err := decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf(
			"error decoding config file. Make sure it has the correct format. See README.md for an example\n\t%w\n",
			err,
		)
	}

	err = cfg.validate()
	if err != nil {
		return nil, err
	}

	cfg.setDefaults()

	return &cfg, nil
}

func (cfg *config) validate() error {
	missingFields := make([]string, 0)
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
		return fmt.Errorf("missing required fields: %v", missingFields)
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

	for idx, acct := range cfg.Accounts {
		if acct.Id == uuid.Nil {
			return fmt.Errorf("invalid or mal-formatted `id` in `accounts` at index %v", idx)
		}
		for _, flag := range acct.ExceptFlags {
			if !validColors[flag] {
				return fmt.Errorf("invalid flag color in `exceptFlags` of account: %v", flag)
			}
		}
		if acct.DefaultPercentTheirShare != nil {
			pctOwed := *acct.DefaultPercentTheirShare
			if pctOwed < 1 || pctOwed > 99 {
				return fmt.Errorf("invalid `defaultPercentTheirShare` of account. Must be between 1 and 99, inclusive: %v", pctOwed)
			}
		}
	}

	for _, flag := range cfg.Flags {
		if !validColors[flag.Color] {
			return fmt.Errorf("invalid flag color in `flags`: %v", flag)
		}
		if flag.PercentTheirShare != nil {
			pctOwed := *flag.PercentTheirShare
			if pctOwed < 1 || pctOwed > 99 {
				return fmt.Errorf("invalid `percentTheirShare`, must be between 1 and 99, inclusive: %v", pctOwed)
			}
		}
	}

	if len(cfg.Accounts) == 0 && len(cfg.Flags) == 0 {
		return fmt.Errorf("config must have at least one of either account or flag")
	}

	return nil
}

func (cfg *config) setDefaults() {
	fifty := new(int)
	*fifty = 50
	for i, acct := range cfg.Accounts {
		if acct.DefaultPercentTheirShare == nil {
			cfg.Accounts[i].DefaultPercentTheirShare = fifty
		}
	}

	for i, flag := range cfg.Flags {
		if flag.PercentTheirShare == nil {
			cfg.Flags[i].PercentTheirShare = fifty
		}
	}
}
