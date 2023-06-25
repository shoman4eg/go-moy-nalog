package moynalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultBaseURL = "https://lknpd.nalog.ru/api"
	baseAPIVersion = "v1"

	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.192 Safari/537.36"
)

var (
	errNonNilContext      = errors.New("context must be non-nil")
	errAccessTokenIsEmpty = errors.New("access token cannot be null")
)

type Client struct {
	clientMu  sync.Mutex
	client    *http.Client
	BaseURL   *url.URL
	UserAgent string

	AccessToken *AccessToken

	common service

	Auth    *AuthSerive
	Users   *UsersService
	Income  *IncomeService
	Receipt *ReceiptService
}

type service struct {
	client *Client
}

func (c *Client) Client() *http.Client {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	clientCopy := *c.client
	return &clientCopy
}

type LimitOptions struct {
	Limit  int    `url:"limit,omitempty"`
	Offset int    `url:"offset,omitempty"`
	SortBy string `url:"sortBy,omitempty"`
}

type DateRangeOptions struct {
	From time.Time `url:"from,omitempty"`
	To   time.Time `url:"to,omitempty"`
}

func NewClient(httpClient *http.Client) *Client {
	return NewClientWithVersion(httpClient, baseAPIVersion)
}

func NewAuthClient(token *AccessToken) *Client {
	bc := BearerTokenTransport{Token: token}
	return NewClient(bc.Client())
}

func NewClientWithVersion(httpClient *http.Client, version string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	baseURL, _ := url.Parse(fmt.Sprintf("%s/%s/", defaultBaseURL, version))

	c := &Client{client: httpClient, BaseURL: baseURL, UserAgent: defaultUserAgent}

	c.common.client = c

	c.Auth = (*AuthSerive)(&c.common)
	c.Users = (*UsersService)(&c.common)
	c.Income = (*IncomeService)(&c.common)
	c.Receipt = (*ReceiptService)(&c.common)

	return c
}

func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	if !strings.HasSuffix(c.BaseURL.Path, "/") {
		return nil, errors.Errorf("BaseURL must have a trailing slash, but %q does not", c.BaseURL)
	}

	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	return req, nil
}

type Response struct {
	*http.Response
}

func newResponse(r *http.Response) *Response {
	response := &Response{Response: r}
	return response
}

type Error struct {
	Code           string `json:"code"`
	Message        string `json:"message"`
	AdditionalInfo any    `json:"additionalInfo"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%v error caused with %v message and %v additionalInfo", e.Code, e.Message, e.AdditionalInfo)
}

func (e *Error) UnmarshalJSON(data []byte) error {
	type aliasError Error // avoid infinite recursion by using type alias.
	if err := json.Unmarshal(data, (*aliasError)(e)); err != nil {
		return json.Unmarshal(data, &e.Message) // data can be json string.
	}
	return nil
}

type ErrorResponse struct {
	Response *http.Response // HTTP response that caused this error
	Message  string         `json:"message"` // error message
	Code     string         `json:"code"`    // error message
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v %+v",
		r.Response.Request.Method, r.Response.Request.URL,
		r.Response.StatusCode, r.Message, r.Code)
}

// BareDo sends an API request and lets you handle the api response. If an error
// or API Error occurs, the error will contain more information. Otherwise, you
// are supposed to read and close the response's Body. If rate limit is exceeded
// and reset time is in the future, BareDo returns *RateLimitError immediately
// without making a network API call.
//
// The provided ctx must be non-nil, if it is nil an error is returned. If it is
// canceled or times out, ctx.Err() will be returned.
func (c *Client) BareDo(ctx context.Context, req *http.Request) (*Response, error) {
	if ctx == nil {
		return nil, errNonNilContext
	}

	req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// If the error type is *url.Error, sanitize its URL before returning.
		if e, ok := err.(*url.Error); ok {
			if uri, err := url.Parse(e.URL); err == nil {
				e.URL = uri.String()
				return nil, e
			}
		}

		return nil, err
	}

	response := newResponse(resp)

	err = CheckResponse(resp)
	if err != nil {
		defer resp.Body.Close()
	}
	return response, err
}

// Do sends an API request and returns the API response. The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred. If v implements the io.Writer interface,
// the raw response body will be written to v, without attempting to first
// decode it. If v is nil, and no error hapens, the response is returned as is.
// If rate limit is exceeded and reset time is in the future, Do returns
// *RateLimitError immediately without making a network API call.
//
// The provided ctx must be non-nil, if it is nil an error is returned. If it
// is canceled or times out, ctx.Err() will be returned.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	resp, err := c.BareDo(ctx, req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	switch v := v.(type) {
	case nil:
	case io.Writer:
		_, err = io.Copy(v, resp.Body)
	default:
		decErr := json.NewDecoder(resp.Body).Decode(v)
		if decErr == io.EOF {
			decErr = nil // ignore EOF errors caused by empty response body
		}
		if decErr != nil {
			err = decErr
		}
	}
	return resp, err
}

func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}
	data, err := io.ReadAll(r.Body)
	if err != nil && data != nil {
		if err0 := json.Unmarshal(data, errorResponse); err0 != nil {
			return err0
		}
	}

	return errorResponse
}

type BearerTokenTransport struct {
	Token *AccessToken
	// Transport is the underlying HTTP transport to use when making requests.
	// It will default to http.DefaultTransport if nil.
	Transport http.RoundTripper
}

// RoundTrip implements the RoundTripper interface.
func (t *BearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.transport().RoundTrip(setBearerTokenHeader(req, t.Token))
}

func setBearerTokenHeader(req *http.Request, token *AccessToken) *http.Request {
	// To set extra headers, we must make a copy of the Request so
	// that we don't modify the Request we were given. This is required by the
	// specification of http.RoundTripper.
	//
	// Since we are going to modify only req.Header here, we only need a deep copy
	// of req.Header.
	convertedRequest := new(http.Request)
	*convertedRequest = *req
	convertedRequest.Header = make(http.Header, len(req.Header))

	for k, s := range req.Header {
		convertedRequest.Header[k] = append([]string(nil), s...)
	}
	convertedRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token.Token))
	return convertedRequest
}

// Client returns an *http.Client that makes requests that are authenticated
// using HTTP Basic Authentication.
func (t *BearerTokenTransport) Client() *http.Client {
	return &http.Client{Transport: t}
}

func (t *BearerTokenTransport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}
