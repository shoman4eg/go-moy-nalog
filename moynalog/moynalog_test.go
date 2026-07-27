package moynalog

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
)

// setup spins up a test server and returns a client configured to talk to it.
func setup(t *testing.T) (*Client, *http.ServeMux) {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := NewClient(
		WithEndpoint(server.URL),
		WithDeviceID("testdeviceid"),
	)

	return client, mux
}

// setupAuthed returns a client already authenticated with a non-expiring token.
func setupAuthed(t *testing.T) (*Client, *http.ServeMux) {
	t.Helper()

	client, mux := setup(t)

	return client.WithToken(&AccessToken{
		Token:        "access-token",
		RefreshToken: "refresh-token",
		Profile:      User{Inn: "770000000000"},
	}), mux
}

func testMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()

	if r.Method != want {
		t.Errorf("request method = %q, want %q", r.Method, want)
	}
}

func testHeader(t *testing.T, r *http.Request, name, want string) {
	t.Helper()

	if got := r.Header.Get(name); got != want {
		t.Errorf("header %s = %q, want %q", name, got, want)
	}
}

func testQuery(t *testing.T, r *http.Request, want map[string]string) {
	t.Helper()

	got := r.URL.Query()
	for key, wantValue := range want {
		if gotValue := got.Get(key); gotValue != wantValue {
			t.Errorf("query %s = %q, want %q", key, gotValue, wantValue)
		}
	}
}

// testBody decodes the request body into a generic map for shape assertions.
func testBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}

	body := map[string]any{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode request body %q: %v", raw, err)
	}

	return body
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload string) {
	t.Helper()

	w.Header().Set("Content-Type", mediaTypeJSON)
	w.WriteHeader(status)
	if _, err := io.WriteString(w, payload); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

// setupNoRequest returns an authenticated client whose test server fails the
// test if it is ever reached. Use it for input that must be rejected locally,
// before any request goes out.
func setupNoRequest(t *testing.T) *Client {
	t.Helper()

	client, mux := setupAuthed(t)
	mux.HandleFunc("/", func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("no request should have been sent, got %s %s", r.Method, r.URL)
	})

	return client
}

// assertLocalError fails unless err is a client-side validation error. An
// *ErrorResponse means the input reached the API instead of being rejected.
func assertLocalError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Error("want a validation error, got nil")

		return
	}

	var errResp *ErrorResponse
	if errors.As(err, &errResp) {
		t.Errorf("want a local validation error, got an API error: %v", err)
	}
}
