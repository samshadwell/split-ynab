package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

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

	budgetsResponse, err := client.GetBudgetsWithResponse(context.TODO(), &ynab.GetBudgetsParams{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while getting budgets: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Budgets: %v\n", budgetsResponse.JSON200.Data.Budgets)
}

func constructClient(authToken string) (*ynab.ClientWithResponses, error) {
	authRequestEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))
		return nil
	}
	return ynab.NewClientWithResponses(ynabServer, ynab.WithRequestEditorFn(authRequestEditor))
}
