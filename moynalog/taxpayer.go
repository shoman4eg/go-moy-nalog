package moynalog

import (
	"context"
	"net/http"

	"github.com/shopspring/decimal"
)

// TaxpayerService handles the taxpayer standing endpoints.
type TaxpayerService service

// Debts is the outstanding tax debt of the taxpayer.
type Debts struct {
	HasDebts    bool            `json:"hasDebts"`
	TotalUnpaid decimal.Decimal `json:"totalUnpaid"`
	Debts       decimal.Decimal `json:"debts"`
}

// AnnualIncome is the income recorded against a single year.
type AnnualIncome struct {
	TotalIncomeAmount                decimal.Decimal `json:"totalIncomeAmount"`
	MaxTotalIncomeThresholdExceeded  bool            `json:"maxTotalIncomeThresholdExceeded"`
	AnnualIncomeThreshold            decimal.Decimal `json:"annualIncomeThreshold"`
	AvailableIncomeToExceedThreshold decimal.Decimal `json:"availableIncomeToExceedThreshold"`
	AnnualIncomeStatus               string          `json:"annualIncomeStatus"`
	UpdatedTime                      Time            `json:"updatedTime"`
}

// Bonus is the remaining tax deduction ("налоговый бонус") together with the
// annual income the threshold is measured against.
type Bonus struct {
	BonusAmount                      decimal.Decimal          `json:"bonusAmount"`
	TotalIncomeAmount                decimal.Decimal          `json:"totalIncomeAmount"`
	MaxTotalIncomeThresholdExceeded  bool                     `json:"maxTotalIncomeThresholdExceeded"`
	AnnualIncomeThreshold            decimal.Decimal          `json:"annualIncomeThreshold"`
	AvailableIncomeToExceedThreshold decimal.Decimal          `json:"availableIncomeToExceedThreshold"`
	AnnualIncomeStatus               string                   `json:"annualIncomeStatus"`
	UpdatedTime                      Time                     `json:"updatedTime"`
	TotalIncomeByYears               map[string]*AnnualIncome `json:"totalIncomeByYears"`
	TeenBonusAmount                  decimal.Decimal          `json:"teenBonusAmount"`
	TeenBonusUpdatedTime             Time                     `json:"teenBonusUpdatedTime"`
}

// Debts returns the outstanding tax debt.
//
// GET /taxpayer/debts
func (s *TaxpayerService) Debts(ctx context.Context) (*Debts, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "taxpayer/debts", nil)
	if err != nil {
		return nil, nil, err
	}

	debts := new(Debts)
	resp, err := s.client.Do(ctx, req, debts)
	if err != nil {
		return nil, resp, err
	}

	return debts, resp, nil
}

// Bonus returns the remaining tax deduction and the annual income totals.
//
// GET /taxpayer/bonus
func (s *TaxpayerService) Bonus(ctx context.Context) (*Bonus, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "taxpayer/bonus", nil)
	if err != nil {
		return nil, nil, err
	}

	bonus := new(Bonus)
	resp, err := s.client.Do(ctx, req, bonus)
	if err != nil {
		return nil, resp, err
	}

	return bonus, resp, nil
}
