package moynalog

import (
	"context"
	"net/http"
)

// UsersService handles the taxpayer profile endpoints.
type UsersService service

// User is the profile of the authenticated taxpayer.
type User struct {
	ID                       int    `json:"id"`
	DisplayName              string `json:"displayName"`
	LastName                 string `json:"lastName,omitempty"`
	MiddleName               string `json:"middleName,omitempty"`
	Email                    string `json:"email,omitempty"`
	Phone                    string `json:"phone"`
	Inn                      string `json:"inn"`
	Snils                    string `json:"snils,omitempty"`
	AvatarExists             bool   `json:"avatarExists"`
	InitialRegistrationDate  Time   `json:"initialRegistrationDate,omitempty"`
	RegistrationDate         Time   `json:"registrationDate,omitempty"`
	FirstReceiptRegisterTime Time   `json:"firstReceiptRegisterTime,omitempty"`
	FirstReceiptCancelTime   Time   `json:"firstReceiptCancelTime,omitempty"`
	HideCancelledReceipt     bool   `json:"hideCancelledReceipt"`
	RegisterAvailable        any    `json:"registerAvailable"`
	Status                   string `json:"status"`
	RestrictedMode           bool   `json:"restrictedMode"`
	PfrURL                   string `json:"pfrUrl,omitempty"`
	Login                    string `json:"login,omitempty"`
}

// Get returns the profile of the authenticated taxpayer.
//
// GET /user
func (s *UsersService) Get(ctx context.Context) (*User, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "user", nil)
	if err != nil {
		return nil, nil, err
	}

	user := new(User)
	resp, err := s.client.Do(ctx, req, user)
	if err != nil {
		return nil, resp, err
	}

	return user, resp, nil
}
