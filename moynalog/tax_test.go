package moynalog

import (
	"context"
	"net/http"
	"testing"

	"github.com/shopspring/decimal"
)

func TestTaxGet(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/taxes", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		writeJSON(t, w, http.StatusOK, `{
			"totalForPayment": 1200.5,
			"total": 1200.5,
			"tax": 1000,
			"debt": 0,
			"overpayment": 0,
			"penalty": 0,
			"nominalTax": 1000,
			"nominalOverpayment": 0,
			"taxPeriodId": 202305,
			"lastPaymentAmount": null,
			"lastPaymentDate": "2023-12-03",
			"regions": []
		}`)
	})

	tax, _, err := client.Tax.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if !tax.TotalForPayment.Equal(decimal.NewFromFloat(1200.5)) {
		t.Errorf("TotalForPayment = %s, want 1200.5", tax.TotalForPayment)
	}
	if tax.TaxPeriodID != 202305 {
		t.Errorf("TaxPeriodID = %d, want 202305", tax.TaxPeriodID)
	}
	// A bare date must decode just as well as a full timestamp.
	if tax.LastPaymentDate.IsZero() {
		t.Error("LastPaymentDate must be decoded from a bare date")
	}
	if !tax.LastPaymentAmount.IsZero() {
		t.Errorf("LastPaymentAmount = %s, want the zero value for null", tax.LastPaymentAmount)
	}
}

func TestTaxHistory(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/taxes/history", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		if got := testBody(t, r)["oktmo"]; got != "260000" {
			t.Errorf("oktmo = %v, want %q", got, "260000")
		}

		writeJSON(t, w, http.StatusOK, `{"records":[{
			"taxPeriodId": 202211,
			"taxAmount": 12.00,
			"bonusAmount": 12.33,
			"paidAmount": 44.23,
			"taxBaseAmount": 12.23,
			"chargeDate": "2022-11-12",
			"dueDate": "2022-12-11",
			"oktmo": "260000",
			"regionName": "Калининградская область",
			"kbk": "",
			"taxOrganCode": "",
			"type": "",
			"krsbTaxChargeId": 0,
			"receiptCount": 0
		}]}`)
	})

	records, _, err := client.Tax.History(context.Background(), "260000")
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	record := records[0]
	if record.TaxPeriodID != 202211 {
		t.Errorf("TaxPeriodID = %d, want 202211", record.TaxPeriodID)
	}
	if !record.BonusAmount.Equal(decimal.NewFromFloat(12.33)) {
		t.Errorf("BonusAmount = %s, want 12.33", record.BonusAmount)
	}
	if record.RegionName != "Калининградская область" {
		t.Errorf("RegionName = %q, want the region name", record.RegionName)
	}
	if record.ChargeDate.IsZero() || record.DueDate.IsZero() {
		t.Error("ChargeDate and DueDate must be decoded")
	}
}

// An empty OKTMO must go out as null, which the API reads as "every region".
func TestTaxHistoryOmittedOktmoIsNull(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/taxes/history", func(w http.ResponseWriter, r *http.Request) {
		body := testBody(t, r)
		if _, present := body["oktmo"]; !present {
			t.Error("oktmo must be present in the body")
		}
		if body["oktmo"] != nil {
			t.Errorf("oktmo = %v, want null", body["oktmo"])
		}
		writeJSON(t, w, http.StatusOK, `{"records":[]}`)
	})

	if _, _, err := client.Tax.History(context.Background(), ""); err != nil {
		t.Fatalf("History: %v", err)
	}
}

func TestTaxPayments(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/taxes/payments", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)

		body := testBody(t, r)
		if body["onlyPaid"] != true {
			t.Errorf("onlyPaid = %v, want true", body["onlyPaid"])
		}

		writeJSON(t, w, http.StatusOK, `{"records":[{
			"sourceType": "MANUAL",
			"type": "TAX",
			"documentIndex": "18201",
			"amount": 44.23,
			"operationDate": "2022-11-12",
			"dueDate": "2022-11-12",
			"oktmo": "260000",
			"kbk": "182",
			"status": "PAID",
			"taxPeriodId": 202211,
			"regionName": "Калининградская область",
			"krsbAcceptedDate": "2022-11-12"
		}]}`)
	})

	payments, _, err := client.Tax.Payments(context.Background(), "", true)
	if err != nil {
		t.Fatalf("Payments: %v", err)
	}
	if len(payments) != 1 {
		t.Fatalf("got %d payments, want 1", len(payments))
	}

	payment := payments[0]
	if !payment.Amount.Equal(decimal.NewFromFloat(44.23)) {
		t.Errorf("Amount = %s, want 44.23", payment.Amount)
	}
	if payment.Status != "PAID" {
		t.Errorf("Status = %q, want %q", payment.Status, "PAID")
	}
	if payment.KrsbAcceptedDate.IsZero() {
		t.Error("KrsbAcceptedDate must be decoded")
	}
}
