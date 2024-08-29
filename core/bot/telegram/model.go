package telegram

import (
	"encoding/json"
)

// https://core.telegram.org/bots/api#making-requests
type ApiResponse[T any] struct {
	Ok                 bool   `json:"ok"`
	Result             []T    `json:"result"`
	ErrorCode          int    `json:"error_code,omitempty"`
	Description        string `json:"description,omitempty"`
	ResponseParameters struct {
		MigrateToChatID int64 `json:"migrate_to_chat_id,omitempty"`
		RetryAfter      int   `json:"retry_after,omitempty"`
	} `json:"response_parameters,omitempty"`
}

// https://core.telegram.org/bots/api#message
type Message struct {
	MessageID int64    `json:"message_id"`
	From      From     `json:"from"`
	Chat      Chat     `json:"chat"`
	Date      int64    `json:"date"`
	Text      string   `json:"text"`
	Entities  []Entity `json:"entities"`
}

type Chat struct {
	ID                          int64  `json:"id"`
	Title                       string `json:"title"`
	Type                        string `json:"type"`
	AllMembersAreAdministrators bool   `json:"all_members_are_administrators"`
}

type From struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

type Entity struct {
	Offset int64  `json:"offset"`
	Length int    `json:"length"`
	Type   string `json:"type"`
}

func (m *Message) BotCommand() string {
	if len(m.Entities) == 0 {
		return ""
	}

	if m.Entities[0].Type == "bot_command" {
		return m.Text[m.Entities[0].Offset+1 : m.Entities[0].Length]
	}

	return ""
}

type Update struct {
	UpdateID int64   `json:"update_id"`
	Message  Message `json:"message"`
}

// https://core.telegram.org/bots/api#sendmessage
type SendMessage struct {
	ChatId           int64  `json:"chat_id"`
	Text             string `json:"text"`
	ParseMode        string `json:"parse_mode"`
	ReplyToMessageId int64  `json:"reply_to_message_id,omitempty"`
}

type SendMessageContext struct {
	message *SendMessage
}

func (m *SendMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *SendMessage) String() string {
	b, _ := m.Marshal()
	return string(b)
}

func NewSendMessage() *SendMessage {
	return &SendMessage{}
}

func (m *SendMessage) With() *SendMessageContext {
	return &SendMessageContext{message: m}
}

func (c *SendMessageContext) ChatId(chatId int64) *SendMessageContext {
	c.message.ChatId = chatId
	return c
}

func (c *SendMessageContext) Text(text string) *SendMessageContext {
	c.message.Text = text
	return c
}

// ParseMode can be "MarkdownV2" or "HTML" or "Markdown"
// https://core.telegram.org/bots/api#formatting-options
func (c *SendMessageContext) ParseMode(parseMode string) *SendMessageContext {
	c.message.ParseMode = parseMode
	return c
}

func (c *SendMessageContext) Message() *SendMessage {
	return c.message
}
