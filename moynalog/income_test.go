package moynalog

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

func TestIncomeCreate(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/income", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testHeader(t, r, "Content-Type", mediaTypeJSON)

		body := testBody(t, r)

		// 1000 * 10 + 1900.33 * 10 = 29003.3, sent as a string.
		if body["totalAmount"] != "29003.3" {
			t.Errorf("totalAmount = %v, want %q", body["totalAmount"], "29003.3")
		}
		if body["paymentType"] != string(PaymentTypeCash) {
			t.Errorf("paymentType = %v, want %q", body["paymentType"], PaymentTypeCash)
		}
		if body["ignoreMaxTotalIncomeRestriction"] != false {
			t.Errorf("ignoreMaxTotalIncomeRestriction = %v, want false", body["ignoreMaxTotalIncomeRestriction"])
		}

		services, ok := body["services"].([]any)
		if !ok || len(services) != 2 {
			t.Fatalf("services = %v, want two items", body["services"])
		}
		first, ok := services[0].(map[string]any)
		if !ok {
			t.Fatalf("services[0] = %v, want an object", services[0])
		}
		// Amounts must go out as JSON numbers, not strings.
		if _, isNumber := first["amount"].(float64); !isNumber {
			t.Errorf("services[0].amount = %#v, want a JSON number", first["amount"])
		}
		if _, isNumber := first["quantity"].(float64); !isNumber {
			t.Errorf("services[0].quantity = %#v, want a JSON number", first["quantity"])
		}

		client, ok := body["client"].(map[string]any)
		if !ok {
			t.Fatalf("client = %v, want an object", body["client"])
		}
		if client["incomeType"] != string(IncomeTypeIndividual) {
			t.Errorf("client.incomeType = %v, want %q", client["incomeType"], IncomeTypeIndividual)
		}
		// Absent optional fields must be null, not omitted.
		if _, present := client["contactPhone"]; !present {
			t.Error("client.contactPhone must be present as null")
		}

		writeJSON(t, w, http.StatusOK, `{"approvedReceiptUuid":"2000xfg2ff"}`)
	})

	created, _, err := client.Income.Create(context.Background(), &IncomeCreateRequest{
		Services: []IncomeServiceItem{
			{Name: "Test service", Amount: decimal.NewFromInt(1000), Quantity: decimal.NewFromInt(10)},
			{Name: "Test 2", Amount: decimal.NewFromFloat(1900.33), Quantity: decimal.NewFromInt(10)},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ApprovedReceiptUUID != "2000xfg2ff" {
		t.Errorf("ApprovedReceiptUUID = %q, want %q", created.ApprovedReceiptUUID, "2000xfg2ff")
	}
}

func TestIncomeCreateItem(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/income", func(w http.ResponseWriter, r *http.Request) {
		services, ok := testBody(t, r)["services"].([]any)
		if !ok || len(services) != 1 {
			t.Fatalf("services = %v, want one item", services)
		}
		writeJSON(t, w, http.StatusOK, `{"approvedReceiptUuid":"single"}`)
	})

	created, _, err := client.Income.CreateItem(
		context.Background(), "Услуга", decimal.NewFromInt(100), decimal.NewFromInt(1),
	)
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if created.ApprovedReceiptUUID != "single" {
		t.Errorf("ApprovedReceiptUUID = %q, want %q", created.ApprovedReceiptUUID, "single")
	}
}

