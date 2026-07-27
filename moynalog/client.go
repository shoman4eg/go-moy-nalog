// Package moynalog is an unofficial client for the lknpd.nalog.ru ("Мой налог")
// API used by self-employed taxpayers in Russia.
package moynalog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"github.com/google/go-querystring/query"
	"github.com/pkg/errors"
)

const (
	defaultEndpoint = "https://lknpd.nalog.ru/api"
	defaultVersion  = "v1"
	// phoneAuthVersion is the API version serving the SMS challenge endpoints.
	phoneAuthVersion = "v2"

	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.192 Safari/537.36"

	mediaTypeJSON = "application/json"

	// maxAuthRetries limits how many times a single request is replayed after a
	// 401 response triggered an access token refresh.
	maxAuthRetries = 2
)

var (
	errNonNilContext = errors.New("moynalog: context must be non-nil")
	errNoAccessToken = errors.New("moynalog: client is not authenticated")
)

// skipAuthContextKey marks requests that must not carry an Authorization header
// and must not be replayed through the token refresh flow.
type skipAuthContextKey struct{}

func withoutAuth(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipAuthContextKey{}, true)
}

func authSkipped(ctx context.Context) bool {
	skip, ok := ctx.Value(skipAuthContextKey{}).(bool)

	return ok && skip
}

// Client manages communication with the lknpd.nalog.ru API. It is safe for
// concurrent use by multiple goroutines.
type Client struct {
	clientMu sync.Mutex
	client   *http.Client

	// BaseURL is the versioned API root. It must always end in a trailing slash.
	BaseURL *url.URL
	// UserAgent is sent with every request. The default impersonates a desktop
	// browser, which is what the upstream API expects.
	UserAgent string

	version string

	// deviceID is derived once in NewClient and never changes afterwards: the
	// API ties registered receipts to the device that created them.
	deviceIDGenerator DeviceIDGenerator
	deviceID          string
	deviceIDErr       error

	tokenMu sync.RWMutex
	token   *AccessToken

	common service // Reuse a single struct instead of allocating one per service.

	// Services used for talking to the different parts of the API.
	Auth        *AuthService
	Users       *UsersService
	Income      *IncomeService
	Invoice     *InvoiceService
	Receipt     *ReceiptService
	PaymentType *PaymentTypeService
	Tax         *TaxService
	Taxpayer    *TaxpayerService
}

type service struct {
	client *Client
}

// Option customises a Client at construction time.
type Option func(*Client)

// WithHTTPClient makes the client send its requests through httpClient.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.client = httpClient
		}
	}
}

// WithEndpoint overrides the API root, which defaults to
// https://lknpd.nalog.ru/api. The version segment is appended to it, so pass
// the URL without one.
func WithEndpoint(endpoint string) Option {
	return func(c *Client) {
		c.BaseURL = mustBaseURL(endpoint, c.version)
	}
}

// WithVersion overrides the API version segment, which defaults to v1.
func WithVersion(version string) Option {
	return func(c *Client) {
		if version == "" {
			return
		}
		endpoint := strings.TrimSuffix(c.BaseURL.String(), "/"+c.version+"/")
		c.version = version
		c.BaseURL = mustBaseURL(endpoint, version)
	}
}

// WithUserAgent overrides the User-Agent header sent with every request.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		if userAgent != "" {
			c.UserAgent = userAgent
		}
	}
}

// WithDeviceID pins the device identifier reported to the API during
// authentication instead of deriving one from the host.
func WithDeviceID(deviceID string) Option {
	return func(c *Client) {
		c.deviceID = deviceID
	}
}

// WithDeviceIDGenerator replaces the strategy used to derive a device
// identifier. It is ignored when WithDeviceID is also supplied.
func WithDeviceIDGenerator(generator DeviceIDGenerator) Option {
	return func(c *Client) {
		if generator != nil {
			c.deviceIDGenerator = generator
		}
	}
}

func mustBaseURL(endpoint, version string) *url.URL {
	parsed, err := url.Parse(strings.TrimSuffix(endpoint, "/") + "/" + version + "/")
	if err != nil {
		return nil
	}

	return parsed
}

