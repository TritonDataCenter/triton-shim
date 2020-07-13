//
// Copyright 2020 Joyent, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//

package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrMissingURL The required API ServiceURL is not provided
	ErrMissingURL = errors.New("missing API URL")
	// ErrInvalidServiceURL The required API ServiceURL is not valid
	ErrInvalidServiceURL = errors.New("invalid format of API URL")
)

// Error represents an error code and message along with
// the status code of the HTTP request which resulted in the error
// message.
type Error struct {
	StatusCode int
	Code       string `json:"code"`
	Message    string `json:"message"`
	// Errors represents concrete errors for one of the multiple
	// parameters provided for the HTTP request, probably due to validation
	Errors []struct {
		Field   string `json:"field"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// Error implements interface Error on the APIError type.
func (e Error) Error() string {
	return strings.Trim(fmt.Sprintf("%+q", e.Code), `"`) + ": " + strings.Trim(fmt.Sprintf("%+q", e.Message), `"`)
}

// Is implements https://pkg.go.dev/errors?tab=doc#Is for APIError
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return (e.Code == t.Code)
}

// Client represents a connection to one of the Triton Internal APIs.
type Client struct {
	HTTPClient    *http.Client
	RequestHeader *http.Header
	URL           url.URL
	RequestID     string
}

// New creates a new client object for the provided ServiceURL
func New(ServiceURL string) (*Client, error) {
	if ServiceURL == "" {
		return nil, ErrMissingURL
	}

	APIURL, err := url.Parse(ServiceURL)
	if err != nil {
		return nil, ErrInvalidServiceURL
	}

	newClient := &Client{
		HTTPClient: &http.Client{
			Transport:     httpTransport(true),
			CheckRedirect: doNotFollowRedirects,
		},
		URL:       *APIURL,
		RequestID: "",
	}

	return newClient, nil

}

// InsecureSkipTLSVerify turns off TLS verification for the client connection. This
// allows connection to an endpoint with a certificate which was signed by a non-
// trusted CA, such as self-signed certificates. This can be useful when connecting
// to temporary Triton installations such as Triton Cloud-On-A-Laptop.
func (c *Client) InsecureSkipTLSVerify() {
	if c.HTTPClient == nil {
		return
	}

	c.HTTPClient.Transport = httpTransport(true)
}

// httpTransport is responsible for setting up our HTTP client's transport
// settings
func httpTransport(insecureSkipTLSVerify bool) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConns:        10,
		IdleConnTimeout:     15 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipTLSVerify,
		},
	}
}

func doNotFollowRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// DecodeError decodes a backend Triton error into a more usable Go error type
func (c *Client) DecodeError(resp *http.Response, requestMethod string, consumeBody bool) error {
	err := &Error{
		StatusCode: resp.StatusCode,
	}

	if requestMethod != http.MethodHead && resp.Body != nil && consumeBody {
		errorDecoder := json.NewDecoder(resp.Body)
		if err := errorDecoder.Decode(err); err != nil {
			return fmt.Errorf("unable to decode error response: %w", err)
		}
	}

	if err.Message == "" {
		err.Message = fmt.Sprintf("HTTP response returned status code %d", err.StatusCode)
	}

	return err
}

// overrideHeader overrides the header of the passed in HTTP request
func (c *Client) overrideHeader(req *http.Request) {
	if c.RequestHeader != nil {
		for k := range *c.RequestHeader {
			req.Header.Set(k, c.RequestHeader.Get(k))
		}
	}
}

// resetHeader will reset the struct field that stores custom header
// information
func (c *Client) resetHeader() {
	c.RequestHeader = nil
}

func (c *Client) getRequestID() string {
	return c.RequestID
}

func (c *Client) setRequestID(requestID string) {
	c.RequestID = requestID
}

// RequestInput take by ExecuteRequestURIParams
type RequestInput struct {
	Method  string
	Path    string
	Query   *url.Values
	Headers *http.Header
	Body    interface{}

	// If the response has the HTTP status code 410 (i.e., "Gone"),
	// should we preserve the contents of the body for the caller?
	PreserveGone bool
}

// ExecuteRequestURIParams performs an http.NewRequest against using the values provided by RequestInput
// If the returned error is nil, a non-nill Response.Body which the user is expected to close will be returned.
func (c *Client) ExecuteRequestURIParams(ctx context.Context, inputs RequestInput) (io.ReadCloser, error) {
	defer c.resetHeader()

	method := inputs.Method
	path := inputs.Path
	body := inputs.Body
	query := inputs.Query

	var requestBody io.Reader
	if body != nil {
		marshaled, err := json.MarshalIndent(body, "", "    ")
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(marshaled)
	}

	endpoint := c.URL
	endpoint.Path = path
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}

	req, err := http.NewRequest(method, endpoint.String(), requestBody)
	if err != nil {
		return nil, fmt.Errorf("Unable to construct HTTP Request: %w", err)
	}

	dateHeader := time.Now().UTC().Format(time.RFC1123)
	req.Header.Set("date", dateHeader)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Version", "*")
	req.Header.Set("User-Agent", "triton-shim")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.RequestID != "" {
		req.Header.Set("request-id", c.RequestID)
	}

	c.overrideHeader(req)

	resp, err := c.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("unable to execute HTTP request: %w", err)
	}

	reqID := resp.Header.Get("request-id")
	if reqID != "" {
		c.setRequestID(reqID)
	}
	// We will only return a response from the API it is in the HTTP StatusCode
	// 2xx range
	// StatusMultipleChoices is StatusCode 300
	if resp.StatusCode >= http.StatusOK &&
		resp.StatusCode < http.StatusMultipleChoices {
		return resp.Body, nil
	}

	return nil, c.DecodeError(resp, req.Method, true)
}
