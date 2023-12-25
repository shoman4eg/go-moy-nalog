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

func (t *AccessToken) IsExpired() bool {
	return t.TokenExpireIn.After(time.Now())
}

func (s *AuthSerive) CreateAccessToken(ctx context.Context, username, password string) (*AccessToken, error) {
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
		return nil, err
	}

	for k, v := range authHeaders {
		req.Header.Set(k, v)
	}

	atResp := new(AccessToken)
	_, err = s.client.Do(ctx, req, atResp)
	if err != nil {
		return nil, err
	}

	return atResp, err
}

func (s *AuthSerive) RefreshToken(ctx context.Context, token *AccessToken) (*AccessToken, error) {
	if token == nil {
		return nil, errAccessTokenIsEmpty
	}
	if token.IsExpired() {
		return token, nil
	}
	if token.RefreshTokenExpiresIn.After(time.Now()) {
		return nil, errRefreshTokenIsExpired
	}
	di := NewDeviceInfo(generateDeviceID())
	di.MetaDetails.UserAgent = s.client.UserAgent
	reqBody := struct {
		RefreshToken string      `json:"refreshToken"`
		DeviceInfo   *DeviceInfo `json:"deviceInfo"`
	}{
		DeviceInfo:   di,
		RefreshToken: token.RefreshToken,
	}

	req, err := s.client.NewRequest(http.MethodPost, "auth/token", reqBody)
	if err != nil {
		return nil, err
	}

	for k, v := range authHeaders {
		req.Header.Set(k, v)
	}

	atResp := new(AccessToken)
	_, err = s.client.Do(ctx, req, atResp)
	if err != nil {
		return nil, err
	}

	return atResp, err
}