func TestIncomeCreateValidation(t *testing.T) {
	t.Parallel()

	valid := IncomeServiceItem{Name: "ok", Amount: decimal.NewFromInt(1), Quantity: decimal.NewFromInt(1)}
	legalEntity := func(inn, displayName string) *IncomeClient {
		return &IncomeClient{IncomeType: IncomeTypeLegalEntity, Inn: inn, DisplayName: displayName}
	}

	// The messages mirror the assertions of the reference PHP client, so that
	// disabling any single guard is caught rather than masked by the next one.
	tests := []struct {
		name        string
		request     *IncomeCreateRequest
		wantMessage string
	}{
		{"nil request", nil, "cannot be nil"},
		{"no services", new(IncomeCreateRequest), "services cannot be empty"},
		{
			"empty name",
			&IncomeCreateRequest{Services: []IncomeServiceItem{{Amount: decimal.NewFromInt(1), Quantity: decimal.NewFromInt(1)}}},
			"name of item[0] cannot be empty",
		},
		{
			"zero amount",
			&IncomeCreateRequest{Services: []IncomeServiceItem{{Name: "x", Quantity: decimal.NewFromInt(1)}}},
			"amount of item[0] must be greater than 0",
		},
		{
			"negative quantity",
			&IncomeCreateRequest{Services: []IncomeServiceItem{{
				Name:     "x",
				Amount:   decimal.NewFromInt(1),
				Quantity: decimal.NewFromInt(-1),
			}}},
			"quantity of item[0] must be greater than 0",
		},
		{
			"legal entity without INN",
			&IncomeCreateRequest{Services: []IncomeServiceItem{valid}, Client: legalEntity("", "ООО")},
			"client INN cannot be empty",
		},
		{
			"legal entity with a non numeric INN",
			&IncomeCreateRequest{Services: []IncomeServiceItem{valid}, Client: legalEntity("abcdefghij", "ООО")},
			"client INN must contain only digits",
		},
		{
			"legal entity with a wrong INN length",
			&IncomeCreateRequest{Services: []IncomeServiceItem{valid}, Client: legalEntity("12345", "ООО")},
			"client INN length must be 10 or 12",
		},
		{
			"legal entity without a display name",
			&IncomeCreateRequest{Services: []IncomeServiceItem{valid}, Client: legalEntity("7724035047", "")},
			"client display name cannot be empty",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := setupNoRequest(t)
			_, _, err := client.Income.Create(context.Background(), tt.request)
			assertLocalError(t, err)

			if err != nil && !strings.Contains(err.Error(), tt.wantMessage) {
				t.Errorf("error = %q, want it to mention %q", err, tt.wantMessage)
			}
		})
	}
}

// A ten or twelve digit INN on a legal entity receipt must be accepted.
func TestIncomeCreateAcceptsLegalEntity(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/income", func(w http.ResponseWriter, r *http.Request) {
		client, ok := testBody(t, r)["client"].(map[string]any)
		if !ok {
			t.Fatal("client must be an object")
		}
		if client["inn"] != "7724035047" {
			t.Errorf("client.inn = %v, want %q", client["inn"], "7724035047")
		}
		writeJSON(t, w, http.StatusOK, `{"approvedReceiptUuid":"legal"}`)
	})

	_, _, err := client.Income.Create(context.Background(), &IncomeCreateRequest{
		Services: []IncomeServiceItem{{
			Name:     "Услуга",
			Amount:   decimal.NewFromInt(1),
			Quantity: decimal.NewFromInt(1),
		}},
		Client: &IncomeClient{
			IncomeType:  IncomeTypeLegalEntity,
			Inn:         "7724035047",
			DisplayName: "ИП Литвинов Сергей Александрович",
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestIncomeList(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/incomes", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		testQuery(t, r, map[string]string{
			"limit":       "5",
			"offset":      "10",
			"sortBy":      string(SortByOperationTimeDesc),
			"buyerType":   string(BuyerTypePerson),
			"receiptType": string(ReceiptTypeRegistered),
			"from":        "2021-03-30T22:39:54.391Z",
		})

		writeJSON(t, w, http.StatusOK, `{
			"content": [{
				"approvedReceiptUuid": "20ht1axfzg",
				"name": "Предоставление информационных услуг",
				"services": [{"name": "Услуга", "quantity": 1, "serviceNumber": 0, "amount": 86.28}],
				"operationTime": "2022-03-30T17:45:12Z",
				"requestTime": "2022-03-30T17:45:12Z",
				"registerTime": "2022-03-30T17:45:13.286887Z",
				"taxPeriodId": 202203,
				"paymentType": "CASH",
				"incomeType": "FROM_INDIVIDUAL",
				"partnerCode": null,
				"totalAmount": 86.28,
				"cancellationInfo": null,
				"sourceDeviceId": "device",
				"clientInn": null,
				"clientDisplayName": null,
				"partnerDisplayName": null,
				"partnerLogo": null,
				"partnerInn": null,
				"inn": "770000000000",
				"profession": "IT",
				"description": [],
				"invoiceId": null
			}],
			"hasMore": true,
			"currentOffset": 10,
			"currentLimit": 5
		}`)
	})

	list, _, err := client.Income.List(context.Background(), &IncomeListOptions{
		From:        NewTime(time.Date(2021, time.March, 30, 22, 39, 54, 391000000, time.UTC)),
		Limit:       5,
		Offset:      10,
		SortBy:      SortByOperationTimeDesc,
		BuyerType:   BuyerTypePerson,
		ReceiptType: ReceiptTypeRegistered,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if !list.HasMore {
		t.Error("HasMore = false, want true")
	}
	if len(list.Content) != 1 {
		t.Fatalf("Content has %d items, want 1", len(list.Content))
	}

	item := list.Content[0]
	if item.ApprovedReceiptUUID != "20ht1axfzg" {
		t.Errorf("ApprovedReceiptUUID = %q, want %q", item.ApprovedReceiptUUID, "20ht1axfzg")
	}
	if !item.TotalAmount.Equal(decimal.NewFromFloat(86.28)) {
		t.Errorf("TotalAmount = %s, want 86.28", item.TotalAmount)
	}
	if item.Cancelled() {
		t.Error("an item without cancellationInfo must not read as cancelled")
	}
	if len(item.Services) != 1 || item.Services[0].Name != "Услуга" {
		t.Errorf("Services = %+v, want one named item", item.Services)
	}
}

// A nil options value must still produce the API defaults.
func TestIncomeListDefaults(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/incomes", func(w http.ResponseWriter, r *http.Request) {
		testQuery(t, r, map[string]string{
			"limit":  "100",
			"offset": "0",
			"sortBy": string(SortByOperationTimeDesc),
		})
		writeJSON(t, w, http.StatusOK, `{"content":[],"hasMore":false,"currentOffset":0,"currentLimit":100}`)
	})

	if _, _, err := client.Income.List(context.Background(), nil); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestIncomeListClampsLimit(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/incomes", func(w http.ResponseWriter, r *http.Request) {
		testQuery(t, r, map[string]string{"limit": "100"})
		writeJSON(t, w, http.StatusOK, `{"content":[]}`)
	})

	if _, _, err := client.Income.List(context.Background(), &IncomeListOptions{Limit: 5000}); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestIncomeListRejectsUnknownSort(t *testing.T) {
	t.Parallel()

	client := setupNoRequest(t)

	_, _, err := client.Income.List(context.Background(), &IncomeListOptions{SortBy: "nope"})
	assertLocalError(t, err)
}

