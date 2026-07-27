package moynalog

import (
	"context"
	"net/http"
	"time"
)

// authReferrer is sent with authentication requests; the API rejects some of
// them without it.
var authHeaders = map[string]string{
	"Referrer": "https://lknpd.nalog.ru/auth/login",
}

// AuthService handles obtaining and refreshing access tokens.
type AuthService service

// AccessToken is the credential returned by every authentication endpoint.
// Persist it whole: Token alone cannot be refreshed.
type AccessToken struct {
	Token         string `json:"token"`
	TokenExpireIn Time   `json:"tokenExpireIn"`
	//nolint:gosec // G117: the JSON key is dictated by the API wire format.
	RefreshToken string `json:"refreshToken"`
	// RefreshTokenExpiresIn is null for tokens that never expire.
	RefreshTokenExpiresIn Time `json:"refreshTokenExpiresIn,omitempty"`
	// Profile is only populated by the endpoints that issue a fresh token.
	Profile User `json:"profile,omitempty"`
}

// IsExpired reports whether the access token can no longer be used.
func (t *AccessToken) IsExpired() bool {
	if t == nil || t.Token == "" {
		return true
	}
	if t.TokenExpireIn.IsZero() {
		return false
	}

	return !time.Now().Before(t.TokenExpireIn.Time)
}

// IsRefreshExpired reports whether the refresh token can no longer be
// exchanged for a new access token.
func (t *AccessToken) IsRefreshExpired() bool {
	if t == nil || t.RefreshToken == "" {
		return true
	}
	if t.RefreshTokenExpiresIn.IsZero() {
		return false
	}

	return !time.Now().Before(t.RefreshTokenExpiresIn.Time)
}

// PhoneChallenge is the pending SMS verification returned by CreatePhoneChallenge.
type PhoneChallenge struct {
	ChallengeToken string `json:"challengeToken"`
	ExpireDate     Time   `json:"expireDate"`
	ExpireIn       int    `json:"expireIn"`
}

// CreateAccessToken exchanges an INN (or login) and password for an access token.
func (s *AuthService) CreateAccessToken(ctx context.Context, username, password string) (*AccessToken, *Response, error) {
	deviceInfo, err := s.deviceInfo()
	if err != nil {
		return nil, nil, err
	}

	body := struct {
		Username string `json:"username"`
		//nolint:gosec // G117: the JSON key is dictated by the API wire format.
		Password   string      `json:"password"`
		DeviceInfo *DeviceInfo `json:"deviceInfo"`
	}{
		Username:   username,
		Password:   password,
		DeviceInfo: deviceInfo,
	}

	req, err := s.client.NewRequest(http.MethodPost, "auth/lkfl", body)
	if err != nil {
		return nil, nil, err
	}
	setAuthHeaders(req)

	return s.do(ctx, req)
}

// CreatePhoneChallenge asks the API to text a verification code to phone. It is
// the first of the two steps of phone authentication; pass the returned
// ChallengeToken and the code from the SMS to CreateAccessTokenByPhone.
//
// The API rate limits these messages to roughly one per one or two minutes
// until a verification succeeds.
func (s *AuthService) CreatePhoneChallenge(ctx context.Context, phone string) (*PhoneChallenge, *Response, error) {
	body := struct {
		Phone               string `json:"phone"`
		RequireTpToBeActive bool   `json:"requireTpToBeActive"`
	}{
		Phone:               phone,
		RequireTpToBeActive: true,
	}

	// The SMS challenge endpoints only exist on v2.
	req, err := s.client.newVersionedRequest(http.MethodPost, phoneAuthVersion, "auth/challenge/sms/start", body)
	if err != nil {
		return nil, nil, err
	}
	setAuthHeaders(req)

	challenge := new(PhoneChallenge)
	resp, err := s.client.Do(withoutAuth(ctx), req, challenge)
	if err != nil {
		return nil, resp, err
	}

	return challenge, resp, nil
}

// CreateAccessTokenByPhone completes phone authentication with the code from
// the SMS and the challenge token returned by CreatePhoneChallenge.
func (s *AuthService) CreateAccessTokenByPhone(ctx context.Context, phone, challengeToken, verificationCode string) (*AccessToken, *Response, error) {
	deviceInfo, err := s.deviceInfo()
	if err != nil {
		return nil, nil, err
	}

	body := struct {
		Phone          string      `json:"phone"`
		Code           string      `json:"code"`
		ChallengeToken string      `json:"challengeToken"`
		DeviceInfo     *DeviceInfo `json:"deviceInfo"`
	}{
		Phone:          phone,
		Code:           verificationCode,
		ChallengeToken: challengeToken,
		DeviceInfo:     deviceInfo,
	}

	req, err := s.client.NewRequest(http.MethodPost, "auth/challenge/sms/verify", body)
	if err != nil {
		return nil, nil, err
	}
	setAuthHeaders(req)

	return s.do(ctx, req)
}

// Refresh exchanges the refresh token of token for a new access token. The
// client calls it automatically when a request comes back 401, so it is rarely
// needed directly.
func (s *AuthService) Refresh(ctx context.Context, token *AccessToken) (*AccessToken, *Response, error) {
	if token == nil || token.RefreshToken == "" {
		return nil, nil, errNoAccessToken
	}
	if token.IsRefreshExpired() {
		return nil, nil, ErrUnauthorized
	}

	deviceInfo, err := s.deviceInfo()
	if err != nil {
		return nil, nil, err
	}

	body := struct {
		DeviceInfo *DeviceInfo `json:"deviceInfo"`
		//nolint:gosec // G117: the JSON key is dictated by the API wire format.
		RefreshToken string `json:"refreshToken"`
	}{
		DeviceInfo:   deviceInfo,
		RefreshToken: token.RefreshToken,
	}

	req, err := s.client.NewRequest(http.MethodPost, "auth/token", body)
	if err != nil {
		return nil, nil, err
	}
	setAuthHeaders(req)

	refreshed, resp, err := s.do(ctx, req)
	if err != nil {
		return nil, resp, err
	}

	// Refresh responses omit the refresh token and the profile; carry the ones
	// we already hold over so the result stays usable on its own.
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = token.RefreshToken
		refreshed.RefreshTokenExpiresIn = token.RefreshTokenExpiresIn
	}
	if refreshed.Profile.Inn == "" {
		refreshed.Profile = token.Profile
	}

	return refreshed, resp, nil
}

// do sends an authentication request, which must never carry or refresh a token.
func (s *AuthService) do(ctx context.Context, req *http.Request) (*AccessToken, *Response, error) {
	token := new(AccessToken)
	resp, err := s.client.Do(withoutAuth(ctx), req, token)
	if err != nil {
		return nil, resp, err
	}

	return token, resp, nil
}

func (s *AuthService) deviceInfo() (*DeviceInfo, error) {
	if s.client.deviceIDErr != nil {
		return nil, s.client.deviceIDErr
	}

	return NewDeviceInfo(s.client.DeviceID(), s.client.UserAgent), nil
}

func setAuthHeaders(req *http.Request) {
	for name, value := range authHeaders {
		req.Header.Set(name, value)
	}
}
