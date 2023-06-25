package main

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/shoman4eg/go-moy-nalog/moynalog"
)

func main() {
	client := moynalog.NewClient(nil)
	token, _, err := client.Auth.CreateAccessToken(context.Background(), "inn", "password")

	client = moynalog.NewAuthClient(token)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	create, err := client.Income.Create(ctx, &moynalog.IncomeCreateRequest{
		PaymentType: moynalog.Cash,
		Client: &moynalog.IncomeClient{
			ContactPhone: "+7990000000",
			DisplayName:  "ИП Пупкин",
			IncomeType:   moynalog.Individual,
		},
		Services: []*moynalog.IncomeServiceItem{
			{
				Name:     "Test service",
				Amount:   decimal.NewFromInt(1000),
				Quantity: 10,
			},
		},
	})
	if err != nil {
		return
	}

	fmt.Printf("Create income %+v, create income response: %+v", create)

	cancel()
}
