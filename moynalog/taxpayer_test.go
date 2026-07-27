package moynalog

import (
	"context"
	"net/http"
	"testing"

	"github.com/shopspring/decimal"
)

func TestTaxpayerDebts(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/taxpayer/debts", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		writeJSON(t, w, http.StatusOK, `{"hasDebts":true,"totalUnpaid":1234.56,"debts":1000.0}`)
	})

	debts, _, err := client.Taxpayer.Debts(context.Background())
	if err != nil {
		t.Fatalf("Debts: %v", err)
	}

	if !debts.HasDebts {
		t.Error("HasDebts = false, want true")
	}
	if !debts.TotalUnpaid.Equal(decimal.NewFromFloat(1234.56)) {
		t.Errorf("TotalUnpaid = %s, want 1234.56", debts.TotalUnpaid)
	}
	if !debts.Debts.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("Debts = %s, want 1000", debts.Debts)
	}
}

func TestTaxpayerBonus(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/taxpayer/bonus", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		writeJSON(t, w, http.StatusOK, `{
			"bonusAmount": 8500,
			"totalIncomeAmount": 759100,
			"maxTotalIncomeThresholdExceeded": false,
			"annualIncomeThreshold": 2400000,
			"availableIncomeToExceedThreshold": 1640900,
			"annualIncomeStatus": "NORMAL",
			"updatedTime": "2026-07-14T21:15:27.647651Z",
			"totalIncomeByYears": {
				"2025": {
					"totalIncomeAmount": 1090000,
					"maxTotalIncomeThresholdExceeded": false,
					"annualIncomeThreshold": 2400000,
					"availableIncomeToExceedThreshold": 1310000,
					"annualIncomeStatus": "NORMAL",
					"updatedTime": "2025-12-29T00:23:34.390419Z"
				}
			},
			"teenBonusAmount": null,
			"teenBonusUpdatedTime": null
		}`)
	})

	bonus, _, err := client.Taxpayer.Bonus(context.Background())
	if err != nil {
		t.Fatalf("Bonus: %v", err)
	}

	if !bonus.BonusAmount.Equal(decimal.NewFromInt(8500)) {
		t.Errorf("BonusAmount = %s, want 8500", bonus.BonusAmount)
	}
	if bonus.AnnualIncomeStatus != "NORMAL" {
		t.Errorf("AnnualIncomeStatus = %q, want %q", bonus.AnnualIncomeStatus, "NORMAL")
	}
	if bonus.UpdatedTime.IsZero() {
		t.Error("UpdatedTime must be decoded")
	}
	if !bonus.TeenBonusUpdatedTime.IsZero() {
		t.Error("a null teenBonusUpdatedTime must decode to the zero time")
	}

	year, ok := bonus.TotalIncomeByYears["2025"]
	if !ok {
		t.Fatalf("TotalIncomeByYears = %v, want a 2025 entry", bonus.TotalIncomeByYears)
	}
	if !year.TotalIncomeAmount.Equal(decimal.NewFromInt(1090000)) {
		t.Errorf("2025 TotalIncomeAmount = %s, want 1090000", year.TotalIncomeAmount)
	}
}

// The API reports large amounts in exponential notation.
func TestTaxpayerBonusExponentialNumbers(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/taxpayer/bonus", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, `{
			"bonusAmount": 0,
			"totalIncomeAmount": 7.591E+5,
			"availableIncomeToExceedThreshold": 1.6409E+6,
			"annualIncomeStatus": "NORMAL",
			"updatedTime": "2026-07-14T21:15:27.647651Z",
			"totalIncomeByYears": {},
			"teenBonusAmount": null,
			"teenBonusUpdatedTime": null
		}`)
	})

	bonus, _, err := client.Taxpayer.Bonus(context.Background())
	if err != nil {
		t.Fatalf("Bonus: %v", err)
	}

	if !bonus.TotalIncomeAmount.Equal(decimal.NewFromInt(759100)) {
		t.Errorf("TotalIncomeAmount = %s, want 759100", bonus.TotalIncomeAmount)
	}
	if !bonus.AvailableIncomeToExceedThreshold.Equal(decimal.NewFromInt(1640900)) {
		t.Errorf("AvailableIncomeToExceedThreshold = %s, want 1640900", bonus.AvailableIncomeToExceedThreshold)
	}
}
