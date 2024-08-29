package dingtalk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/kochabonline/kit/core/bot"
)

var _ bot.HttpClient = (*DingTalk)(nil)

type DingTalk struct {
	Webhook string `json:"webhook"`
	Secret  string `json:"secret"`
	client  *http.Client
}

type Option func(*DingTalk)

func WithClient(client *http.Client) Option {
	return func(d *DingTalk) {
		d.client = client
	}
}

func New(webhook string, secret string, opts ...Option) *DingTalk {
	d := &DingTalk{
		Webhook: webhook,
		Secret:  secret,
		client:  http.DefaultClient,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func NewPool(webhook string, secret string, opts ...Option) *DingTalk {
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

	d := &DingTalk{
		Webhook: webhook,
		Secret:  secret,
		client: &http.Client{
			Transport: transport,
		},
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *DingTalk) sign() (string, string) {
	timestamp := strconv.FormatInt(time.Now().Unix()*1000, 10)
	stringToSign := fmt.Sprintf("%s\n%s", timestamp, d.Secret)
	h := hmac.New(sha256.New, []byte(d.Secret))
	h.Write([]byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return timestamp, sign
}

func (d *DingTalk) url() string {
	if d.Secret == "" {
		return d.Webhook
	}

	timestamp, sign := d.sign()
	return fmt.Sprintf("%s&timestamp=%s&sign=%s", d.Webhook, timestamp, sign)
}

func (d *DingTalk) Do(req *http.Request) (*http.Response, error) {
	return d.client.Do(req)
}

func (d *DingTalk) Send(message bot.Message) (*http.Response, error) {
	msgBytes, err := message.Marshal()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, d.url(), bytes.NewBuffer(msgBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return resp, nil
}
