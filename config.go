package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	YnabToken string `yaml:"ynabToken"`
	BudgetId  string `yaml:"budgetId"`
}

func NewConfig() *Config {
	f, err := os.Open("config.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening config file. Did you create a config.yml? %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding config file. Make sure it has the correct format. See config.yml.example for an example %v\n", err)
	}

	return &cfg
}
