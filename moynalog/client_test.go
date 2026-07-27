package moynalog

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pkg/errors"
)

func TestNewClientDefaults(t *testing.T) {
	t.Parallel()

	client := NewClient()

	if got, want := client.BaseURL.String(), defaultEndpoint+"/"+defaultVersion+"/"; got != want {
		t.Errorf("BaseURL = %q, want %q", got, want)
	}
	if client.UserAgent != defaultUserAgent {
		t.Errorf("UserAgent = %q, want the default browser agent", client.UserAgent)
	}
	if client.Token() != nil {
		t.Error("a fresh client must not be authenticated")
	}
	if client.DeviceID() == "" {
		t.Error("device id must be derived by NewClient")
	}
}

func TestNewClientOptions(t *testing.T) {
	t.Parallel()

	httpClient := new(http.Client)
	client := NewClient(
		WithHTTPClient(httpClient),
		WithEndpoint("https://example.test/api"),
		WithVersion("v9"),
		WithUserAgent("test-agent"),
		WithDeviceID("fixed-device"),
	)

	if got, want := client.BaseURL.String(), "https://example.test/api/v9/"; got != want {
		t.Errorf("BaseURL = %q, want %q", got, want)
	}
	if client.UserAgent != "test-agent" {
		t.Errorf("UserAgent = %q, want %q", client.UserAgent, "test-agent")
	}
	if client.DeviceID() != "fixed-device" {
		t.Errorf("DeviceID = %q, want %q", client.DeviceID(), "fixed-device")
	}
	if client.Client() == httpClient {
		t.Error("Client must hand out a copy, not the original http.Client")
	}
}

// The device id must be stable for the lifetime of the client and survive
// deriving an authenticated client from it.
func TestDeviceIDIsStable(t *testing.T) {
	t.Parallel()

	client := NewClient(WithDeviceIDGenerator(NewRandomDeviceIDGenerator()))

	first := client.DeviceID()
	if first == "" {
		t.Fatal("device id must not be empty")
	}
	if second := client.DeviceID(); second != first {
		t.Errorf("device id changed between calls: %q then %q", first, second)
	}

	authed := client.WithToken(&AccessToken{Token: "t"})
	if got := authed.DeviceID(); got != first {
		t.Errorf("WithToken changed the device id: %q, want %q", got, first)
	}
}

