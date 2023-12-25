package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shoman4eg/go-moy-nalog/moynalog"
)

func main() {
	client := moynalog.NewClient(nil)
	token, err := client.Auth.CreateAccessToken(context.Background(), "inn", "password")
	if err != nil {
		log.Fatal(err)
	}
	if token.IsExpired() {
		token, err = client.Auth.RefreshToken(context.Background(), token)
		if err != nil {
			log.Fatal(err)
		}
	}

	client = moynalog.NewAuthClient(token)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	cancelIncome, resp, err := client.Income.Cancel(ctx, &moynalog.IncomeCancelRequest{
		Comment:     moynalog.Cancel,
		ReceiptUUID: "receiptUUID",
		PartnerCode: "",
	})
	if err != nil {
		log.Print(err)
	}

	fmt.Printf("Cancel income %+v, create income response: %+v", cancelIncome, resp)

	cancel()
}
