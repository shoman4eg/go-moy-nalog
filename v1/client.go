package moynalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Client Define API client
type Client struct {
	AccessToken string
	BaseURL     string
	HTTPClient  *http.Client
	Debug       bool
	Logger      logger
	do          doFunc
}
type doFunc func(req *http.Request) (*http.Response, error)
type logger func(format string, v ...interface{})

const (
	baseAPIMainURL string = "https://lknpd.nalog.ru/api/v1"
)

var defaultHeaders = map[string]string{
	"User-Agent":      UserAgent,
	"Content-type":    "application/json",
	"Accept":          "application/json, text/plain, */*",
	"Accept-language": "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7",
}

var authHeaders = map[string]string{
	"Referrer":        "https://lknpd.nalog.ru/",
	"Referrer-Policy": "strict-origin-when-cross-origin",
}

// NewClient Create new http client
func NewClient() *Client {
	client := &Client{
		BaseURL:    baseAPIMainURL,
		HTTPClient: http.DefaultClient,
	}

	return client
}

// WithLogger add logger
func (c *Client) WithLogger(l logger) *Client {
	c.Logger = l
	return c
}

// debug Put data in log
func (c *Client) debug(format string, v ...interface{}) {
	if !c.Debug || c.Logger == nil {
		return
	}

	c.Logger(format, v...)
}

func (c *Client) parseRequest(r *request, opts ...RequestOption) (err error) {
	for _, opt := range opts {
		opt(r)
	}
	err = r.validate()
	if err != nil {
		return err
	}
	fullURL := fmt.Sprintf("%s%s", c.BaseURL, r.endpoint)

	queryString := r.query.Encode()
	body := &bytes.Buffer{}
	header := http.Header{}
	if r.header != nil {
		header = r.header.Clone()
	}

	for headerName, headerValue := range defaultHeaders {
		header.Set(headerName, headerValue)
	}

	if r.json != nil {
		jsonString, _ := json.Marshal(r.json)
		c.debug("json: %v", r.json)

		body = bytes.NewBuffer(jsonString)
	}

	if c.AccessToken != "" {
		header.Set("Authorization", fmt.Sprintf("%s %s", "Bearer", c.AccessToken))
	} else {
		for headerName, headerValue := range authHeaders {
			header.Set(headerName, headerValue)
		}
	}

	if queryString != "" {
		fullURL = fmt.Sprintf("%s?%s", fullURL, queryString)
	}

	r.fullURL = fullURL
	r.header = header
	r.body = body
	return nil
}

func (c *Client) callAPI(ctx context.Context, r *request, opts ...RequestOption) (data []byte, err error) {
	err = c.parseRequest(r, opts...)
	if err != nil {
		return []byte{}, err
	}
	req, err := http.NewRequest(r.method, r.fullURL, r.body)
	if err != nil {
		return []byte{}, err
	}
	req = req.WithContext(ctx)
	req.Header = r.header
	c.debug("request: %#v", req)
	f := c.do
	if f == nil {
		f = c.HTTPClient.Do
	}
	res, err := f(req)
	if err != nil {
		return []byte{}, err
	}
	data, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, err
	}
	defer func() {
		cerr := res.Body.Close()
		if err == nil && cerr != nil {
			err = cerr
		}
	}()
	c.debug("response: %#v", res)
	c.debug("response body: %#v", string(data))
	c.debug("response status code: %#v", res.StatusCode)

	if res.StatusCode >= http.StatusBadRequest {
		apiErr := &APIError{
			Code:    int64(res.StatusCode),
			Message: string(data),
		}
		return nil, apiErr
	}

	return data, nil
}

func (c *Client) CreateNewAccessToken(username, password string) {

}

func (c *Client) Authenticate(accessToken string) {
	c.AccessToken = accessToken
}

// NewIncomeCreateService Init Income create Service
func (c *Client) NewIncomeCreateService() *IncomeCreateService {
	return &IncomeCreateService{c: c}
}
