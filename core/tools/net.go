package tools

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/kochabonline/kit/errors"
)

type UrlOption struct {
	Refs   []string
	Params map[string]string
}

func WithUrlRefs(refs ...string) func(*UrlOption) {
	return func(opt *UrlOption) {
		opt.Refs = refs
	}
}

func WithUrlParams(params map[string]string) func(*UrlOption) {
	return func(opt *UrlOption) {
		opt.Params = params
	}
}

func Url(base string, opts ...func(*UrlOption)) string {
	baseUrl, err := url.Parse(base)
	if err != nil {
		return ""
	}

	opt := &UrlOption{}
	for _, o := range opts {
		o(opt)
	}

	if len(opt.Refs) > 0 {
		for _, ref := range opt.Refs {
			baseUrl.Path, err = url.JoinPath(baseUrl.Path, ref)
			if err != nil {
				return ""
			}
		}
	}
	if len(opt.Params) > 0 {
		q := baseUrl.Query()
		for k, v := range opt.Params {
			q.Add(k, v)
		}
		baseUrl.RawQuery = q.Encode()
	}

	return baseUrl.String()
}

type Client interface {
	Request(method, url string, body io.Reader, opts ...func(*RequestOption)) (any, error)
}

type Http struct {
	client *http.Client
}

type Option func(*Http)

func WithHttpClient(client *http.Client) func(*Http) {
	return func(h *Http) {
		h.client = client
	}
}

func New(opts ...Option) *Http {
	h := &Http{
		client: &http.Client{},
	}
	for _, o := range opts {
		o(h)
	}
	return h
}

type RequestOption struct {
	header   map[string]string
	response any
}

func WithRequestHeader(header map[string]string) func(*RequestOption) {
	return func(opt *RequestOption) {
		opt.header = header
	}
}

// Must pass a pointer
// Passing a pointer is to modify the value of an external variable within the function
func WithRequestResponse(response any) func(*RequestOption) {
	return func(opt *RequestOption) {
		opt.response = response
	}
}

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
		return errors.BadRequest("http request failed", string(respByte))
	}

	if opt.response != nil {
		err = json.Unmarshal(respByte, &opt.response)
		if err != nil {
			return err
		}
	}

	return nil
}
