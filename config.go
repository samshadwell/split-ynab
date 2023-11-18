package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type config struct {
	YnabToken       string   `yaml:"ynabToken"`
	BudgetId        string   `yaml:"budgetId"`
	SplitAccountIds []string `yaml:"splitAccountIds"`
}

func LoadConfig() (*config, error) {
	f, err := os.Open("config.yml")
	if err != nil {
		return nil, fmt.Errorf("Error opening config file. Did you create a config.yml?\n\t%w\n", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	var cfg config
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf(
			"Error decoding config file. Make sure it has the correct format. See config.yml.example for an example\n\t%w\n",
			err,
		)
	}

	if len(cfg.YnabToken) == 0 {
		return nil, fmt.Errorf("Error: config file is missing ynabToken")
	}

	if len(cfg.BudgetId) == 0 {
		return nil, fmt.Errorf("Error: config file is missing budgetId")
	}

	return &cfg, nil
}