func TestIncomeCancel(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/cancel", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)

		body := testBody(t, r)
		if body["receiptUuid"] != "20hukwpbp2" {
			t.Errorf("receiptUuid = %v, want %q", body["receiptUuid"], "20hukwpbp2")
		}
		if body["comment"] != string(CancelCommentMistake) {
			t.Errorf("comment = %v, want %q", body["comment"], CancelCommentMistake)
		}
		if body["partnerCode"] != nil {
			t.Errorf("partnerCode = %v, want null", body["partnerCode"])
		}

		writeJSON(t, w, http.StatusOK, `{
			"incomeInfo": {
				"approvedReceiptUuid": "20hukwpbp2",
				"name": "Предоставление информационных услуг",
				"operationTime": "2022-03-30T22:46:06+02:00",
				"requestTime": "2022-03-30T22:46:06+02:00",
				"paymentType": "CASH",
				"partnerCode": null,
				"totalAmount": 50,
				"cancellationInfo": {
					"operationTime": "2022-03-30T22:48:09+02:00",
					"registerTime": "2022-03-30T20:48:09.646286Z",
					"taxPeriodId": 202203,
					"comment": "Чек сформирован ошибочно"
				},
				"sourceDeviceId": "Q0CJdFn9zmk7BQw4vYz_M"
			}
		}`)
	})

	cancelled, _, err := client.Income.Cancel(context.Background(), &IncomeCancelRequest{
		ReceiptUUID: "20hukwpbp2",
		Comment:     CancelCommentMistake,
	})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	if cancelled.ApprovedReceiptUUID != "20hukwpbp2" {
		t.Errorf("ApprovedReceiptUUID = %q, want %q", cancelled.ApprovedReceiptUUID, "20hukwpbp2")
	}
	if !cancelled.TotalAmount.Equal(decimal.NewFromInt(50)) {
		t.Errorf("TotalAmount = %s, want 50", cancelled.TotalAmount)
	}
	if cancelled.CancellationInfo == nil {
		t.Fatal("CancellationInfo must be decoded")
	}
	if cancelled.CancellationInfo.Comment != CancelCommentMistake {
		t.Errorf("Comment = %q, want %q", cancelled.CancellationInfo.Comment, CancelCommentMistake)
	}
	if cancelled.CancellationInfo.TaxPeriodID != 202203 {
		t.Errorf("TaxPeriodID = %d, want 202203", cancelled.CancellationInfo.TaxPeriodID)
	}
}

func TestIncomeCancelValidation(t *testing.T) {
	t.Parallel()

	client := setupNoRequest(t)

	_, _, err := client.Income.Cancel(context.Background(), nil)
	assertLocalError(t, err)

	_, _, err = client.Income.Cancel(context.Background(), &IncomeCancelRequest{
		Comment: CancelCommentRefund,
	})
	assertLocalError(t, err)

	_, _, err = client.Income.Cancel(context.Background(), &IncomeCancelRequest{
		ReceiptUUID: "uuid",
		Comment:     "своя причина",
	})
	assertLocalError(t, err)
}

