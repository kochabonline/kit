package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/kochabonline/kit/core/bot"
	"github.com/kochabonline/kit/errors"
)

var _ bot.HttpClient = (*Telegram)(nil)

type Telegram struct {
	Token string `json:"token"`

	api      string
	client   *http.Client
	shutdown chan struct{}

	handleCommandsChatId  int64
	handleCommandsErrChan chan error
}

type Option func(*Telegram)

func WithApi(api string) Option {
	return func(d *Telegram) {
		d.api = api
	}
}

func WithClient(client *http.Client) Option {
	return func(d *Telegram) {
		d.client = client
	}
}

func WithHandleCommandsChatId(chatId int64) Option {
	return func(d *Telegram) {
		d.handleCommandsChatId = chatId
	}
}

func WithHandleCommandsErrChan(ch chan error) Option {
	return func(d *Telegram) {
		d.handleCommandsErrChan = ch
	}
}

func New(token string, opts ...Option) *Telegram {
	d := &Telegram{
		Token:    token,
		api:      API,
		client:   http.DefaultClient,
		shutdown: make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func NewPool(token string, opts ...Option) *Telegram {
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

	d := &Telegram{
		Token: token,
		api:   API,
		client: &http.Client{
			Transport: transport,
		},
		shutdown: make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (t *Telegram) url(method string, kvparams ...string) string {
	baseUrl := fmt.Sprintf("%s%s/%s", API, t.Token, method)
	if len(kvparams) == 0 {
		return baseUrl
	}

	params := url.Values{}
	for i := 0; i < len(kvparams); i += 2 {
		params.Add(kvparams[i], kvparams[i+1])
	}

	return fmt.Sprintf("%s?%s", baseUrl, params.Encode())
}

func (t *Telegram) Do(req *http.Request) (*http.Response, error) {
	return t.client.Do(req)
}

func (t *Telegram) Send(message bot.Message) (*http.Response, error) {
	msgBytes, err := message.Marshal()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, t.url(MethodSendMessage), bytes.NewBuffer(msgBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return resp, nil
}

func (t *Telegram) receive(params ...string) (*ApiResponse[Update], error) {
	req, err := http.NewRequest(http.MethodGet, t.url(MethodGetUpdates, params...), nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response ApiResponse[Update]
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if !response.Ok {
		return nil, errors.New(response.ErrorCode, "%s", response.Description)
	}

	return &response, nil
}

func (t *Telegram) process(update Update, fn func() string) error {
	// only process commands from the same chat
	if t.handleCommandsChatId != 0 && t.handleCommandsChatId != update.Message.Chat.ID {
		return nil
	}

	resp, err := t.Send(&SendMessage{
		ChatId:           update.Message.Chat.ID,
		Text:             fn(),
		ParseMode:        MarkdownV2,
		ReplyToMessageId: update.Message.MessageID,
	})
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram failed to send message: %s", resp.Status)
	}

	return nil
}

func (t *Telegram) HandleCommands() {
	updatesChan := make(chan Update)

	// get updates
	go func() {
		var offset int64
		for {
			select {
			case <-t.shutdown:
				close(updatesChan)
				return
			case <-time.After(500 * time.Microsecond):
				updates, err := t.receive("offset", fmt.Sprintf("%d", offset))
				if err != nil && t.handleCommandsErrChan != nil {
					t.handleCommandsErrChan <- fmt.Errorf("Telegram failed to get updates: %v", err)
				}
				if len(updates.Result) > 0 {
					for _, update := range updates.Result {
						offset = update.UpdateID + 1
						updatesChan <- update
					}
				}
			}
		}
	}()

	// process updates
	go func(botCommand *Store) {
		for update := range updatesChan {
			cmd, ok := botCommand.getCommand(update.Message.BotCommand())
			if !ok && t.handleCommandsErrChan != nil {
				t.handleCommandsErrChan <- fmt.Errorf("Telegram received unknown command: %s", update.Message.BotCommand())
				continue
			}
			if err := t.process(update, cmd); err != nil && t.handleCommandsErrChan != nil {
				t.handleCommandsErrChan <- err
			}
		}
	}(BotCommand)
}

func (t *Telegram) Close() {
	t.shutdown <- struct{}{}
}
