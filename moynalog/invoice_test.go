package moynalog

import (
	"context"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

func TestInvoiceCreate(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/invoice", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)

		body := testBody(t, r)
		// Invoices are always settled through an account.
		if body["paymentType"] != string(PaymentTypeAccount) {
			t.Errorf("paymentType = %v, want %q", body["paymentType"], PaymentTypeAccount)
		}
		if body["totalAmount"] != "89" {
			t.Errorf("totalAmount = %v, want %q", body["totalAmount"], "89")
		}

		services, ok := body["services"].([]any)
		if !ok || len(services) != 1 {
			t.Fatalf("services = %v, want one item", body["services"])
		}
		item, ok := services[0].(map[string]any)
		if !ok {
			t.Fatalf("services[0] = %v, want an object", services[0])
		}
		if _, present := item["serviceNumber"]; !present {
			t.Error("serviceNumber must be sent")
		}

		writeJSON(t, w, http.StatusOK, `{"approvedReceiptUuid":"invoice-uuid"}`)
	})

	created, _, err := client.Invoice.Create(context.Background(), &InvoiceCreateRequest{
		Services: []InvoiceServiceItem{{
			Name:     "Предоставление информационных услуг",
			Amount:   decimal.NewFromInt(89),
			Quantity: decimal.NewFromInt(1),
		}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ApprovedReceiptUUID != "invoice-uuid" {
		t.Errorf("ApprovedReceiptUUID = %q, want %q", created.ApprovedReceiptUUID, "invoice-uuid")
	}
}

func TestInvoiceCreateItem(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/invoice", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"approvedReceiptUuid":"single"}`)
	})

	created, _, err := client.Invoice.CreateItem(
		context.Background(), "Услуга", decimal.NewFromInt(10), decimal.NewFromInt(2),
	)
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if created.ApprovedReceiptUUID != "single" {
		t.Errorf("ApprovedReceiptUUID = %q, want %q", created.ApprovedReceiptUUID, "single")
	}
}

func TestInvoiceCreateValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request *InvoiceCreateRequest
	}{
		{"nil request", nil},
		{"no services", new(InvoiceCreateRequest)},
		{
			"empty name",
			&InvoiceCreateRequest{Services: []InvoiceServiceItem{{
				Amount:   decimal.NewFromInt(1),
				Quantity: decimal.NewFromInt(1),
			}}},
		},
		{
			"zero amount",
			&InvoiceCreateRequest{Services: []InvoiceServiceItem{{Name: "x", Quantity: decimal.NewFromInt(1)}}},
		},
		{
			"zero quantity",
			&InvoiceCreateRequest{Services: []InvoiceServiceItem{{Name: "x", Amount: decimal.NewFromInt(1)}}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := setupNoRequest(t)
			_, _, err := client.Invoice.Create(context.Background(), tt.request)
			assertLocalError(t, err)
		})
	}
}

// The upstream API exposes neither endpoint; both must say so explicitly.
func TestInvoiceUnimplementedEndpoints(t *testing.T) {
	t.Parallel()

	client, _ := setupAuthed(t)

	if _, err := client.Invoice.Cancel(context.Background(), 1); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("Cancel error = %v, want ErrNotImplemented", err)
	}
	if _, err := client.Invoice.UpdatePaymentInfo(context.Background()); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("UpdatePaymentInfo error = %v, want ErrNotImplemented", err)
	}
}

// The PHP client rejects a negative amount alongside the empty and zero cases.
func TestInvoiceCreateRejectsNegativeAmount(t *testing.T) {
	t.Parallel()

	client := setupNoRequest(t)

	_, _, err := client.Invoice.Create(context.Background(), &InvoiceCreateRequest{
		Services: []InvoiceServiceItem{{
			Name:     "name",
			Amount:   decimal.NewFromInt(-1),
			Quantity: decimal.NewFromInt(1),
		}},
	})
	assertLocalError(t, err)
}

func TestInvoiceServiceItemTotalAmount(t *testing.T) {
	t.Parallel()

	item := InvoiceServiceItem{
		Name:     "service",
		Amount:   decimal.RequireFromString("30.23"),
		Quantity: decimal.NewFromInt(3),
	}

	if got := item.TotalAmount().String(); got != "90.69" {
		t.Errorf("TotalAmount = %s, want 90.69", got)
	}
}

// An invoice issued to a legal entity is validated exactly like a receipt.
func TestInvoiceCreateValidatesClient(t *testing.T) {
	t.Parallel()

	validItem := InvoiceServiceItem{
		Name:     "name",
		Amount:   decimal.NewFromInt(1),
		Quantity: decimal.NewFromInt(1),
	}

	tests := []struct {
		name   string
		client *IncomeClient
	}{
		{"legal entity without INN", &IncomeClient{IncomeType: IncomeTypeLegalEntity, DisplayName: "ООО"}},
		{"legal entity with a bad INN", &IncomeClient{
			IncomeType:  IncomeTypeLegalEntity,
			Inn:         "12345",
			DisplayName: "ООО",
		}},
		{"legal entity without a display name", &IncomeClient{
			IncomeType: IncomeTypeLegalEntity,
			Inn:        "7724035047",
		}},
		{"unknown income type", &IncomeClient{IncomeType: "FROM_MARS"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := setupNoRequest(t)
			_, _, err := client.Invoice.Create(context.Background(), &InvoiceCreateRequest{
				Services: []InvoiceServiceItem{validItem},
				Client:   tt.client,
			})
			assertLocalError(t, err)
		})
	}
}
