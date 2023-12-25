package moynalog

import (
	"context"
	"fmt"
	"time"
)

type ReceiptService service

type Receipt struct {
	ReceiptID string `json:"receiptId"`
	Services  []struct {
		Name          string  `json:"name"`
		Quantity      int     `json:"quantity"`
		ServiceNumber int     `json:"serviceNumber"`
		Amount        float64 `json:"amount"`
	} `json:"services"`
	OperationTime      time.Time `json:"operationTime"`
	RequestTime        time.Time `json:"requestTime"`
	RegisterTime       time.Time `json:"registerTime"`
	TaxPeriodID        int       `json:"taxPeriodId"`
	PaymentType        string    `json:"paymentType"`
	IncomeType         string    `json:"incomeType"`
	TotalAmount        int       `json:"totalAmount"`
	CancellationInfo   any       `json:"cancellationInfo"`
	SourceDeviceID     any       `json:"sourceDeviceId"`
	ClientInn          any       `json:"clientInn"`
	ClientDisplayName  string    `json:"clientDisplayName"`
	PartnerDisplayName string    `json:"partnerDisplayName"`
	PartnerInn         string    `json:"partnerInn"`
	Inn                string    `json:"inn"`
	Profession         string    `json:"profession"`
	Description        []any     `json:"description"`
	Email              any       `json:"email"`
	Phone              any       `json:"phone"`
	InvoiceID          any       `json:"invoiceId"`
}

func (s *ReceiptService) JSON(ctx context.Context, receiptUUID string) (*Receipt, error) {
	token := s.client.AccessToken
	if token == nil {
		return nil, errAccessTokenIsEmpty
	}

	inn := token.Profile.Inn
	u := fmt.Sprintf("receipt/%v/%v/json", inn, receiptUUID)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	receipt := new(Receipt)
	_, err = s.client.Do(ctx, req, &receipt)
	if err != nil {
		return nil, err
	}

	return receipt, nil
}
