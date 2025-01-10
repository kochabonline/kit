package lark

import "encoding/json"

type CardMessage struct {
	Timestamp int64  `json:"timestamp"`
	Sign      string `json:"sign"`
	MsgType   string `json:"msg_type"`
	Card      struct {
		Elements []struct {
			Tag  string `json:"tag"`
			Text struct {
				Content string `json:"content"`
				Tag     string `json:"tag"`
			} `json:"text,omitempty"`
		} `json:"elements"`
		Header struct {
			Title struct {
				Content string `json:"content"`
				Tag     string `json:"tag"`
			} `json:"title"`
		} `json:"header"`
	} `json:"card"`
}

type CardMessageContext struct {
	cardMessage *CardMessage
}

func (card *CardMessage) Marshal() ([]byte, error) {
	return json.Marshal(card)
}

func NewCardMessage() *CardMessage {
	return &CardMessage{
		MsgType: "interactive",
	}
}

func (card *CardMessage) With() *CardMessageContext {
	return &CardMessageContext{cardMessage: card}
}

func (c *CardMessageContext) CardHeader(title string) *CardMessageContext {
	c.cardMessage.Card.Header = struct {
		Title struct {
			Content string `json:"content"`
			Tag     string `json:"tag"`
		} `json:"title"`
	}{
		Title: struct {
			Content string `json:"content"`
			Tag     string `json:"tag"`
		}{
			Content: title,
			Tag:     "plain_text",
		},
	}
	return c
}

func (c *CardMessageContext) CardElement(tag, content string) *CardMessageContext {
	c.cardMessage.Card.Elements = append(c.cardMessage.Card.Elements, struct {
		Tag  string `json:"tag"`
		Text struct {
			Content string `json:"content"`
			Tag     string `json:"tag"`
		} `json:"text,omitempty"`
	}{
		Tag: "div",
		Text: struct {
			Content string `json:"content"`
			Tag     string `json:"tag"`
		}{
			Content: content,
			Tag:     tag,
		},
	})
	return c
}

func (c *CardMessageContext) Message() *CardMessage {
	return c.cardMessage
}
