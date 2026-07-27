package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/shoman4eg/go-moy-nalog/moynalog"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := moynalog.NewClient()

	token, _, err := client.Auth.CreateAccessToken(ctx, "inn", "password")
	if err != nil {
		return errors.Wrap(err, "create access token")
	}
	client = client.WithToken(token)

	created, _, err := client.Income.Create(ctx, &moynalog.IncomeCreateRequest{
		Client: &moynalog.IncomeClient{
			ContactPhone: "+79900000000",
			DisplayName:  "ИП Пупкин",
			IncomeType:   moynalog.IncomeTypeIndividual,
		},
		Services: []moynalog.IncomeServiceItem{
			{
				Name:     "Test service",
				Amount:   decimal.NewFromInt(1000),
				Quantity: decimal.NewFromInt(10),
			},
			{
				Name:     "Test 2 service",
				Amount:   decimal.NewFromFloat(1900.33),
				Quantity: decimal.NewFromInt(10),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "create income")
	}

	fmt.Printf("Created receipt %s\n", created.ApprovedReceiptUUID)

	printURL, err := client.Receipt.PrintURL(ctx, created.ApprovedReceiptUUID)
	if err != nil {
		return errors.Wrap(err, "build print url")
	}

	fmt.Printf("Printable receipt: %s\n", printURL)

	return nil
}
