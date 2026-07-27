package moynalog

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func TestAuthCreateAccessToken(t *testing.T) {
	t.Parallel()

	client, mux := setup(t)
	mux.HandleFunc("/v1/auth/lkfl", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testHeader(t, r, "Referrer", "https://lknpd.nalog.ru/auth/login")

		body := testBody(t, r)
		if body["username"] != "770000000000" {
			t.Errorf("username = %v, want %q", body["username"], "770000000000")
		}
		if body["password"] != "secret" {
			t.Errorf("password = %v, want %q", body["password"], "secret")
		}

		deviceInfo, ok := body["deviceInfo"].(map[string]any)
		if !ok {
			t.Fatalf("deviceInfo = %v, want an object", body["deviceInfo"])
		}
		if deviceInfo["sourceDeviceId"] != "testdeviceid" {
			t.Errorf("sourceDeviceId = %v, want %q", deviceInfo["sourceDeviceId"], "testdeviceid")
		}
		if deviceInfo["sourceType"] != SourceTypeWeb {
			t.Errorf("sourceType = %v, want %q", deviceInfo["sourceType"], SourceTypeWeb)
		}

		writeJSON(t, w, http.StatusOK, `{
			"refreshToken": "refresh",
			"refreshTokenExpiresIn": null,
			"token": "access",
			"tokenExpireIn": "2222-02-01T00:47:30.446Z",
			"profile": {"id": 1, "inn": "770000000000", "displayName": "ПУПКИН"}
		}`)
	})

	token, _, err := client.Auth.CreateAccessToken(context.Background(), "770000000000", "secret")
	if err != nil {
		t.Fatalf("CreateAccessToken: %v", err)
	}

	if token.Token != "access" {
		t.Errorf("Token = %q, want %q", token.Token, "access")
	}
	if token.RefreshToken != "refresh" {
		t.Errorf("RefreshToken = %q, want %q", token.RefreshToken, "refresh")
	}
	if token.Profile.Inn != "770000000000" {
		t.Errorf("Profile.Inn = %q, want %q", token.Profile.Inn, "770000000000")
	}
	if token.IsExpired() {
		t.Error("a token expiring in 2222 must not read as expired")
	}
	// A null refreshTokenExpiresIn means the refresh token never expires.
	if token.IsRefreshExpired() {
		t.Error("a null refresh expiry must not read as expired")
	}
}

func TestAccessTokenIsExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token *AccessToken
		want  bool
	}{
		{"nil token", nil, true},
		{"empty token", new(AccessToken), true},
		{"no expiry", &AccessToken{Token: "t"}, false},
		{"future expiry", &AccessToken{Token: "t", TokenExpireIn: NewTime(time.Now().Add(time.Hour))}, false},
		{"past expiry", &AccessToken{Token: "t", TokenExpireIn: NewTime(time.Now().Add(-time.Hour))}, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.token.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccessTokenIsRefreshExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token *AccessToken
		want  bool
	}{
		{"nil token", nil, true},
		{"no refresh token", &AccessToken{Token: "t"}, true},
		{"no expiry", &AccessToken{RefreshToken: "r"}, false},
		{"future expiry", &AccessToken{RefreshToken: "r", RefreshTokenExpiresIn: NewTime(time.Now().Add(time.Hour))}, false},
		{"past expiry", &AccessToken{RefreshToken: "r", RefreshTokenExpiresIn: NewTime(time.Now().Add(-time.Hour))}, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.token.IsRefreshExpired(); got != tt.want {
				t.Errorf("IsRefreshExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

// The SMS challenge lives on v2 while everything else is v1.
func TestAuthCreatePhoneChallengeUsesV2(t *testing.T) {
	t.Parallel()

	client, mux := setup(t)
	mux.HandleFunc("/v2/auth/challenge/sms/start", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)

		body := testBody(t, r)
		if body["phone"] != "79000000000" {
			t.Errorf("phone = %v, want %q", body["phone"], "79000000000")
		}
		if body["requireTpToBeActive"] != true {
			t.Errorf("requireTpToBeActive = %v, want true", body["requireTpToBeActive"])
		}

		writeJSON(t, w, http.StatusOK, `{
			"challengeToken": "challenge",
			"expireDate": "2026-07-27T12:00:00Z",
			"expireIn": 60
		}`)
	})

	challenge, _, err := client.Auth.CreatePhoneChallenge(context.Background(), "79000000000")
	if err != nil {
		t.Fatalf("CreatePhoneChallenge: %v", err)
	}

	if challenge.ChallengeToken != "challenge" {
		t.Errorf("ChallengeToken = %q, want %q", challenge.ChallengeToken, "challenge")
	}
	if challenge.ExpireIn != 60 {
		t.Errorf("ExpireIn = %d, want 60", challenge.ExpireIn)
	}
	if challenge.ExpireDate.IsZero() {
		t.Error("ExpireDate must be decoded")
	}
}

func TestAuthCreateAccessTokenByPhone(t *testing.T) {
	t.Parallel()

	client, mux := setup(t)
	mux.HandleFunc("/v1/auth/challenge/sms/verify", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)

		body := testBody(t, r)
		if body["phone"] != "79000000000" {
			t.Errorf("phone = %v, want %q", body["phone"], "79000000000")
		}
		if body["code"] != "1234" {
			t.Errorf("code = %v, want %q", body["code"], "1234")
		}
		if body["challengeToken"] != "challenge" {
			t.Errorf("challengeToken = %v, want %q", body["challengeToken"], "challenge")
		}

		writeJSON(t, w, http.StatusOK, `{"token":"access","refreshToken":"refresh"}`)
	})

	token, _, err := client.Auth.CreateAccessTokenByPhone(context.Background(), "79000000000", "challenge", "1234")
	if err != nil {
		t.Fatalf("CreateAccessTokenByPhone: %v", err)
	}
	if token.Token != "access" {
		t.Errorf("Token = %q, want %q", token.Token, "access")
	}
}

// Refresh responses omit the refresh token and profile; both must be carried over.
func TestAuthRefreshCarriesOverMissingFields(t *testing.T) {
	t.Parallel()

	client, mux := setup(t)
	mux.HandleFunc("/v1/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"token":"fresh","tokenExpireIn":"2222-01-01T00:00:00Z"}`)
	})

	original := &AccessToken{
		Token:        "stale",
		RefreshToken: "refresh",
		Profile:      User{Inn: "770000000000"},
	}

	refreshed, _, err := client.Auth.Refresh(context.Background(), original)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if refreshed.Token != "fresh" {
		t.Errorf("Token = %q, want %q", refreshed.Token, "fresh")
	}
	if refreshed.RefreshToken != "refresh" {
		t.Errorf("RefreshToken = %q, want it carried over", refreshed.RefreshToken)
	}
	if refreshed.Profile.Inn != "770000000000" {
		t.Errorf("Profile.Inn = %q, want it carried over", refreshed.Profile.Inn)
	}
}

func TestAuthRefreshRejectsUnusableTokens(t *testing.T) {
	t.Parallel()

	client, mux := setup(t)
	mux.HandleFunc("/", func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("no request should have been sent, got %s %s", r.Method, r.URL)
	})

	// A missing refresh token is reported as such, not as a generic 401.
	_, _, err := client.Auth.Refresh(context.Background(), nil)
	assertLocalError(t, err)
	if !errors.Is(err, errNoAccessToken) {
		t.Errorf("error = %v, want errNoAccessToken", err)
	}

	_, _, err = client.Auth.Refresh(context.Background(), &AccessToken{Token: "t"})
	assertLocalError(t, err)
	if !errors.Is(err, errNoAccessToken) {
		t.Errorf("error = %v, want errNoAccessToken", err)
	}

	expired := &AccessToken{
		RefreshToken:          "refresh",
		RefreshTokenExpiresIn: NewTime(time.Now().Add(-time.Hour)),
	}
	_, _, err = client.Auth.Refresh(context.Background(), expired)
	assertLocalError(t, err)
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("error = %v, want ErrUnauthorized", err)
	}
}
