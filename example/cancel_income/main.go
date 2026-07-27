package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"

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

	// Expired tokens are refreshed automatically on the first 401, so this is
	// only worth doing when reusing a token persisted from an earlier run.
	if token.IsExpired() {
		token, _, err = client.Auth.Refresh(ctx, token)
		if err != nil {
			return errors.Wrap(err, "refresh access token")
		}
	}

	client = client.WithToken(token)

	cancelled, _, err := client.Income.Cancel(ctx, &moynalog.IncomeCancelRequest{
		ReceiptUUID: "receiptUUID",
		Comment:     moynalog.CancelCommentMistake,
	})
	if err != nil {
		return errors.Wrap(err, "cancel income")
	}

	fmt.Printf(
		"Cancelled receipt %s at %s\n",
		cancelled.ApprovedReceiptUUID,
		cancelled.CancellationInfo.RegisterTime,
	)

	return nil
}
