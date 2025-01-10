package lark

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/kochabonline/kit/core/bot"
	"github.com/kochabonline/kit/errors"
)

var _ bot.Bot = (*Lark)(nil)

type Lark struct {
	Webhook string `json:"webhook"`
	Secret  string `json:"secret"`
	client  *http.Client
}

type Option func(*Lark)

func WithClient(client *http.Client) Option {
	return func(l *Lark) {
		l.client = client
	}
}

func New(webhook string, secret string, opts ...Option) *Lark {
	lark := &Lark{
		Webhook: webhook,
		Secret:  secret,
		client:  http.DefaultClient,
	}
	for _, opt := range opts {
		opt(lark)
	}
	return lark
}

func NewPool(webhook string, secret string, opts ...Option) *Lark {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	l := &Lark{
		Webhook: webhook,
		Secret:  secret,
		client: &http.Client{
			Transport: transport,
		},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *Lark) sign() (int64, string) {
	timestamp := time.Now().Unix()
	toSign := fmt.Sprintf("%v\n%v", timestamp, l.Secret)
	var data []byte
	h := hmac.New(sha256.New, []byte(toSign))
	h.Write(data)
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return timestamp, signature
}

func (l *Lark) Send(msg bot.Sendable) (*http.Response, error) {
	message, ok := msg.(*CardMessage)
	if !ok {
		return nil, errors.BadRequest("invalid message")
	}
	if l.Secret != "" {
		timestamp, signature := l.sign()
		message.Timestamp = timestamp
		message.Sign = signature
	}
	msgBytes, err := message.Marshal()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, l.Webhook, bytes.NewBuffer(msgBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return resp, nil
}
