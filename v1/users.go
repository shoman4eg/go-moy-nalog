package moynalog

import (
	"context"
	"net/http"
	"time"
)

type UsersService service

type User struct {
	LastName                 string      `json:"lastName,omitempty"`
	ID                       int         `json:"id"`
	DisplayName              string      `json:"displayName"`
	MiddleName               string      `json:"middleName,omitempty"`
	Email                    string      `json:"email"`
	Phone                    string      `json:"phone"`
	Inn                      string      `json:"inn"`
	Snils                    string      `json:"snils"`
	AvatarExists             bool        `json:"avatarExists"`
	InitialRegistrationDate  time.Time   `json:"initialRegistrationDate,omitempty"`
	RegistrationDate         time.Time   `json:"registrationDate,omitempty"`
	FirstReceiptRegisterTime time.Time   `json:"firstReceiptRegisterTime,omitempty"`
	FirstReceiptCancelTime   time.Time   `json:"firstReceiptCancelTime,omitempty"`
	HideCancelledReceipt     bool        `json:"hideCancelledReceipt"`
	RegisterAvailable        interface{} `json:"registerAvailable"`
	Status                   string      `json:"status"`
	RestrictedMode           bool        `json:"restrictedMode"`
	PfrURL                   string      `json:"pfrUrl"`
	Login                    string      `json:"login,omitempty"`
}

func (s *UsersService) Get(ctx context.Context) (*User, *Response, error) {
	req, err := s.client.NewRequestWithAuth(http.MethodGet, "user", nil)
	if err != nil {
		return nil, nil, err
	}

	uResp := new(User)
	resp, err := s.client.Do(ctx, req, uResp)
	if err != nil {
		return nil, resp, err
	}

	return uResp, resp, err
}
