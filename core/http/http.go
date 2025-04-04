package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"sync"

	"github.com/kochabonline/kit/errors"
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

type Clienter interface {
	Request(method, url string, body any, opts ...func(*RequestOption)) (*http.Response, error)
}

type Client struct {
	client         *http.Client
	requestOptPool sync.Pool
	bufferPool     sync.Pool
}

type Option func(*Client)

// WithClient sets the HTTP client.
func WithClient(client *http.Client) func(*Client) {
	return func(h *Client) {
		h.client = client
	}
}

// New creates a new HTTP.
func New(opts ...Option) *Client {
	h := &Client{
		client: &http.Client{},
		requestOptPool: sync.Pool{
			New: func() any {
				return &RequestOption{
					header: make(map[string]string),
				}
			},
		},
		bufferPool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 4096))
			},
		},
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

type RequestOption struct {
	ctx      context.Context
	header   map[string]string
	response any
}

// WithContext sets a custom context for this specific request.
func WithContext(ctx context.Context) func(*RequestOption) {
	return func(opt *RequestOption) {
		opt.ctx = ctx
	}
}

// WithHeader sets multiple headers.
func WithHeader(header map[string]string) func(*RequestOption) {
	return func(opt *RequestOption) {
		maps.Copy(opt.header, header)
	}
}

// WithResponse sets the response object.
func WithResponse(response any) func(*RequestOption) {
	return func(opt *RequestOption) {
		opt.response = response
	}
}

func (opt *RequestOption) reset() {
	opt.ctx = nil
	for k := range opt.header {
		delete(opt.header, k)
	}
	opt.response = nil
}

// Request sends an HTTP request and returns the response.
func (cli *Client) Request(method, url string, body any, opts ...func(*RequestOption)) (*http.Response, error) {
	// Get RequestOption object from the pool
	optInterface := cli.requestOptPool.Get()
	opt := optInterface.(*RequestOption)
	// Reset the RequestOption object to its initial state
	opt.reset()
	opt.header["Content-Type"] = "application/json"
	// Put RequestOption back to the pool when the function returns
	defer cli.requestOptPool.Put(opt)

	for _, o := range opts {
		o(opt)
	}

	var req *http.Request
	var err error

	switch v := body.(type) {
	case nil:
		req, err = http.NewRequest(method, url, nil)
	case io.Reader:
		req, err = http.NewRequest(method, url, v)
	default:
		// Get buffer from the pool
		buf := cli.bufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		defer cli.bufferPool.Put(buf)

		// Write the body to the buffer
		enc := json.NewEncoder(buf)
		if jsonErr := enc.Encode(v); jsonErr != nil {
			return nil, errors.BadRequest("encode request body: %v", jsonErr)
		}

		req, err = http.NewRequest(method, url, buf)
	}

	if err != nil {
		return nil, err
	}

	for k, v := range opt.header {
		req.Header.Set(k, v)
	}

	// Set the context for the request
	if opt.ctx != nil {
		req = req.WithContext(opt.ctx)
	}

	resp, err := cli.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check if the response status is not in the 2xx range
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		buf := cli.bufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		defer cli.bufferPool.Put(buf)

		// Read the response body into the buffer
		if _, err = io.Copy(buf, resp.Body); err != nil {
			return nil, errors.BadRequest("copy response body: %v", err)
		}

		return nil, errors.New(resp.StatusCode, "%s", buf.String())
	}

	if opt.response != nil {
		// Decode directly from response body to target object, avoiding intermediate memory allocation
		dec := json.NewDecoder(resp.Body)
		if err = dec.Decode(opt.response); err != nil {
			return nil, errors.BadRequest("decode response body: %v", err)
		}
	}

	return resp, nil
}
