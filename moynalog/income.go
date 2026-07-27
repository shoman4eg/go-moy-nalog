package moynalog

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	// minListLimit and maxListLimit bound the page size the listing accepts.
	minListLimit = 1
	maxListLimit = 100

	innLengthIndividual  = 12
	innLengthLegalEntity = 10
)

// IncomeService handles the receipt (income) endpoints.
type IncomeService service

// IncomeClient is the counterparty a receipt is issued to. The zero value
// describes an anonymous individual, which is what the API defaults to.
type IncomeClient struct {
	ContactPhone string     `json:"contactPhone"`
	DisplayName  string     `json:"displayName"`
	IncomeType   IncomeType `json:"incomeType"`
	Inn          string     `json:"inn"`
}

// MarshalJSON implements json.Marshaler. Absent fields must be sent as null,
// and an unset income type defaults to an individual.
func (c IncomeClient) MarshalJSON() ([]byte, error) {
	incomeType := c.IncomeType
	if incomeType == "" {
		incomeType = IncomeTypeIndividual
	}

	return json.Marshal(struct {
		ContactPhone *string    `json:"contactPhone"`
		DisplayName  *string    `json:"displayName"`
		IncomeType   IncomeType `json:"incomeType"`
		Inn          *string    `json:"inn"`
	}{
		ContactPhone: nullableString(c.ContactPhone),
		DisplayName:  nullableString(c.DisplayName),
		IncomeType:   incomeType,
		Inn:          nullableString(c.Inn),
	})
}

// IncomeServiceItem is a single line of a receipt.
type IncomeServiceItem struct {
	Name     string          `json:"name"`
	Amount   decimal.Decimal `json:"amount"`
	Quantity decimal.Decimal `json:"quantity"`
}

// MarshalJSON implements json.Marshaler. The API expects the amounts as JSON
// numbers, whereas decimal.Decimal encodes itself as a string by default.
func (i IncomeServiceItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name     string          `json:"name"`
		Amount   json.RawMessage `json:"amount"`
		Quantity json.RawMessage `json:"quantity"`
	}{
		Name:     i.Name,
		Amount:   json.RawMessage(i.Amount.String()),
		Quantity: json.RawMessage(i.Quantity.String()),
	})
}

// TotalAmount returns the amount this line contributes to the receipt.
func (i IncomeServiceItem) TotalAmount() decimal.Decimal {
	return i.Amount.Mul(i.Quantity)
}

// IncomeCreateRequest describes a receipt to register.
type IncomeCreateRequest struct {
	// Services are the receipt lines. At least one is required.
	Services []IncomeServiceItem
	// OperationTime is when the service was rendered. Defaults to now.
	OperationTime time.Time
	// Client is the counterparty. Defaults to an anonymous individual.
	Client *IncomeClient
	// PaymentType defaults to PaymentTypeCash.
	PaymentType PaymentType
	// IgnoreMaxTotalIncomeRestriction registers the receipt even when it takes
	// the taxpayer past the annual income threshold.
	IgnoreMaxTotalIncomeRestriction bool
}

// incomeCreateBody is the wire representation of IncomeCreateRequest.
type incomeCreateBody struct {
	OperationTime                   Time                `json:"operationTime"`
	RequestTime                     Time                `json:"requestTime"`
	Services                        []IncomeServiceItem `json:"services"`
	TotalAmount                     string              `json:"totalAmount"`
	Client                          IncomeClient        `json:"client"`
	PaymentType                     PaymentType         `json:"paymentType"`
	IgnoreMaxTotalIncomeRestriction bool                `json:"ignoreMaxTotalIncomeRestriction"`
}

// IncomeCreated identifies a freshly registered receipt.
type IncomeCreated struct {
	ApprovedReceiptUUID string `json:"approvedReceiptUuid"`
}

