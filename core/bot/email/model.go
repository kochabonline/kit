package email

import "encoding/json"

type Message struct {
	Type    string   `json:"type"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

type MessageContext struct {
	message *Message
}

func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func NewMessage() *Message {
	return &Message{}
}

func (m *Message) With() *MessageContext {
	return &MessageContext{message: m}
}

func (c *MessageContext) Type(t string) *MessageContext {
	c.message.Type = t
	return c
}

func (c *MessageContext) To(to []string) *MessageContext {
	c.message.To = to
	return c
}

func (c *MessageContext) Subject(subject string) *MessageContext {
	c.message.Subject = subject
	return c
}

func (c *MessageContext) Body(body string) *MessageContext {
	c.message.Body = body
	return c
}

func (c *MessageContext) Message() *Message {
	return c.message
}