func TestNewRequest(t *testing.T) {
	t.Parallel()

	client := NewClient(WithEndpoint("https://example.test/api"))

	req, err := client.NewRequest(http.MethodPost, "income", map[string]string{"a": "b"})
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got, want := req.URL.String(), "https://example.test/api/v1/income"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
	testHeader(t, req, "Content-Type", mediaTypeJSON)
	testHeader(t, req, "Accept", "application/json, text/plain, */*")
	if req.Header.Get("User-Agent") == "" {
		t.Error("User-Agent must be set")
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if got, want := strings.TrimSpace(string(body)), `{"a":"b"}`; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

// A leading slash must not escape the versioned base path.
func TestNewRequestLeadingSlash(t *testing.T) {
	t.Parallel()

	client := NewClient(WithEndpoint("https://example.test/api"))

	req, err := client.NewRequest(http.MethodGet, "/taxes", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if got, want := req.URL.String(), "https://example.test/api/v1/taxes"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
}

func TestNewRequestNoBody(t *testing.T) {
	t.Parallel()

	client := NewClient()

	req, err := client.NewRequest(http.MethodGet, "user", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if req.Body != nil {
		t.Error("a nil body must produce a request without a body")
	}
	if got := req.Header.Get("Content-Type"); got != "" {
		t.Errorf("Content-Type = %q, want it unset", got)
	}
}

func TestDoDecodesJSON(t *testing.T) {
	t.Parallel()

	client, mux := setup(t)
	mux.HandleFunc("/v1/thing", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		writeJSON(t, w, http.StatusOK, `{"name":"value"}`)
	})

	req, err := client.NewRequest(http.MethodGet, "thing", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	got := struct {
		Name string `json:"name"`
	}{}
	if _, err := client.Do(context.Background(), req, &got); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got.Name != "value" {
		t.Errorf("Name = %q, want %q", got.Name, "value")
	}
}

func TestDoEmptyBodyIsNotAnError(t *testing.T) {
	t.Parallel()

	client, mux := setup(t)
	mux.HandleFunc("/v1/thing", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req, err := client.NewRequest(http.MethodGet, "thing", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	got := struct{}{}
	if _, err := client.Do(context.Background(), req, &got); err != nil {
		t.Fatalf("Do with empty body: %v", err)
	}
}

func TestDoNilContext(t *testing.T) {
	t.Parallel()

	client := NewClient()
	req, err := client.NewRequest(http.MethodGet, "user", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	//nolint:staticcheck // Passing a nil context is exactly what is under test.
	if _, err := client.Do(nil, req, nil); !errors.Is(err, errNonNilContext) {
		t.Errorf("error = %v, want errNonNilContext", err)
	}
}

func TestDoSendsBearerToken(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/v1/thing", func(w http.ResponseWriter, r *http.Request) {
		testHeader(t, r, "Authorization", "Bearer access-token")
		writeJSON(t, w, http.StatusOK, `{}`)
	})

	req, err := client.NewRequest(http.MethodGet, "thing", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := client.Do(context.Background(), req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
}

// A 401 must trigger a refresh and a replay of the original request, body included.
func TestDoRefreshesTokenOn401(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)

	var refreshes, attempts int32
	mux.HandleFunc("/v1/auth/token", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&refreshes, 1)
		testMethod(t, r, http.MethodPost)
		if got := testBody(t, r)["refreshToken"]; got != "refresh-token" {
			t.Errorf("refreshToken = %v, want %q", got, "refresh-token")
		}
		writeJSON(t, w, http.StatusOK, `{"token":"new-token","refreshToken":"new-refresh"}`)
	})
	mux.HandleFunc("/v1/thing", func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)

		body := testBody(t, r)
		if got := body["payload"]; got != "kept" {
			t.Errorf("replayed body payload = %v, want %q", got, "kept")
		}

		if attempt == 1 {
			testHeader(t, r, "Authorization", "Bearer access-token")
			writeJSON(t, w, http.StatusUnauthorized, `{"message":"token expired"}`)

			return
		}

		testHeader(t, r, "Authorization", "Bearer new-token")
		writeJSON(t, w, http.StatusOK, `{"ok":true}`)
	})

	req, err := client.NewRequest(http.MethodPost, "thing", map[string]string{"payload": "kept"})
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	got := struct {
		OK bool `json:"ok"`
	}{}
	if _, err := client.Do(context.Background(), req, &got); err != nil {
		t.Fatalf("Do: %v", err)
	}

	if !got.OK {
		t.Error("the replayed request must return the successful payload")
	}
	if refreshes != 1 {
		t.Errorf("refresh calls = %d, want 1", refreshes)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
	if token := client.Token(); token == nil || token.Token != "new-token" {
		t.Errorf("client token = %+v, want it replaced with the refreshed one", token)
	}
}

// A token that keeps coming back 401 must give up rather than loop.
func TestDoGivesUpAfterMaxRetries(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)

	var attempts int32
	mux.HandleFunc("/v1/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"token":"another","refreshToken":"refresh-token"}`)
	})
	mux.HandleFunc("/v1/thing", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&attempts, 1)
		writeJSON(t, w, http.StatusUnauthorized, `{"message":"nope"}`)
	})

	req, err := client.NewRequest(http.MethodGet, "thing", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	_, err = client.Do(context.Background(), req, nil)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
	if want := int32(maxAuthRetries + 1); attempts != want {
		t.Errorf("attempts = %d, want %d", attempts, want)
	}
}

// Authentication requests must never be replayed through the refresh flow.
func TestAuthRequestsAreNotRetried(t *testing.T) {
	t.Parallel()

	client, mux := setupAuthed(t)

	var attempts int32
	mux.HandleFunc("/v1/auth/lkfl", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization = %q, want it unset on auth requests", got)
		}
		writeJSON(t, w, http.StatusUnauthorized, `{"message":"bad credentials"}`)
	})

	_, _, err := client.Auth.CreateAccessToken(context.Background(), "inn", "wrong")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestAddOptions(t *testing.T) {
	t.Parallel()

	opts := IncomeListOptions{Limit: 5, Offset: 10, SortBy: SortByTotalAmountAsc}

	got, err := addOptions("incomes", opts)
	if err != nil {
		t.Fatalf("addOptions: %v", err)
	}
	if want := "incomes?limit=5&offset=10&sortBy=total_amount%3Aasc"; got != want {
		t.Errorf("addOptions = %q, want %q", got, want)
	}
}

func TestAddOptionsNilPointer(t *testing.T) {
	t.Parallel()

	var opts *IncomeListOptions

	got, err := addOptions("incomes", opts)
	if err != nil {
		t.Fatalf("addOptions: %v", err)
	}
	if got != "incomes" {
		t.Errorf("addOptions = %q, want it unchanged", got)
	}
}
