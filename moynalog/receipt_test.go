package moynalog

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestReceiptJSON(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/receipt/770000000000/20dkx5w2wt/json", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		writeJSON(t, w, http.StatusOK, `{
			"receiptId": "20dkx5w2wt",
			"services": [{"name": "Услуга", "quantity": 1, "serviceNumber": 0, "amount": 86.28}],
			"operationTime": "2022-03-30T17:45:12Z",
			"requestTime": "2022-03-30T17:45:12Z",
			"registerTime": "2022-03-30T17:45:13.286887Z",
			"taxPeriodId": 202203,
			"paymentType": "CASH",
			"incomeType": "FROM_INDIVIDUAL",
			"totalAmount": 86.28,
			"cancellationInfo": null,
			"inn": "770000000000",
			"profession": "IT",
			"description": []
		}`)
	})

	receipt, _, err := client.Receipt.JSON(context.Background(), "20dkx5w2wt")
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	if receipt.ReceiptID != "20dkx5w2wt" {
		t.Errorf("ReceiptID = %q, want %q", receipt.ReceiptID, "20dkx5w2wt")
	}
	if !receipt.TotalAmount.Equal(decimal.NewFromFloat(86.28)) {
		t.Errorf("TotalAmount = %s, want 86.28", receipt.TotalAmount)
	}
	if receipt.Cancelled() {
		t.Error("a receipt without cancellationInfo must not read as cancelled")
	}
}

// Without a profile in the token the INN has to be fetched before the receipt.
func TestReceiptJSONFetchesINNWhenMissing(t *testing.T) {
	t.Parallel()

	client, mux := setup(t)
	client = client.WithToken(&AccessToken{Token: "access"})

	var userCalls int
	mux.HandleFunc("/v1/user", func(w http.ResponseWriter, _ *http.Request) {
		userCalls++
		writeJSON(t, w, http.StatusOK, `{"id":1,"inn":"770000000001","displayName":"ПУПКИН"}`)
	})
	mux.HandleFunc("/v1/receipt/770000000001/uuid/json", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"receiptId":"uuid"}`)
	})

	receipt, _, err := client.Receipt.JSON(context.Background(), "uuid")
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if receipt.ReceiptID != "uuid" {
		t.Errorf("ReceiptID = %q, want %q", receipt.ReceiptID, "uuid")
	}
	if userCalls != 1 {
		t.Errorf("user endpoint called %d times, want 1", userCalls)
	}
}

func TestReceiptPrintURL(t *testing.T) {
	t.Parallel()

	client, _ := setupAuthed(t)

	got, err := client.Receipt.PrintURL(context.Background(), "20dkx5w2wt")
	if err != nil {
		t.Fatalf("PrintURL: %v", err)
	}

	want := client.BaseURL.String() + "receipt/770000000000/20dkx5w2wt/print"
	if got != want {
		t.Errorf("PrintURL = %q, want %q", got, want)
	}
}

func TestReceiptPrint(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/receipt/770000000000/uuid/print", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Header().Set("Content-Type", "application/pdf")
		if _, err := w.Write([]byte("%PDF-1.4 fake")); err != nil {
			t.Fatalf("write response: %v", err)
		}
	})

	pdf, _, err := client.Receipt.Print(context.Background(), "uuid")
	if err != nil {
		t.Fatalf("Print: %v", err)
	}
	if !strings.HasPrefix(string(pdf), "%PDF") {
		t.Errorf("Print returned %q, want PDF bytes", pdf)
	}
}

func TestReceiptRejectsEmptyUUID(t *testing.T) {
	t.Parallel()

	client := setupNoRequest(t)

	_, _, err := client.Receipt.JSON(context.Background(), "")
	assertLocalError(t, err)

	_, err = client.Receipt.PrintURL(context.Background(), "")
	assertLocalError(t, err)

	_, _, err = client.Receipt.Print(context.Background(), "")
	assertLocalError(t, err)
}

// A profile without an INN cannot address any receipt endpoint.
func TestReceiptRequiresProfileINN(t *testing.T) {
	t.Parallel()

	client, mux := setup(t)
	client = client.WithToken(&AccessToken{Token: "access"})
	mux.HandleFunc("/v1/user", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"id":1,"inn":"","displayName":"ПУПКИН"}`)
	})
	mux.HandleFunc("/v1/receipt/", func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("no receipt request should have been sent, got %s", r.URL)
	})

	_, _, err := client.Receipt.JSON(context.Background(), "uuid")
	assertLocalError(t, err)
}
