package main

import (
	"fmt"
	"os"

	"github.com/brunomvsouza/ynab.go"
)

func main() {
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	client := ynab.NewClient(config.YnabToken)

	transactions, err := client.Transaction().GetTransactionsByAccount(config.BudgetId, config.SplitAccountIds[0], nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while fetching transactions from YNAB:\n\t%v\n", err)
		os.Exit(1)
	}

	fmt.Println("Server knowledge: ", transactions[0])
}
