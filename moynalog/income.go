package moynalog

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type IncomeService service

type IncomeType string

const (
	Individual    IncomeType = "FROM_INDIVIDUAL"
	LegalEntity   IncomeType = "FROM_LEGAL_ENTITY"
	ForeignAgency IncomeType = "FROM_FOREIGN_AGENCY"
)

type CancelComment string

const (
	Cancel CancelComment = "Чек сформирован ошибочно"
	Refund CancelComment = "Возврат средств"
)

type IncomeClient struct {
	ContactPhone string     `json:"contactPhone"`
	DisplayName  string     `json:"displayName"`
	IncomeType   IncomeType `json:"incomeType"`
	Inn          string     `json:"inn"`
}

type IncomeServiceItem struct {
	Name     string          `json:"name"`
	Amount   decimal.Decimal `json:"amount"`
	Quantity uint64          `json:"quantity"`
}

type incomeCreateRequest struct {
	PaymentType   paymentType         `json:"paymentType"`
	Client        IncomeClient        `json:"client"`
	RequestTime   string              `json:"requestTime"`
	OperationTime string              `json:"operationTime"`
	Services      []IncomeServiceItem `json:"services"`
	TotalAmount   string              `json:"totalAmount"`

	IgnoreMaxTotalIncomeRestriction bool `json:"ignoreMaxTotalIncomeRestriction"`
}

type IncomeCreate struct {
	ApprovedReceiptUUID string `json:"approvedReceiptUuid"`
}

type IncomeCancel struct {
	ApprovedReceiptUUID string    `json:"approvedReceiptUuid"`
	Name                string    `json:"name"`
	OperationTime       time.Time `json:"operationTime"`
	RequestTime         time.Time `json:"requestTime"`
	PaymentType         string    `json:"paymentType"`
	PartnerCode         string    `json:"partnerCode"`
	TotalAmount         string    `json:"totalAmount"`
	CancellationInfo    struct {
		OperationTime time.Time     `json:"operationTime"`
		RegisterTime  time.Time     `json:"registerTime"`
		TaxPeriodID   int           `json:"taxPeriodId"`
		Comment       CancelComment `json:"comment"`
	} `json:"cancellationInfo"`
}

type incomeCancelRequest struct {
	RequestTime   string        `json:"requestTime"`
	OperationTime string        `json:"operationTime"`
	Comment       CancelComment `json:"comment"`
	ReceiptUUID   string        `json:"receiptUuid"`
	PartnerCode   string        `json:"partnerCode"`
}

func (s *IncomeService) Create(
	ctx context.Context,
	items []IncomeServiceItem,
	client IncomeClient,
	operationTime time.Time,
) (*IncomeCreate, *Response, error) {
	totalAmount := decimal.NewFromInt(0)
	for _, serviceItem := range items {
		totalAmount = totalAmount.Add(serviceItem.Amount.Mul(decimal.NewFromInt(int64(serviceItem.Quantity))))
	}

	reqBody := incomeCreateRequest{
		PaymentType:   Cash,
		Client:        client,
		RequestTime:   time.Now().Format(time.RFC3339),
		OperationTime: operationTime.Format(time.RFC3339),
		Services:      items,
		TotalAmount:   totalAmount.String(),

		IgnoreMaxTotalIncomeRestriction: false,
	}

	err := validateCreateIncome(reqBody)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequestWithAuth(http.MethodPost, "income", reqBody)
	icResp := new(IncomeCreate)
	resp, err := s.client.Do(ctx, req, icResp)
	if err != nil {
		return nil, resp, err
	}

	return icResp, resp, err
}

func validateCreateIncome(reqBody incomeCreateRequest) error {
	if len(reqBody.Services) == 0 {
		return errors.Errorf("ServiceItems cannot be empty")
	}
	for key, serviceItem := range reqBody.Services {
		if serviceItem.Name == "" {
			return errors.Errorf("Name of item[%d] cannot be empty", key)
		}
		if serviceItem.Quantity == 0 {
			return errors.Errorf("Quantity of item[%d] must be greater than %d", key, 0)
		}
		if serviceItem.Amount.LessThanOrEqual(decimal.NewFromInt(0)) {
			return errors.Errorf("Amount of item[%d] must be greater than %d", key, 0)
		}
	}

	if reqBody.Client.IncomeType == LegalEntity {
		if reqBody.Client.Inn == "" {
			return errors.Errorf("Clientt INN cannot be empty")
		}
		if _, err := strconv.ParseInt(reqBody.Client.Inn, 10, 64); err != nil {
			return errors.Errorf("Clientt INN must contain only numbers")
		}
		if len(reqBody.Client.Inn) != 10 || len(reqBody.Client.Inn) != 12 {
			return errors.Errorf("Clientt INN length must been 10 or 12")
		}
		if reqBody.Client.DisplayName == "" {
			return errors.Errorf("Clientt DisplayName cannot be empty")
		}
	}

	return nil
}

func (s *IncomeService) Cancel(
	ctx context.Context,
	receiptUUID string,
	comment CancelComment,
	operationTime time.Time,
	partnerCode string,
) (*IncomeCancel, *Response, error) {
	reqBody := incomeCancelRequest{
		RequestTime:   time.Now().Format(time.RFC3339),
		OperationTime: operationTime.Format(time.RFC3339),
		Comment:       comment,
		ReceiptUUID:   receiptUUID,
		PartnerCode:   partnerCode,
	}

	req, err := s.client.NewRequestWithAuth(http.MethodPost, "income", reqBody)
	icResp := new(IncomeCancel)
	resp, err := s.client.Do(ctx, req, icResp)
	if err != nil {
		return nil, resp, err
	}

	return icResp, resp, err
}