func TestIncomeCancelledRequiresEnvelope(t *testing.T) {
	t.Parallel()

	var cancelled IncomeCancelled
	if err := json.Unmarshal([]byte(`{}`), &cancelled); err == nil {
		t.Error("a response without incomeInfo must be reported")
	}
}

func TestIncomeErrorsAreTyped(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/incomes", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusBadRequest, `{"code":"invalid","message":"Неверный запрос"}`)
	})

	_, resp, err := client.Income.List(context.Background(), nil)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
	if resp == nil || resp.StatusCode != http.StatusBadRequest {
		t.Error("the response must be returned alongside the error")
	}

	var errResp *ErrorResponse
	if !errors.As(err, &errResp) || errResp.Message != "Неверный запрос" {
		t.Errorf("error = %v, want the API message preserved", err)
	}
}

// The exact datasets the PHP client checks its BigDecimal arithmetic against.
func TestIncomeCreateTotalAmount(t *testing.T) {
	t.Parallel()

	type item struct {
		amount   string
		quantity int64
	}

	tests := []struct {
		name  string
		items []item
		want  string
	}{
		{
			name:  "integer amounts",
			items: []item{{"100", 1}, {"200", 2}, {"300", 3}},
			want:  "1400",
		},
		{
			name:  "fractional amounts must not drift",
			items: []item{{"30.23", 1}, {"12.33", 8}, {"32.44", 9}},
			want:  "420.83",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, mux := setupAuthed(t)
			mux.HandleFunc("/v1/income", func(w http.ResponseWriter, r *http.Request) {
				if got := testBody(t, r)["totalAmount"]; got != tt.want {
					t.Errorf("totalAmount = %v, want %q", got, tt.want)
				}
				writeJSON(t, w, http.StatusOK, `{"approvedReceiptUuid":"randomReceiptId"}`)
			})

			services := make([]IncomeServiceItem, 0, len(tt.items))
			for _, it := range tt.items {
				amount, err := decimal.NewFromString(it.amount)
				if err != nil {
					t.Fatalf("parse %q: %v", it.amount, err)
				}
				services = append(services, IncomeServiceItem{
					Name:     "randomName",
					Amount:   amount,
					Quantity: decimal.NewFromInt(it.quantity),
				})
			}

			if _, _, err := client.Income.Create(context.Background(), &IncomeCreateRequest{
				Services: services,
			}); err != nil {
				t.Fatalf("Create: %v", err)
			}
		})
	}
}

func TestIncomeServiceItemTotalAmount(t *testing.T) {
	t.Parallel()

	item := IncomeServiceItem{
		Name:     "service",
		Amount:   decimal.RequireFromString("30.23"),
		Quantity: decimal.NewFromInt(3),
	}

	if got := item.TotalAmount().String(); got != "90.69" {
		t.Errorf("TotalAmount = %s, want 90.69", got)
	}
}

// Only 10 and 12 digit INNs are valid for a legal entity.
func TestIncomeCreateINNLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		inn     string
		wantErr bool
	}{
		{"nine digits", strings.Repeat("1", 9), true},
		{"ten digits", strings.Repeat("1", 10), false},
		{"eleven digits", strings.Repeat("1", 11), true},
		{"twelve digits", strings.Repeat("1", 12), false},
		{"thirteen digits", strings.Repeat("1", 13), true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, mux := setupAuthed(t)
			mux.HandleFunc("/v1/income", func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(t, w, http.StatusOK, `{"approvedReceiptUuid":"randomReceiptId"}`)
			})

			_, _, err := client.Income.Create(context.Background(), &IncomeCreateRequest{
				Services: []IncomeServiceItem{{
					Name:     "name",
					Amount:   decimal.NewFromInt(1),
					Quantity: decimal.NewFromInt(1),
				}},
				Client: &IncomeClient{
					IncomeType:  IncomeTypeLegalEntity,
					Inn:         tt.inn,
					DisplayName: "testClient",
				},
			})

			if tt.wantErr && err == nil {
				t.Errorf("INN of length %d must be rejected", len(tt.inn))
			}
			if !tt.wantErr && err != nil {
				t.Errorf("INN of length %d must be accepted, got %v", len(tt.inn), err)
			}
		})
	}
}

