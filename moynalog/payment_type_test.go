package moynalog

import (
	"context"
	"net/http"
	"testing"
)

const paymentTypeTableResponse = `{"items":[
	{
		"id": 151293,
		"type": "ACCOUNT",
		"bankName": "АО \"SUPER BANK\"",
		"bankBik": "000000000",
		"currentAccount": "10000000000000000000",
		"corrAccount": "10000000000000000000",
		"phone": null,
		"bankId": null,
		"favorite": false,
		"availableForPa": false
	},
	{
		"id": 151294,
		"type": "ACCOUNT",
		"bankName": "АО \"OTHER BANK\"",
		"bankBik": "000000001",
		"currentAccount": "20000000000000000000",
		"corrAccount": "20000000000000000000",
		"phone": "79000000000",
		"bankId": "bank-2",
		"favorite": true,
		"availableForPa": true
	}
]}`

func TestPaymentTypeTable(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/payment-type/table", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		writeJSON(t, w, http.StatusOK, paymentTypeTableResponse)
	})

	accounts, _, err := client.PaymentType.Table(context.Background())
	if err != nil {
		t.Fatalf("Table: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("got %d accounts, want 2", len(accounts))
	}

	first := accounts[0]
	if first.ID != 151293 {
		t.Errorf("ID = %d, want 151293", first.ID)
	}
	if first.BankName != `АО "SUPER BANK"` {
		t.Errorf("BankName = %q, want the bank name", first.BankName)
	}
	if first.Phone != "" {
		t.Errorf("Phone = %q, want it empty for null", first.Phone)
	}
}

func TestPaymentTypeFavorite(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/payment-type/table", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, paymentTypeTableResponse)
	})

	favorite, _, err := client.PaymentType.Favorite(context.Background())
	if err != nil {
		t.Fatalf("Favorite: %v", err)
	}
	if favorite == nil {
		t.Fatal("Favorite returned nil, want the account marked favorite")
	}
	if favorite.ID != 151294 {
		t.Errorf("ID = %d, want 151294", favorite.ID)
	}
}

func TestPaymentTypeFavoriteNoneMarked(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/payment-type/table", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"items":[{"id":1,"favorite":false}]}`)
	})

	favorite, _, err := client.PaymentType.Favorite(context.Background())
	if err != nil {
		t.Fatalf("Favorite: %v", err)
	}
	if favorite != nil {
		t.Errorf("Favorite = %+v, want nil when none is marked", favorite)
	}
}
