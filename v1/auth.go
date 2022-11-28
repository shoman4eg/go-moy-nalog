package moynalog

import (
	"context"
	"net/http"
	"time"
)

var authHeaders = map[string]string{
	"Referrer":        "https://lknpd.nalog.ru/",
	"Referrer-Policy": "strict-origin-when-cross-origin",
}

type AuthSerive service

type AccessToken struct {
	RefreshToken          string    `json:"refreshToken"`
	RefreshTokenExpiresIn time.Time `json:"refreshTokenExpiresIn,omitempty"`
	Token                 string    `json:"token"`
	TokenExpireIn         time.Time `json:"tokenExpireIn"`
	Profile               User      `json:"profile,omitempty"`
}

func (s *AuthSerive) CreateAccessToken(ctx context.Context, username, password string) (*AccessToken, *Response, error) {
	di := NewDeviceInfo(generateDeviceID())
	di.MetaDetails.UserAgent = s.client.UserAgent
	reqBody := struct {
		Username   string      `json:"username"`
		Password   string      `json:"password"`
		DeviceInfo *DeviceInfo `json:"deviceInfo"`
	}{
		Username:   username,
		Password:   password,
		DeviceInfo: di,
	}

	req, err := s.client.NewRequest(http.MethodPost, "auth/lkfl", reqBody)
	if err != nil {
		return nil, nil, err
	}

	for k, v := range authHeaders {
		req.Header.Set(k, v)
	}

	atResp := new(AccessToken)
	resp, err := s.client.Do(ctx, req, atResp)
	if err != nil {
		return nil, resp, err
	}

	return atResp, resp, err
}

func (s *AuthSerive) CreatePhoneChallenge(phone string) {
}