// Create registers a receipt.
//
// POST /income
func (s *IncomeService) Create(ctx context.Context, income *IncomeCreateRequest) (*IncomeCreated, *Response, error) {
	if income == nil {
		return nil, nil, errors.New("moynalog: income create request cannot be nil")
	}
	if err := validateIncomeCreate(income); err != nil {
		return nil, nil, err
	}

	totalAmount := decimal.Zero
	for _, item := range income.Services {
		totalAmount = totalAmount.Add(item.TotalAmount())
	}

	paymentType := income.PaymentType
	if paymentType == "" {
		paymentType = PaymentTypeCash
	}

	operationTime := income.OperationTime
	requestTime := time.Now()
	if operationTime.IsZero() {
		operationTime = requestTime
	}

	client := IncomeClient{IncomeType: IncomeTypeIndividual}
	if income.Client != nil {
		client = *income.Client
	}

	body := &incomeCreateBody{
		OperationTime:                   NewTime(operationTime),
		RequestTime:                     NewTime(requestTime),
		Services:                        income.Services,
		TotalAmount:                     totalAmount.String(),
		Client:                          client,
		PaymentType:                     paymentType,
		IgnoreMaxTotalIncomeRestriction: income.IgnoreMaxTotalIncomeRestriction,
	}

	req, err := s.client.NewRequest(http.MethodPost, "income", body)
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

// CreateItem registers a single line receipt for an anonymous individual. It is
// a shorthand for Create with one service item.
func (s *IncomeService) CreateItem(ctx context.Context, name string, amount, quantity decimal.Decimal) (*IncomeCreated, *Response, error) {
	return s.Create(ctx, &IncomeCreateRequest{
		Services: []IncomeServiceItem{{Name: name, Amount: amount, Quantity: quantity}},
	})
}

func validateIncomeCreate(income *IncomeCreateRequest) error {
	if len(income.Services) == 0 {
		return errors.New("moynalog: services cannot be empty")
	}

	for i, item := range income.Services {
		if item.Name == "" {
			return errors.Errorf("moynalog: name of item[%d] cannot be empty", i)
		}
		if item.Amount.LessThanOrEqual(decimal.Zero) {
			return errors.Errorf("moynalog: amount of item[%d] must be greater than 0", i)
		}
		if item.Quantity.LessThanOrEqual(decimal.Zero) {
			return errors.Errorf("moynalog: quantity of item[%d] must be greater than 0", i)
		}
	}

	if income.PaymentType != "" && !income.PaymentType.Valid() {
		return errors.Errorf("moynalog: payment type %q is invalid", income.PaymentType)
	}

	return validateIncomeClient(income.Client)
}

// validateIncomeClient checks the counterparty of a receipt or an invoice. An
// empty income type is allowed and defaults to an individual on the wire.
func validateIncomeClient(client *IncomeClient) error {
	if client == nil {
		return nil
	}
	if client.IncomeType != "" && !client.IncomeType.Valid() {
		return errors.Errorf("moynalog: income type %q is invalid", client.IncomeType)
	}
	if client.IncomeType != IncomeTypeLegalEntity {
		return nil
	}

	// Only receipts issued to a legal entity have to identify it.
	if client.Inn == "" {
		return errors.New("moynalog: client INN cannot be empty")
	}
	if _, err := strconv.ParseUint(client.Inn, 10, 64); err != nil {
		return errors.New("moynalog: client INN must contain only digits")
	}
	if len(client.Inn) != innLengthLegalEntity && len(client.Inn) != innLengthIndividual {
		return errors.Errorf("moynalog: client INN length must be %d or %d", innLengthLegalEntity, innLengthIndividual)
	}
	if client.DisplayName == "" {
		return errors.New("moynalog: client display name cannot be empty")
	}

	return nil
}

// IncomeListOptions filters and paginates a receipt listing.
type IncomeListOptions struct {
	From        Time        `url:"from,omitempty"`
	To          Time        `url:"to,omitempty"`
	Limit       int         `url:"limit"`
	Offset      int         `url:"offset"`
	SortBy      SortBy      `url:"sortBy,omitempty"`
	BuyerType   BuyerType   `url:"buyerType,omitempty"`
	ReceiptType ReceiptType `url:"receiptType,omitempty"`
}

// IncomeList is one page of registered receipts.
type IncomeList struct {
	Content       []*IncomeListItem `json:"content"`
	HasMore       bool              `json:"hasMore"`
	CurrentOffset int               `json:"currentOffset"`
	CurrentLimit  int               `json:"currentLimit"`
}

// IncomeListItem is a registered receipt as returned by a listing.
type IncomeListItem struct {
	ApprovedReceiptUUID string            `json:"approvedReceiptUuid"`
	Name                string            `json:"name"`
	Services            []*ServiceItem    `json:"services"`
	OperationTime       Time              `json:"operationTime"`
	RequestTime         Time              `json:"requestTime"`
	RegisterTime        Time              `json:"registerTime"`
	TaxPeriodID         int               `json:"taxPeriodId"`
	PaymentType         PaymentType       `json:"paymentType"`
	IncomeType          IncomeType        `json:"incomeType"`
	PartnerCode         string            `json:"partnerCode"`
	TotalAmount         decimal.Decimal   `json:"totalAmount"`
	CancellationInfo    *CancellationInfo `json:"cancellationInfo"`
	SourceDeviceID      string            `json:"sourceDeviceId"`
	ClientInn           string            `json:"clientInn"`
	ClientDisplayName   string            `json:"clientDisplayName"`
	PartnerDisplayName  string            `json:"partnerDisplayName"`
	PartnerLogo         string            `json:"partnerLogo"`
	PartnerInn          string            `json:"partnerInn"`
	Inn                 string            `json:"inn"`
	Profession          string            `json:"profession"`
	Description         []string          `json:"description"`
	InvoiceID           string            `json:"invoiceId"`
}

// Cancelled reports whether the receipt has been cancelled.
func (i *IncomeListItem) Cancelled() bool {
	return i.CancellationInfo != nil
}

// ServiceItem is a receipt line as returned by the API.
type ServiceItem struct {
	Name          string          `json:"name"`
	Quantity      decimal.Decimal `json:"quantity"`
	ServiceNumber int             `json:"serviceNumber"`
	Amount        decimal.Decimal `json:"amount"`
}

// CancellationInfo records when and why a receipt was cancelled.
type CancellationInfo struct {
	OperationTime Time          `json:"operationTime"`
	RegisterTime  Time          `json:"registerTime"`
	TaxPeriodID   int           `json:"taxPeriodId"`
	Comment       CancelComment `json:"comment"`
}

// List returns a page of registered receipts. A nil opts lists the 100 most
// recent receipts.
//
// GET /incomes
func (s *IncomeService) List(ctx context.Context, opts *IncomeListOptions) (*IncomeList, *Response, error) {
	query := IncomeListOptions{
		Limit:  maxListLimit,
		SortBy: SortByOperationTimeDesc,
	}
	if opts != nil {
		query = *opts
	}

	if query.Limit == 0 {
		query.Limit = maxListLimit
	}
	query.Limit = clamp(query.Limit, minListLimit, maxListLimit)

	if query.SortBy != "" && !query.SortBy.Valid() {
		return nil, nil, errors.Errorf("moynalog: sort order %q is invalid", query.SortBy)
	}
	if query.BuyerType != "" && !query.BuyerType.Valid() {
		return nil, nil, errors.Errorf("moynalog: buyer type %q is invalid", query.BuyerType)
	}
	if query.ReceiptType != "" && !query.ReceiptType.Valid() {
		return nil, nil, errors.Errorf("moynalog: receipt type %q is invalid", query.ReceiptType)
	}

	u, err := addOptions("incomes", query)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	list := new(IncomeList)
	resp, err := s.client.Do(ctx, req, list)
	if err != nil {
		return nil, resp, err
	}

	return list, resp, nil
}

// IncomeCancelRequest describes a receipt cancellation.
type IncomeCancelRequest struct {
	// ReceiptUUID identifies the receipt to cancel. Required.
	ReceiptUUID string
	// Comment is the reason for the cancellation. Required.
	Comment CancelComment
	// OperationTime defaults to now.
	OperationTime time.Time
	// RequestTime defaults to now.
	RequestTime time.Time
	// PartnerCode is optional.
	PartnerCode string
}

type incomeCancelBody struct {
	OperationTime Time          `json:"operationTime"`
	RequestTime   Time          `json:"requestTime"`
	Comment       CancelComment `json:"comment"`
	ReceiptUUID   string        `json:"receiptUuid"`
	PartnerCode   *string       `json:"partnerCode"`
}

// IncomeCancelled describes a cancelled receipt. The API nests it under an
// "incomeInfo" key, which this type unwraps.
type IncomeCancelled struct {
	ApprovedReceiptUUID string            `json:"approvedReceiptUuid"`
	Name                string            `json:"name"`
	OperationTime       Time              `json:"operationTime"`
	RequestTime         Time              `json:"requestTime"`
	PaymentType         PaymentType       `json:"paymentType"`
	PartnerCode         string            `json:"partnerCode"`
	TotalAmount         decimal.Decimal   `json:"totalAmount"`
	CancellationInfo    *CancellationInfo `json:"cancellationInfo"`
	SourceDeviceID      string            `json:"sourceDeviceId"`
}

// UnmarshalJSON implements json.Unmarshaler, unwrapping the "incomeInfo" envelope.
func (c *IncomeCancelled) UnmarshalJSON(data []byte) error {
	type alias IncomeCancelled // Avoid recursing back into this method.
	envelope := struct {
		IncomeInfo *alias `json:"incomeInfo"`
	}{}

	if err := json.Unmarshal(data, &envelope); err != nil {
		return errors.Wrap(err, "moynalog: cannot decode cancelled income")
	}
	if envelope.IncomeInfo == nil {
		return errors.New("moynalog: cancelled income response has no incomeInfo")
	}
	*c = IncomeCancelled(*envelope.IncomeInfo)

	return nil
}

// Cancel cancels a registered receipt.
//
// POST /cancel
func (s *IncomeService) Cancel(ctx context.Context, income *IncomeCancelRequest) (*IncomeCancelled, *Response, error) {
	if income == nil {
		return nil, nil, errors.New("moynalog: income cancel request cannot be nil")
	}
	if income.ReceiptUUID == "" {
		return nil, nil, errors.New("moynalog: receipt UUID cannot be empty")
	}
	if !income.Comment.Valid() {
		return nil, nil, errors.Errorf("moynalog: cancel comment %q is invalid", income.Comment)
	}

	now := time.Now()
	operationTime := income.OperationTime
	if operationTime.IsZero() {
		operationTime = now
	}
	requestTime := income.RequestTime
	if requestTime.IsZero() {
		requestTime = now
	}

	body := &incomeCancelBody{
		OperationTime: NewTime(operationTime),
		RequestTime:   NewTime(requestTime),
		Comment:       income.Comment,
		ReceiptUUID:   income.ReceiptUUID,
		PartnerCode:   nullableString(income.PartnerCode),
	}

	req, err := s.client.NewRequest(http.MethodPost, "cancel", body)
	if err != nil {
		return nil, nil, err
	}

	cancelled := new(IncomeCancelled)
	resp, err := s.client.Do(ctx, req, cancelled)
	if err != nil {
		return nil, resp, err
	}

	return cancelled, resp, nil
}

// nullableString returns nil for the empty string so it encodes as JSON null,
// which is what the API expects for absent optional fields.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}

	return value
}
