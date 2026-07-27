package moynalog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// Sentinel errors describing the class of an API failure. Match them with
// errors.Is, and reach for the details with errors.As and *ErrorResponse.
var (
	// ErrValidation is returned for HTTP 400 responses.
	ErrValidation = errors.New("moynalog: validation error")
	// ErrUnauthorized is returned for HTTP 401 responses.
	ErrUnauthorized = errors.New("moynalog: unauthorized")
	// ErrForbidden is returned for HTTP 403 responses.
	ErrForbidden = errors.New("moynalog: forbidden")
	// ErrNotFound is returned for HTTP 404 responses.
	ErrNotFound = errors.New("moynalog: not found")
	// ErrClient is returned for HTTP 406 responses, which the API uses to
	// reject unacceptable Accept headers.
	ErrClient = errors.New("moynalog: client error")
	// ErrPhone is returned for HTTP 422 responses raised by the SMS
	// authentication endpoints.
	ErrPhone = errors.New("moynalog: phone error")
	// ErrServer is returned for HTTP 500 responses.
	ErrServer = errors.New("moynalog: server error")
	// ErrUnknown is returned for any other unsuccessful status code.
	ErrUnknown = errors.New("moynalog: unknown error")

	// ErrNotImplemented marks endpoints the upstream API does not expose.
	ErrNotImplemented = errors.New("moynalog: not implemented by the upstream API")
)

// ErrorResponse reports an error caused by an API request. It unwraps to one of
// the sentinel errors declared above, so both of these work:
//
//	errors.Is(err, moynalog.ErrNotFound)
//
//	var errResp *moynalog.ErrorResponse
//	errors.As(err, &errResp)
type ErrorResponse struct {
	// Response is the HTTP response that caused this error.
	Response *http.Response `json:"-"`
	// Code is the machine readable error code reported by the API, if any.
	Code string `json:"code"`
	// Message is the human readable error message reported by the API, if any.
	Message string `json:"message"`
	// AdditionalInfo carries whatever extra payload the API attached.
	AdditionalInfo any `json:"additionalInfo"`
	// Body is the raw response body.
	Body []byte `json:"-"`

	kind error
}

// Error implements the error interface.
func (r *ErrorResponse) Error() string {
	message := r.Message
	if message == "" {
		message = http.StatusText(r.StatusCode())
	}

	if r.Response == nil || r.Response.Request == nil {
		return fmt.Sprintf("%d %v %v", r.StatusCode(), message, r.Code)
	}

	return fmt.Sprintf(
		"%v %v: %d %v %v",
		r.Response.Request.Method,
		r.Response.Request.URL,
		r.StatusCode(),
		message,
		r.Code,
	)
}

// StatusCode returns the HTTP status code of the failed response.
func (r *ErrorResponse) StatusCode() int {
	if r.Response == nil {
		return 0
	}

	return r.Response.StatusCode
}

// Unwrap returns the sentinel error describing the class of this failure.
func (r *ErrorResponse) Unwrap() error {
	return r.kind
}

// errorKind maps a status code onto one of the sentinel errors. The mapping
// mirrors the domain exceptions raised by the reference PHP client.
func errorKind(statusCode int) error {
	switch statusCode {
	case http.StatusBadRequest:
		return ErrValidation
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusNotAcceptable:
		return ErrClient
	case http.StatusUnprocessableEntity:
		return ErrPhone
	case http.StatusInternalServerError:
		return ErrServer
	default:
		return ErrUnknown
	}
}

// CheckResponse returns an *ErrorResponse when r reports failure, and nil
// otherwise. It consumes r.Body on the error path only.
func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; c >= 200 && c <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{
		Response: r,
		kind:     errorKind(r.StatusCode),
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return errorResponse
	}
	errorResponse.Body = body

	// The API is inconsistent: most endpoints answer with an error object,
	// some with a bare JSON string. Decoding either is best effort.
	if err := json.Unmarshal(body, errorResponse); err == nil {
		return errorResponse
	}

	var message string
	if err := json.Unmarshal(body, &message); err == nil {
		errorResponse.Message = message
	}

	return errorResponse
}
