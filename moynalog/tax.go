package moynalog

import (
	"context"
	"net/http"

	"github.com/shopspring/decimal"
)

// TaxService handles the tax accrual and payment endpoints.
type TaxService service

// Tax is the current tax position of the taxpayer.
type Tax struct {
	TotalForPayment    decimal.Decimal `json:"totalForPayment"`
	Total              decimal.Decimal `json:"total"`
	Tax                decimal.Decimal `json:"tax"`
	Debt               decimal.Decimal `json:"debt"`
	Overpayment        decimal.Decimal `json:"overpayment"`
	Penalty            decimal.Decimal `json:"penalty"`
	NominalTax         decimal.Decimal `json:"nominalTax"`
	NominalOverpayment decimal.Decimal `json:"nominalOverpayment"`
	TaxPeriodID        int             `json:"taxPeriodId"`
	LastPaymentAmount  decimal.Decimal `json:"lastPaymentAmount"`
	LastPaymentDate    Time            `json:"lastPaymentDate"`
	Regions            []any           `json:"regions"`
}

// TaxHistoryRecord is a single tax accrual.
type TaxHistoryRecord struct {
	TaxPeriodID     int             `json:"taxPeriodId"`
	TaxAmount       decimal.Decimal `json:"taxAmount"`
	BonusAmount     decimal.Decimal `json:"bonusAmount"`
	PaidAmount      decimal.Decimal `json:"paidAmount"`
	TaxBaseAmount   decimal.Decimal `json:"taxBaseAmount"`
	ChargeDate      Time            `json:"chargeDate"`
	DueDate         Time            `json:"dueDate"`
	Oktmo           string          `json:"oktmo"`
	RegionName      string          `json:"regionName"`
	Kbk             string          `json:"kbk"`
	TaxOrganCode    string          `json:"taxOrganCode"`
	Type            string          `json:"type"`
	KrsbTaxChargeID int             `json:"krsbTaxChargeId"`
	ReceiptCount    int             `json:"receiptCount"`
}

// TaxPayment is a single tax payment.
type TaxPayment struct {
	SourceType       string          `json:"sourceType"`
	Type             string          `json:"type"`
	DocumentIndex    string          `json:"documentIndex"`
	Amount           decimal.Decimal `json:"amount"`
	OperationDate    Time            `json:"operationDate"`
	DueDate          Time            `json:"dueDate"`
	Oktmo            string          `json:"oktmo"`
	Kbk              string          `json:"kbk"`
	Status           string          `json:"status"`
	TaxPeriodID      int             `json:"taxPeriodId"`
	RegionName       string          `json:"regionName"`
	KrsbAcceptedDate Time            `json:"krsbAcceptedDate"`
}

// Get returns the current tax position.
//
// GET /taxes
func (s *TaxService) Get(ctx context.Context) (*Tax, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "taxes", nil)
	if err != nil {
		return nil, nil, err
	}

	tax := new(Tax)
	resp, err := s.client.Do(ctx, req, tax)
	if err != nil {
		return nil, resp, err
	}

	return tax, resp, nil
}

// History returns the tax accruals, optionally narrowed to a single OKTMO. Pass
// an empty oktmo for every region.
//
// POST /taxes/history
func (s *TaxService) History(ctx context.Context, oktmo string) ([]*TaxHistoryRecord, *Response, error) {
	body := struct {
		Oktmo *string `json:"oktmo"`
	}{Oktmo: nullableString(oktmo)}

	req, err := s.client.NewRequest(http.MethodPost, "taxes/history", body)
	if err != nil {
		return nil, nil, err
	}

	records := struct {
		Records []*TaxHistoryRecord `json:"records"`
	}{}
	resp, err := s.client.Do(ctx, req, &records)
	if err != nil {
		return nil, resp, err
	}

	return records.Records, resp, nil
}

// Payments returns the tax payments, optionally narrowed to a single OKTMO and
// to settled payments only. Pass an empty oktmo for every region.
//
// POST /taxes/payments
func (s *TaxService) Payments(ctx context.Context, oktmo string, onlyPaid bool) ([]*TaxPayment, *Response, error) {
	body := struct {
		Oktmo    *string `json:"oktmo"`
		OnlyPaid bool    `json:"onlyPaid"`
	}{
		Oktmo:    nullableString(oktmo),
		OnlyPaid: onlyPaid,
	}

	req, err := s.client.NewRequest(http.MethodPost, "taxes/payments", body)
	if err != nil {
		return nil, nil, err
	}

	records := struct {
		Records []*TaxPayment `json:"records"`
	}{}
	resp, err := s.client.Do(ctx, req, &records)
	if err != nil {
		return nil, resp, err
	}

	return records.Records, resp, nil
}
