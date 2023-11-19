package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/samshadwell/split-ynab/storage"
	"github.com/samshadwell/split-ynab/ynab"
)

const ynabServer = "https://api.ynab.com/v1"

func main() {
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	client, err := constructClient(config.YnabToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while constructing client: %v\n", err)
		os.Exit(1)
	}

	storage := storage.NewLocalStorageAdapter()
	// Ignore error return, we can use the default value of 0 in case of error.
	serverKnowledge, _ := storage.GetLastServerKnowledge(config.BudgetId)

	transactionsResponse, err := client.GetTransactionsByAccountWithResponse(
		context.TODO(),
		config.BudgetId,
		config.SplitAccountIds[0],
		&ynab.GetTransactionsByAccountParams{
			LastKnowledgeOfServer: &serverKnowledge,
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while getting transactions: %v\n", err)
		os.Exit(1)
	}

	newKnowledge := transactionsResponse.JSON200.Data.ServerKnowledge
	fmt.Printf("Transaction count: %v, New server knowledge: %v\n", len(transactionsResponse.JSON200.Data.Transactions), newKnowledge)
	err = storage.SetLastServerKnowledge(config.BudgetId, newKnowledge)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while setting new server knowledge: %v\n", err)
		os.Exit(1)
	}
}

func constructClient(authToken string) (*ynab.ClientWithResponses, error) {
	authRequestEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))
		return nil
	}
	return ynab.NewClientWithResponses(ynabServer, ynab.WithRequestEditorFn(authRequestEditor))
}