// NewClient returns a new lknpd.nalog.ru API client. The returned client is
// unauthenticated; obtain a token through the Auth service and derive an
// authenticated client from it with WithToken.
func NewClient(opts ...Option) *Client {
	c := &Client{
		client:            new(http.Client),
		BaseURL:           mustBaseURL(defaultEndpoint, defaultVersion),
		UserAgent:         defaultUserAgent,
		version:           defaultVersion,
		deviceIDGenerator: NewPlatformDeviceIDGenerator(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Derive the device identifier once, here, so it stays stable for every
	// request this client will ever make. A failure is remembered and reported
	// by the authentication calls, which are the only users of it.
	if c.deviceID == "" {
		c.deviceID, c.deviceIDErr = c.deviceIDGenerator.DeviceID()
	}

	c.initServices()

	return c
}

func (c *Client) initServices() {
	c.common.client = c

	c.Auth = (*AuthService)(&c.common)
	c.Users = (*UsersService)(&c.common)
	c.Income = (*IncomeService)(&c.common)
	c.Invoice = (*InvoiceService)(&c.common)
	c.Receipt = (*ReceiptService)(&c.common)
	c.PaymentType = (*PaymentTypeService)(&c.common)
	c.Tax = (*TaxService)(&c.common)
	c.Taxpayer = (*TaxpayerService)(&c.common)
}

// Client returns a copy of the underlying *http.Client.
func (c *Client) Client() *http.Client {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	clientCopy := *c.client

	return &clientCopy
}

// WithToken returns a copy of the client that authenticates its requests with
// token. Expired access tokens are refreshed automatically, so read the current
// token back with Token before persisting it.
func (c *Client) WithToken(token *AccessToken) *Client {
	authed := &Client{
		client:            c.client,
		BaseURL:           c.BaseURL,
		UserAgent:         c.UserAgent,
		version:           c.version,
		deviceIDGenerator: c.deviceIDGenerator,
		deviceID:          c.deviceID,
		deviceIDErr:       c.deviceIDErr,
		token:             token,
	}
	authed.initServices()

	return authed
}

// Token returns the access token currently held by the client, which may have
// been refreshed since it was passed to WithToken. It returns nil when the
// client is unauthenticated.
func (c *Client) Token() *AccessToken {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()

	return c.token
}

func (c *Client) setToken(token *AccessToken) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	c.token = token
}

// DeviceID returns the device identifier reported to the API during
// authentication. It is derived by NewClient and never changes.
func (c *Client) DeviceID() string {
	return c.deviceID
}

// NewRequest creates an API request. urlStr is resolved relative to BaseURL and
// should therefore never have a leading slash. A non-nil body is JSON encoded.
func (c *Client) NewRequest(method, urlStr string, body any) (*http.Request, error) {
	return c.newRequest(c.BaseURL, method, urlStr, body)
}

// newVersionedRequest builds a request against an API version other than the
// configured one. Only the SMS challenge endpoints need this.
func (c *Client) newVersionedRequest(method, version, urlStr string, body any) (*http.Request, error) {
	base := c.BaseURL
	if version != c.version {
		endpoint := strings.TrimSuffix(c.BaseURL.String(), "/"+c.version+"/")
		base = mustBaseURL(endpoint, version)
	}

	return c.newRequest(base, method, urlStr, body)
}

func (c *Client) newRequest(base *url.URL, method, urlStr string, body any) (*http.Request, error) {
	if base == nil || !strings.HasSuffix(base.Path, "/") {
		return nil, errors.Errorf("moynalog: BaseURL must have a trailing slash, but %q does not", base)
	}

	u, err := base.Parse(strings.TrimPrefix(urlStr, "/"))
	if err != nil {
		return nil, errors.Wrap(err, "moynalog: cannot resolve request URL")
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(body); err != nil {
			return nil, errors.Wrap(err, "moynalog: cannot encode request body")
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, errors.Wrap(err, "moynalog: cannot create request")
	}

	if body != nil {
		req.Header.Set("Content-Type", mediaTypeJSON)
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	return req, nil
}

// addOptions adds the parameters in opts as URL query parameters of s. opts
// must be a struct whose fields may contain "url" tags.
func addOptions(s string, opts any) (string, error) {
	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, errors.Wrap(err, "moynalog: cannot parse URL")
	}

	qs, err := query.Values(opts)
	if err != nil {
		return s, errors.Wrap(err, "moynalog: cannot encode query parameters")
	}
	u.RawQuery = qs.Encode()

	return u.String(), nil
}

// Response wraps *http.Response so response metadata can grow independently of
// the standard library type.
type Response struct {
	*http.Response
}

func newResponse(r *http.Response) *Response {
	return &Response{Response: r}
}

// Do sends an API request and decodes a successful JSON response into v. When v
// implements io.Writer the raw body is copied into it instead; a nil v discards
// the body.
//
// A 401 response triggers an access token refresh and a replay of the request,
// at most maxAuthRetries times.
func (c *Client) Do(ctx context.Context, req *http.Request, v any) (*Response, error) {
	resp, err := c.BareDo(ctx, req)
	if err != nil {
		return resp, err
	}
	//nolint:gosec // G104: the body has been consumed; a close failure here
	// is not actionable by the caller.
	defer func() { _ = resp.Body.Close() }()

	switch v := v.(type) {
	case nil:
	case io.Writer:
		if _, err := io.Copy(v, resp.Body); err != nil {
			return resp, errors.Wrap(err, "moynalog: cannot read response body")
		}
	default:
		// An empty body is not an error; several endpoints answer with one.
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil && !errors.Is(err, io.EOF) {
			return resp, errors.Wrap(err, "moynalog: cannot decode response body")
		}
	}

	return resp, nil
}

// BareDo sends an API request and leaves the response body open for the caller
// to read and close. The body is already drained and closed when a non-nil
// error is returned.
func (c *Client) BareDo(ctx context.Context, req *http.Request) (*Response, error) {
	if ctx == nil {
		return nil, errNonNilContext
	}

	skipAuth := authSkipped(ctx)

	for attempt := 0; ; attempt++ {
		token := c.Token()

		outReq, err := cloneRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		if !skipAuth && token != nil && outReq.Header.Get("Authorization") == "" {
			outReq.Header.Set("Authorization", "Bearer "+token.Token)
		}

		resp, err := c.send(ctx, outReq)
		if err != nil {
			return nil, err
		}

		apiErr := CheckResponse(resp)
		if apiErr == nil {
			return newResponse(resp), nil
		}
		//nolint:gosec // G104: CheckResponse already drained the body.
		_ = resp.Body.Close()

		retriable := !skipAuth &&
			resp.StatusCode == http.StatusUnauthorized &&
			attempt < maxAuthRetries &&
			token != nil &&
			token.RefreshToken != ""
		if !retriable {
			return newResponse(resp), apiErr
		}

		refreshed, _, err := c.Auth.Refresh(ctx, token)
		if err != nil {
			// Surface the original 401; the refresh failure is only context.
			return newResponse(resp), errors.Wrap(apiErr, err.Error())
		}
		c.setToken(refreshed)
	}
}

func (c *Client) send(ctx context.Context, req *http.Request) (*http.Response, error) {
	//nolint:gosec // G704: relaying caller-built requests to the configured
	// endpoint is the entire purpose of this package.
	resp, err := c.client.Do(req)
	if err != nil {
		// A cancelled context explains the failure better than the transport error.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Sanitize the URL of *url.Error before letting it escape.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			if sanitized, perr := url.Parse(urlErr.URL); perr == nil {
				urlErr.URL = sanitized.String()

				return nil, urlErr
			}
		}

		return nil, err
	}

	return resp, nil
}

// cloneRequest produces a replayable copy of req bound to ctx.
func cloneRequest(ctx context.Context, req *http.Request) (*http.Request, error) {
	clone := req.Clone(ctx)
	if req.GetBody == nil {
		return clone, nil
	}

	body, err := req.GetBody()
	if err != nil {
		return nil, errors.Wrap(err, "moynalog: cannot rewind request body")
	}
	clone.Body = body

	return clone, nil
}