// Only legal entities are validated; a foreign agency needs no INN.
func TestIncomeCreateForeignAgencyNeedsNoINN(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/income", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"approvedReceiptUuid":"randomReceiptId"}`)
	})

	_, _, err := client.Income.Create(context.Background(), &IncomeCreateRequest{
		Services: []IncomeServiceItem{{
			Name:     "name",
			Amount:   decimal.NewFromInt(100),
			Quantity: decimal.NewFromInt(1),
		}},
		Client: &IncomeClient{
			ContactPhone: "phone",
			DisplayName:  "testClient",
			IncomeType:   IncomeTypeForeignAgency,
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestIncomeListLimitClamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		limit int
		want  string
	}{
		// Unlike the PHP client, which clamps an explicit 0 up to 1, a zero
		// Limit here means "unset" and falls back to the API maximum.
		{"unset", 0, "100"},
		{"below min", -5, "1"},
		{"minimum", 1, "1"},
		{"within range", 50, "50"},
		{"above max", 200, "100"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, mux := setupAuthed(t)
			mux.HandleFunc("/v1/incomes", func(w http.ResponseWriter, r *http.Request) {
				testQuery(t, r, map[string]string{"limit": tt.want})
				writeJSON(t, w, http.StatusOK, `{"content":[]}`)
			})

			if _, _, err := client.Income.List(context.Background(), &IncomeListOptions{Limit: tt.limit}); err != nil {
				t.Fatalf("List: %v", err)
			}
		})
	}
}

// The API sometimes answers with empty strings where timestamps are expected.
func TestIncomeCancelEmptyTimestamps(t *testing.T) {
	t.Parallel()

	for _, comment := range []CancelComment{CancelCommentMistake, CancelCommentRefund} {
		comment := comment
		t.Run(string(comment), func(t *testing.T) {
			t.Parallel()

			client, mux := setupAuthed(t)
			mux.HandleFunc("/v1/cancel", func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(t, w, http.StatusOK, `{
					"incomeInfo": {
						"approvedReceiptUuid": "12345678",
						"name": "",
						"operationTime": "",
						"requestTime": "",
						"paymentType": "CASH",
						"partnerCode": "",
						"totalAmount": 50,
						"cancellationInfo": {
							"operationTime": "",
							"registerTime": "",
							"taxPeriodId": 202203,
							"comment": "`+string(comment)+`"
						},
						"sourceDeviceId": "davxcvc90876rsdf"
					}
				}`)
			})

			cancelled, _, err := client.Income.Cancel(context.Background(), &IncomeCancelRequest{
				ReceiptUUID: "12345678",
				Comment:     comment,
			})
			if err != nil {
				t.Fatalf("Cancel: %v", err)
			}
			if cancelled.ApprovedReceiptUUID != "12345678" {
				t.Errorf("ApprovedReceiptUUID = %q, want %q", cancelled.ApprovedReceiptUUID, "12345678")
			}
			if cancelled.CancellationInfo.Comment != comment {
				t.Errorf("Comment = %q, want %q", cancelled.CancellationInfo.Comment, comment)
			}
			if !cancelled.OperationTime.IsZero() {
				t.Error("an empty timestamp must decode to the zero time")
			}
		})
	}
}

// Enum-typed fields are plain strings in Go, so anything can be assigned to
// them; the client must reject values the API does not know.
func TestIncomeRejectsUnknownEnumValues(t *testing.T) {
	t.Parallel()

	validItem := IncomeServiceItem{
		Name:     "name",
		Amount:   decimal.NewFromInt(1),
		Quantity: decimal.NewFromInt(1),
	}

	t.Run("payment type", func(t *testing.T) {
		t.Parallel()

		client := setupNoRequest(t)
		_, _, err := client.Income.Create(context.Background(), &IncomeCreateRequest{
			Services:    []IncomeServiceItem{validItem},
			PaymentType: "BITCOIN",
		})
		assertLocalError(t, err)
	})

	t.Run("income type", func(t *testing.T) {
		t.Parallel()

		client := setupNoRequest(t)
		_, _, err := client.Income.Create(context.Background(), &IncomeCreateRequest{
			Services: []IncomeServiceItem{validItem},
			Client:   &IncomeClient{IncomeType: "FROM_MARS"},
		})
		assertLocalError(t, err)
	})

	t.Run("buyer type", func(t *testing.T) {
		t.Parallel()

		client := setupNoRequest(t)
		_, _, err := client.Income.List(context.Background(), &IncomeListOptions{BuyerType: "ALIEN"})
		assertLocalError(t, err)
	})

	t.Run("receipt type", func(t *testing.T) {
		t.Parallel()

		client := setupNoRequest(t)
		_, _, err := client.Income.List(context.Background(), &IncomeListOptions{ReceiptType: "SHREDDED"})
		assertLocalError(t, err)
	})
}
