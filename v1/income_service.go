package moynalog

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"net/http"
	"strconv"
	"time"
)

type IncomeCreateService struct {
	c             *Client
	serviceItems  []incomeServiceItem
	operationTime time.Time
	incomeClient  IncomeClient
}

type incomeRequest struct {
	PaymentType                     paymentType         `json:"paymentType"`
	IgnoreMaxTotalIncomeRestriction bool                `json:"ignoreMaxTotalIncomeRestriction"`
	Client                          IncomeClient        `json:"client"`
	RequestTime                     string              `json:"requestTime"`
	OperationTime                   string              `json:"operationTime"`
	Services                        []incomeServiceItem `json:"services"`
	TotalAmount                     string              `json:"totalAmount"`
}

type IncomeClient struct {
	ContactPhone *string    `json:"contactPhone"`
	DisplayName  *string    `json:"displayName"`
	IncomeType   incomeType `json:"incomeType"`
	Inn          *string    `json:"inn"`
}

type incomeServiceItem struct {
	Name     string          `json:"name"`
	Amount   decimal.Decimal `json:"amount"`
	Quantity decimal.Decimal `json:"quantity"`
}

type IncomeResponse struct {
	ApprovedReceiptUuid string `json:"approvedReceiptUuid"`
}

func (s *IncomeCreateService) AddItem(name string, amount decimal.Decimal, quantity int64) *IncomeCreateService {
	s.serviceItems = append(s.serviceItems, incomeServiceItem{name, amount, decimal.NewFromInt(quantity)})
	return s
}

func (s *IncomeCreateService) WithClient(contactPhone, displayName string, clientType incomeType, inn string) *IncomeCreateService {
	s.incomeClient = IncomeClient{
		ContactPhone: &contactPhone,
		DisplayName:  &displayName,
		IncomeType:   clientType,
		Inn:          &inn,
	}

	return s
}

func (s *IncomeCreateService) WithOperationTime(time time.Time) *IncomeCreateService {
	s.operationTime = time
	return s
}

func (s *IncomeCreateService) validate() error {
	if len(s.serviceItems) == 0 {
		return errors.Errorf("ServiceItems cannot be empty")
	}
	for key, serviceItem := range s.serviceItems {
		if serviceItem.Name == "" {
			return errors.Errorf("Name of item[%d] cannot be empty", key)
		}
		if serviceItem.Quantity.LessThanOrEqual(decimal.NewFromInt(0)) {
			return errors.Errorf("Quantity of item[%d] must be greater than %d", key, 0)
		}
		if serviceItem.Amount.LessThanOrEqual(decimal.NewFromInt(0)) {
			return errors.Errorf("Amount of item[%d] must be greater than %d", key, 0)
		}
	}

	if s.incomeClient.IncomeType == LegalEntity {
		if s.incomeClient.Inn == nil {
			return errors.Errorf("Client INN cannot be empty")
		}
		if _, err := strconv.ParseInt(*s.incomeClient.Inn, 10, 64); err != nil {
			return errors.Errorf("Client INN must contain only numbers")
		}
		if len(*s.incomeClient.Inn) != 10 || len(*s.incomeClient.Inn) != 12 {
			return errors.Errorf("Client INN length must been 10 or 12")
		}
		if s.incomeClient.DisplayName == nil {
			return errors.Errorf("Client DisplayName cannot be empty")
		}
	}

	return nil
}

// Do Send request
func (s *IncomeCreateService) Do(ctx context.Context, opts ...RequestOption) (*IncomeResponse, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}

	totalAmount := decimal.NewFromInt(0)

	for _, serviceItem := range s.serviceItems {
		totalAmount = totalAmount.Add(serviceItem.Amount.Mul(serviceItem.Quantity))
	}

	r := &request{
		method:   http.MethodPost,
		endpoint: "/income",
		json: incomeRequest{
			PaymentType:   Cash,
			Client:        s.incomeClient,
			RequestTime:   time.Time{}.Format(time.RFC3339),
			OperationTime: s.operationTime.Format(time.RFC3339),
			Services:      s.serviceItems,
			TotalAmount:   totalAmount.String(),

			IgnoreMaxTotalIncomeRestriction: false,
		},
	}
	data, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	res := new(IncomeResponse)

	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
