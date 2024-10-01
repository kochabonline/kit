package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/kochabonline/kit/errors"
)

const (
	ErrBadRequest = "http request failed"
)

const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH" // RFC 5789
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
)

type Client interface {
	Request(method, url string, body io.Reader, opts ...func(*RequestOption)) (any, error)
}

type Http struct {
	client *http.Client
}

type Option func(*Http)

// WithClient sets the HTTP client.
func WithClient(client *http.Client) func(*Http) {
	return func(h *Http) {
		h.client = client
	}
}

// New creates a new HTTP client.
func New(opts ...Option) *Http {
	h := &Http{
		client: &http.Client{},
	}
	for _, o := range opts {
		o(h)
	}
	return h
}

// RequestOption is the option for the HTTP request.
type RequestOption struct {
	header   map[string]string
	response any
}

// WithRequestHeader sets the request header.
func WithRequestHeader(header map[string]string) func(*RequestOption) {
	return func(opt *RequestOption) {
		opt.header = header
	}
}

// WithRequestResponse sets the response object to unmarshal the response body.
// The response object must be a pointer.
func WithRequestResponse(response any) func(*RequestOption) {
	return func(opt *RequestOption) {
		opt.response = response
	}
}

// Request sends an HTTP request.
func (h *Http) Request(method, url string, body io.Reader, opts ...func(*RequestOption)) error {
	opt := &RequestOption{}

	for _, o := range opts {
		o(opt)
	}

	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	for k, v := range opt.header {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.BadRequest(ErrBadRequest, string(respByte))
	}

	if opt.response != nil {
		err = json.Unmarshal(respByte, &opt.response)
		if err != nil {
			return err
		}
	}

	return nil
}
