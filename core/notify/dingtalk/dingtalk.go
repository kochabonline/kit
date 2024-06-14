package dingtalk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/kochabonline/kit/log"
)

type DingTalk struct {
	Webhook string `json:"webhook"`
	Secret  string `json:"secret"`
	log     *log.Helper
}

type Option func(*DingTalk)

func WithLogger(logger *log.Helper) Option {
	return func(d *DingTalk) {
		d.log = logger
	}
}

func New(webhook string, secret string) *DingTalk {
	return &DingTalk{
		Webhook: webhook,
		Secret:  secret,
		log:     log.NewHelper(log.DefaultLogger),
	}
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

func (d *DingTalk) Send(message Message) error {
	msgBytes, err := message.ToBytes()
	if err != nil {
		d.log.Error("failed to marshal message", "error", err, "body", message)
		return err
	}

	req, err := http.NewRequest("POST", d.url(), bytes.NewBuffer(msgBytes))
	if err != nil {
		d.log.Error("failed to create request", "error", err, "body", message)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		d.log.Error("failed to send message", "error", err, "body", message)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		d.log.Error("failed to send message", "status", resp.Status, "body", message)
		return fmt.Errorf("failed to send message, status: %s", resp.Status)
	}
	defer resp.Body.Close()

	return nil
}
