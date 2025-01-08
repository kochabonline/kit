package http

import (
	"net/url"
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

// Url returns the URL string by joining the base URL with the references and parameters.
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
