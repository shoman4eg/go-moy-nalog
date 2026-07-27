package moynalog

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func TestCheckResponseSuccess(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusNoContent} {
		resp := &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(""))}
		if err := CheckResponse(resp); err != nil {
			t.Errorf("CheckResponse(%d) = %v, want nil", status, err)
		}
	}
}

func TestCheckResponseStatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		want   error
	}{
		{"bad request", http.StatusBadRequest, ErrValidation},
		{"unauthorized", http.StatusUnauthorized, ErrUnauthorized},
		{"forbidden", http.StatusForbidden, ErrForbidden},
		{"not found", http.StatusNotFound, ErrNotFound},
		{"not acceptable", http.StatusNotAcceptable, ErrClient},
		{"unprocessable entity", http.StatusUnprocessableEntity, ErrPhone},
		{"internal server error", http.StatusInternalServerError, ErrServer},
		{"anything else", http.StatusTeapot, ErrUnknown},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &http.Response{
				StatusCode: tt.status,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}

			err := CheckResponse(resp)
			if !errors.Is(err, tt.want) {
				t.Errorf("CheckResponse(%d) = %v, want it to wrap %v", tt.status, err, tt.want)
			}
		})
	}
}

func TestCheckResponseDetails(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body: io.NopCloser(strings.NewReader(
			`{"code":"taxpayer.unregistered","message":"Не зарегистрирован","additionalInfo":{"x":1}}`,
		)),
	}

	err := CheckResponse(resp)

	var errResp *ErrorResponse
	if !errors.As(err, &errResp) {
		t.Fatalf("error = %v, want an *ErrorResponse", err)
	}
	if errResp.Code != "taxpayer.unregistered" {
		t.Errorf("Code = %q, want %q", errResp.Code, "taxpayer.unregistered")
	}
	if errResp.Message != "Не зарегистрирован" {
		t.Errorf("Message = %q, want %q", errResp.Message, "Не зарегистрирован")
	}
	if errResp.AdditionalInfo == nil {
		t.Error("AdditionalInfo must be preserved")
	}
	if errResp.StatusCode() != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", errResp.StatusCode(), http.StatusBadRequest)
	}
	if len(errResp.Body) == 0 {
		t.Error("the raw body must be preserved")
	}
}

// Some endpoints answer with a bare JSON string rather than an error object.
func TestCheckResponseBareStringBody(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(`"boom"`)),
	}

	err := CheckResponse(resp)

	var errResp *ErrorResponse
	if !errors.As(err, &errResp) {
		t.Fatalf("error = %v, want an *ErrorResponse", err)
	}
	if errResp.Message != "boom" {
		t.Errorf("Message = %q, want %q", errResp.Message, "boom")
	}
}

func TestErrorResponseMessageFallsBackToStatusText(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader(``)),
	}

	err := CheckResponse(resp)
	if !strings.Contains(err.Error(), http.StatusText(http.StatusNotFound)) {
		t.Errorf("Error() = %q, want it to mention the status text", err.Error())
	}
}

// A body that is not JSON at all must still produce a typed error.
func TestCheckResponseNonJSONBody(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(strings.NewReader(`not-json`)),
	}

	err := CheckResponse(resp)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}

	var errResp *ErrorResponse
	if !errors.As(err, &errResp) {
		t.Fatalf("error = %v, want an *ErrorResponse", err)
	}
	if errResp.Message != "" {
		t.Errorf("Message = %q, want it empty for an unparsable body", errResp.Message)
	}
	if string(errResp.Body) != "not-json" {
		t.Errorf("Body = %q, want the raw body preserved", errResp.Body)
	}
}

// The message the API puts in a 401 must reach the caller.
func TestCheckResponseUnauthorizedMessage(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(strings.NewReader(`{"message":"Неверный логин или пароль"}`)),
	}

	var errResp *ErrorResponse
	if !errors.As(CheckResponse(resp), &errResp) {
		t.Fatal("want an *ErrorResponse")
	}
	if errResp.Message != "Неверный логин или пароль" {
		t.Errorf("Message = %q, want the API message", errResp.Message)
	}
}
