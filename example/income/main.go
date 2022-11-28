package main

import (
	"context"
	"fmt"
	"time"

	moynalog "github.com/shoman4eg/go-moy-nalog/v1"
	"github.com/shopspring/decimal"
)

func main() {
	client := moynalog.NewClient(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	items := []moynalog.IncomeServiceItem{
		{
			Name:     "",
			Amount:   decimal.NewFromInt(1000),
			Quantity: 10,
		},
	}
	incomeClient := moynalog.IncomeClient{
		ContactPhone: "+7990000000",
		DisplayName:  "ИП Пупкин",
		IncomeType:   moynalog.Individual,
	}
	create, r, err := client.Income.Create(ctx, items, incomeClient, time.Now())
	if err != nil {
		return
	}

	fmt.Printf("Create income %+v, create income response: %+v", create, r)

	cancel()
}
