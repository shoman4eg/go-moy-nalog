package moynalog

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

// ReceiptService handles fetching issued receipts.
type ReceiptService service

// Receipt is the full record of a registered receipt.
type Receipt struct {
	ReceiptID          string            `json:"receiptId"`
	Services           []*ServiceItem    `json:"services"`
	OperationTime      Time              `json:"operationTime"`
	RequestTime        Time              `json:"requestTime"`
	RegisterTime       Time              `json:"registerTime"`
	TaxPeriodID        int               `json:"taxPeriodId"`
	PaymentType        PaymentType       `json:"paymentType"`
	IncomeType         IncomeType        `json:"incomeType"`
	TotalAmount        decimal.Decimal   `json:"totalAmount"`
	CancellationInfo   *CancellationInfo `json:"cancellationInfo"`
	SourceDeviceID     string            `json:"sourceDeviceId"`
	ClientInn          string            `json:"clientInn"`
	ClientDisplayName  string            `json:"clientDisplayName"`
	PartnerDisplayName string            `json:"partnerDisplayName"`
	PartnerInn         string            `json:"partnerInn"`
	Inn                string            `json:"inn"`
	Profession         string            `json:"profession"`
	Description        []string          `json:"description"`
	Email              string            `json:"email"`
	Phone              string            `json:"phone"`
	InvoiceID          string            `json:"invoiceId"`
}

// Cancelled reports whether the receipt has been cancelled.
func (r *Receipt) Cancelled() bool {
	return r.CancellationInfo != nil
}

// JSON returns the receipt identified by receiptUUID.
//
// GET /receipt/{inn}/{receiptUuid}/json
func (s *ReceiptService) JSON(ctx context.Context, receiptUUID string) (*Receipt, *Response, error) {
	u, err := s.receiptPath(ctx, receiptUUID, "json")
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	receipt := new(Receipt)
	resp, err := s.client.Do(ctx, req, receipt)
	if err != nil {
		return nil, resp, err
	}

	return receipt, resp, nil
}

// PrintURL returns the absolute URL of the printable receipt. The URL is not
// public: fetch it with an authenticated client, or use Print.
func (s *ReceiptService) PrintURL(ctx context.Context, receiptUUID string) (string, error) {
	u, err := s.receiptPath(ctx, receiptUUID, "print")
	if err != nil {
		return "", err
	}

	resolved, err := s.client.BaseURL.Parse(u)
	if err != nil {
		return "", errors.Wrap(err, "moynalog: cannot build receipt print URL")
	}

	return resolved.String(), nil
}

// Print downloads the printable receipt, which the API renders as a PDF.
//
// GET /receipt/{inn}/{receiptUuid}/print
func (s *ReceiptService) Print(ctx context.Context, receiptUUID string) ([]byte, *Response, error) {
	u, err := s.receiptPath(ctx, receiptUUID, "print")
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	buf := new(bytes.Buffer)
	resp, err := s.client.Do(ctx, req, buf)
	if err != nil {
		return nil, resp, err
	}

	return buf.Bytes(), resp, nil
}

// receiptPath builds a receipt path, resolving the taxpayer INN the API keys
// receipts by. It is taken from the token profile when present, and fetched
// otherwise.
func (s *ReceiptService) receiptPath(ctx context.Context, receiptUUID, action string) (string, error) {
	if receiptUUID == "" {
		return "", errors.New("moynalog: receipt UUID cannot be empty")
	}

	inn, err := s.inn(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("receipt/%s/%s/%s", inn, receiptUUID, action), nil
}

func (s *ReceiptService) inn(ctx context.Context) (string, error) {
	if token := s.client.Token(); token != nil && token.Profile.Inn != "" {
		return token.Profile.Inn, nil
	}

	user, _, err := s.client.Users.Get(ctx)
	if err != nil {
		return "", err
	}
	if user.Inn == "" {
		return "", errors.New("moynalog: taxpayer profile has no INN")
	}

	return user.Inn, nil
}
