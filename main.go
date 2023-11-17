package main

import (
	"fmt"
	"reflect"

	"github.com/brunomvsouza/ynab.go"
)

func main() {
	config := NewConfig()
	c := ynab.NewClient(config.YnabToken)

	transactions, _ := c.Transaction().GetTransactions(config.BudgetId, nil)
	fmt.Println(reflect.TypeOf(transactions))
	fmt.Println(len(transactions))
}
