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

type PaymentType string

const (
	Cash    PaymentType = "CASH"
	Account PaymentType = "ACCOUNT"
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
	Quantity int64           `json:"quantity"`
}

type IncomeCreate struct {
	ApprovedReceiptUUID string `json:"approvedReceiptUuid"`
}

type IncomeCreateRequest struct {
	PaymentType   PaymentType          `json:"paymentType"`
	Client        *IncomeClient        `json:"client"`
	RequestTime   time.Time            `json:"requestTime"`
	OperationTime time.Time            `json:"operationTime"`
	Services      []*IncomeServiceItem `json:"services"`
	TotalAmount   string               `json:"totalAmount"`

	IgnoreMaxTotalIncomeRestriction bool `json:"ignoreMaxTotalIncomeRestriction"`
}

func validateCreateIncome(income *IncomeCreateRequest) error {
	if len(income.Services) == 0 {
		return errors.Errorf("ServiceItems cannot be empty")
	}
	for key, serviceItem := range income.Services {
		if serviceItem.Name == "" {
			return errors.Errorf("Name of item[%d] cannot be empty", key)
		}
		if serviceItem.Quantity <= 0 {
			return errors.Errorf("Quantity of item[%d] must be greater than %d", key, 0)
		}
		if serviceItem.Amount.LessThanOrEqual(decimal.NewFromInt(0)) {
			return errors.Errorf("Amount of item[%d] must be greater than %d", key, 0)
		}
	}

	if income.Client.IncomeType == LegalEntity {
		if income.Client.Inn == "" {
			return errors.Errorf("Clientt INN cannot be empty")
		}
		if _, err := strconv.ParseInt(income.Client.Inn, 10, 64); err != nil {
			return errors.Errorf("Clientt INN must contain only numbers")
		}
		if len(income.Client.Inn) != 10 || len(income.Client.Inn) != 12 {
			return errors.Errorf("Clientt INN length must been 10 or 12")
		}
		if income.Client.DisplayName == "" {
			return errors.Errorf("Client DisplayName cannot be empty")
		}
	}

	return nil
}

func (s *IncomeService) Create(ctx context.Context, income *IncomeCreateRequest) (*IncomeCreate, error) {
	totalAmount := decimal.NewFromInt(0)
	for _, serviceItem := range income.Services {
		totalAmount = totalAmount.Add(serviceItem.Amount.Mul(decimal.NewFromInt(serviceItem.Quantity)))
	}

	if income.PaymentType == "" {
		income.PaymentType = Cash
	}

	income.TotalAmount = totalAmount.String()

	income.RequestTime = time.Now().Truncate(time.Millisecond)
	if income.OperationTime.IsZero() {
		income.OperationTime = income.RequestTime
	}
	income.OperationTime = income.OperationTime.Truncate(time.Millisecond)

	err := validateCreateIncome(income)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest(http.MethodPost, "income", income)
	icResp := new(IncomeCreate)
	_, err = s.client.Do(ctx, req, icResp)
	if err != nil {
		return nil, err
	}

	return icResp, err
}

type CancelComment string

const (
	Cancel CancelComment = "Чек сформирован ошибочно"
	Refund CancelComment = "Возврат средств"
)

type IncomeCancelRequest struct {
	RequestTime   time.Time     `json:"requestTime"`
	OperationTime time.Time     `json:"operationTime"`
	Comment       CancelComment `json:"comment"`
	ReceiptUUID   string        `json:"receiptUuid"`
	PartnerCode   string        `json:"partnerCode"`
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

func (s *IncomeService) Cancel(ctx context.Context, income *IncomeCancelRequest) (*IncomeCancel, error) {
	income.RequestTime = time.Now().Truncate(time.Millisecond)
	if income.OperationTime.IsZero() {
		income.OperationTime = income.RequestTime
	}
	income.OperationTime = income.OperationTime.Truncate(time.Millisecond)

	req, err := s.client.NewRequest(http.MethodPost, "income", income)
	icResp := new(IncomeCancel)
	_, err = s.client.Do(ctx, req, icResp)
	if err != nil {
		return nil, err
	}

	return icResp, err
}
