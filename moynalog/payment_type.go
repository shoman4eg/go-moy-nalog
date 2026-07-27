package moynalog

import (
	"context"
	"net/http"
)

// PaymentTypeService handles the registered bank accounts a taxpayer can be
// paid through.
type PaymentTypeService service

// BankAccount is a payment method registered by the taxpayer.
type BankAccount struct {
	ID             int    `json:"id"`
	Type           string `json:"type"`
	BankName       string `json:"bankName"`
	BankBik        string `json:"bankBik"`
	CurrentAccount string `json:"currentAccount"`
	CorrAccount    string `json:"corrAccount"`
	Phone          string `json:"phone"`
	BankID         string `json:"bankId"`
	Favorite       bool   `json:"favorite"`
	AvailableForPa bool   `json:"availableForPa"`
}

// Table returns every payment method registered by the taxpayer.
//
// GET /payment-type/table
func (s *PaymentTypeService) Table(ctx context.Context) ([]*BankAccount, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "payment-type/table", nil)
	if err != nil {
		return nil, nil, err
	}

	table := struct {
		Items []*BankAccount `json:"items"`
	}{}
	resp, err := s.client.Do(ctx, req, &table)
	if err != nil {
		return nil, resp, err
	}

	return table.Items, resp, nil
}

// Favorite returns the payment method the taxpayer marked as preferred, or nil
// when none is marked.
func (s *PaymentTypeService) Favorite(ctx context.Context) (*BankAccount, *Response, error) {
	accounts, resp, err := s.Table(ctx)
	if err != nil {
		return nil, resp, err
	}

	for _, account := range accounts {
		if account.Favorite {
			return account, resp, nil
		}
	}

	return nil, resp, nil
}
