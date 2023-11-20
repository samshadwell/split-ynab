package main

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type config struct {
	YnabToken       string      `yaml:"ynabToken"`
	BudgetId        uuid.UUID   `yaml:"budgetId"`
	SplitCategoryId uuid.UUID   `yaml:"splitCategoryId"`
	SplitAccountIds []uuid.UUID `yaml:"splitAccountIds"`
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

	for _, id := range cfg.SplitAccountIds {
		if id == uuid.Nil {
			return nil, fmt.Errorf("invalid or mal-formatted UUID in splitAccountIds config")
		}
	}

	return &cfg, nil
}
