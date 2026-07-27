package moynalog

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

// InvoiceService handles the payment invoice endpoints.
type InvoiceService service

// InvoiceServiceItem is a single line of an invoice.
type InvoiceServiceItem struct {
	Name          string          `json:"name"`
	Amount        decimal.Decimal `json:"amount"`
	Quantity      decimal.Decimal `json:"quantity"`
	ServiceNumber int             `json:"serviceNumber"`
}

// MarshalJSON implements json.Marshaler. The API expects the amounts as JSON
// numbers, whereas decimal.Decimal encodes itself as a string by default.
func (i InvoiceServiceItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name          string          `json:"name"`
		Amount        json.RawMessage `json:"amount"`
		Quantity      json.RawMessage `json:"quantity"`
		ServiceNumber int             `json:"serviceNumber"`
	}{
		Name:          i.Name,
		Amount:        json.RawMessage(i.Amount.String()),
		Quantity:      json.RawMessage(i.Quantity.String()),
		ServiceNumber: i.ServiceNumber,
	})
}

// TotalAmount returns the amount this line contributes to the invoice.
func (i InvoiceServiceItem) TotalAmount() decimal.Decimal {
	return i.Amount.Mul(i.Quantity)
}

// InvoiceCreateRequest describes an invoice to issue.
type InvoiceCreateRequest struct {
	// Services are the invoice lines. At least one is required.
	Services []InvoiceServiceItem
	// OperationTime is when the service was rendered. Defaults to now.
	OperationTime time.Time
	// Client is the counterparty. Defaults to an anonymous individual.
	Client *IncomeClient
	// IgnoreMaxTotalIncomeRestriction issues the invoice even when it takes the
	// taxpayer past the annual income threshold.
	IgnoreMaxTotalIncomeRestriction bool
}

type invoiceCreateBody struct {
	PaymentType                     PaymentType          `json:"paymentType"`
	IgnoreMaxTotalIncomeRestriction bool                 `json:"ignoreMaxTotalIncomeRestriction"`
	Client                          IncomeClient         `json:"client"`
	Services                        []InvoiceServiceItem `json:"services"`
	RequestTime                     Time                 `json:"requestTime"`
	OperationTime                   Time                 `json:"operationTime"`
	TotalAmount                     string               `json:"totalAmount"`
}

// Create issues an invoice. Invoices are always settled through an account, so
// the payment type is fixed to PaymentTypeAccount.
//
// POST /invoice
func (s *InvoiceService) Create(ctx context.Context, invoice *InvoiceCreateRequest) (*IncomeCreated, *Response, error) {
	if invoice == nil {
		return nil, nil, errors.New("moynalog: invoice create request cannot be nil")
	}
	if len(invoice.Services) == 0 {
		return nil, nil, errors.New("moynalog: services cannot be empty")
	}
	// The counterparty is validated exactly as it is for a receipt: an invoice
	// issued to a legal entity has to identify it.
	if err := validateIncomeClient(invoice.Client); err != nil {
		return nil, nil, err
	}

	totalAmount := decimal.Zero
	for i, item := range invoice.Services {
		if item.Name == "" {
			return nil, nil, errors.Errorf("moynalog: name of item[%d] cannot be empty", i)
		}
		if item.Amount.LessThanOrEqual(decimal.Zero) {
			return nil, nil, errors.Errorf("moynalog: amount of item[%d] must be greater than 0", i)
		}
		if item.Quantity.LessThanOrEqual(decimal.Zero) {
			return nil, nil, errors.Errorf("moynalog: quantity of item[%d] must be greater than 0", i)
		}
		totalAmount = totalAmount.Add(item.TotalAmount())
	}

	requestTime := time.Now()
	operationTime := invoice.OperationTime
	if operationTime.IsZero() {
		operationTime = requestTime
	}

	client := IncomeClient{IncomeType: IncomeTypeIndividual}
	if invoice.Client != nil {
		client = *invoice.Client
	}

	body := &invoiceCreateBody{
		PaymentType:                     PaymentTypeAccount,
		IgnoreMaxTotalIncomeRestriction: invoice.IgnoreMaxTotalIncomeRestriction,
		Client:                          client,
		Services:                        invoice.Services,
		RequestTime:                     NewTime(requestTime),
		OperationTime:                   NewTime(operationTime),
		TotalAmount:                     totalAmount.String(),
	}

	req, err := s.client.NewRequest(http.MethodPost, "invoice", body)
	if err != nil {
		return nil, nil, err
	}

	created := new(IncomeCreated)
	resp, err := s.client.Do(ctx, req, created)
	if err != nil {
		return nil, resp, err
	}

	return created, resp, nil
}

// CreateItem issues a single line invoice. It is a shorthand for Create.
func (s *InvoiceService) CreateItem(ctx context.Context, name string, amount, quantity decimal.Decimal) (*IncomeCreated, *Response, error) {
	return s.Create(ctx, &InvoiceCreateRequest{
		Services: []InvoiceServiceItem{{Name: name, Amount: amount, Quantity: quantity}},
	})
}

// Cancel always fails with ErrNotImplemented: the upstream API exposes no
// invoice cancellation endpoint. It exists to mirror the reference PHP client.
func (s *InvoiceService) Cancel(_ context.Context, _ int64) (*Response, error) {
	return nil, ErrNotImplemented
}

// UpdatePaymentInfo always fails with ErrNotImplemented: the upstream API
// exposes no such endpoint. It exists to mirror the reference PHP client.
func (s *InvoiceService) UpdatePaymentInfo(_ context.Context) (*Response, error) {
	return nil, ErrNotImplemented
}
