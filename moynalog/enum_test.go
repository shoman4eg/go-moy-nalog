package moynalog

import (
	"bytes"
	"encoding/json"
	"testing"
)

// Every enum value must serialise to the exact string the API expects. The
// cancellation reasons are Cyrillic and must not come out escaped.
func TestEnumJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value any
		want  string
	}{
		{BuyerTypePerson, `"PERSON"`},
		{BuyerTypeCompany, `"COMPANY"`},
		{BuyerTypeForeignAgency, `"FOREIGN_AGENCY"`},

		{PaymentTypeCash, `"CASH"`},
		{PaymentTypeAccount, `"ACCOUNT"`},

		{IncomeTypeIndividual, `"FROM_INDIVIDUAL"`},
		{IncomeTypeLegalEntity, `"FROM_LEGAL_ENTITY"`},
		{IncomeTypeForeignAgency, `"FROM_FOREIGN_AGENCY"`},

		{ReceiptTypeRegistered, `"REGISTERED"`},
		{ReceiptTypeCancelled, `"CANCELLED"`},

		{CancelCommentMistake, `"Чек сформирован ошибочно"`},
		{CancelCommentRefund, `"Возврат средств"`},

		{SortByOperationTimeDesc, `"operation_time:desc"`},
		{SortByOperationTimeAsc, `"operation_time:asc"`},
		{SortByTotalAmountDesc, `"total_amount:desc"`},
		{SortByTotalAmountAsc, `"total_amount:asc"`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("Marshal(%v): %v", tt.value, err)
			}
			if string(got) != tt.want {
				t.Errorf("Marshal(%v) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}
}

func TestEnumValid(t *testing.T) {
	t.Parallel()

	valid := []interface{ Valid() bool }{
		BuyerTypePerson, PaymentTypeCash, IncomeTypeIndividual,
		ReceiptTypeRegistered, CancelCommentRefund, SortByTotalAmountAsc,
	}
	for _, value := range valid {
		if !value.Valid() {
			t.Errorf("%v.Valid() = false, want true", value)
		}
	}

	invalid := []interface{ Valid() bool }{
		BuyerType("NOPE"), PaymentType("NOPE"), IncomeType("NOPE"),
		ReceiptType("NOPE"), CancelComment("своя причина"), SortBy("nope"),
	}
	for _, value := range invalid {
		if value.Valid() {
			t.Errorf("%v.Valid() = true, want false", value)
		}
	}
}

// The zero IncomeClient must go out as an anonymous individual with explicit
// nulls, which is what the API treats as "no counterparty given".
func TestIncomeClientMarshalJSONDefaults(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(IncomeClient{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	want := `{"contactPhone":null,"displayName":null,"incomeType":"FROM_INDIVIDUAL","inn":null}`
	if string(got) != want {
		t.Errorf("IncomeClient JSON =\n%s\nwant\n%s", got, want)
	}
}

func TestIncomeClientMarshalJSON(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(IncomeClient{
		ContactPhone: "79001234567",
		DisplayName:  "Test LLC",
		IncomeType:   IncomeTypeLegalEntity,
		Inn:          "1234567890",
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	want := `{"contactPhone":"79001234567","displayName":"Test LLC","incomeType":"FROM_LEGAL_ENTITY","inn":"1234567890"}`
	if string(got) != want {
		t.Errorf("IncomeClient JSON =\n%s\nwant\n%s", got, want)
	}
}

// Cyrillic must survive the encoder the client actually uses, which has HTML
// escaping switched off.
func TestCancelCommentSurvivesRequestEncoder(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(map[string]CancelComment{"comment": CancelCommentMistake}); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if want := `{"comment":"Чек сформирован ошибочно"}`; buf.String() != want+"\n" {
		t.Errorf("encoded = %q, want %q", buf.String(), want)
	}
}
