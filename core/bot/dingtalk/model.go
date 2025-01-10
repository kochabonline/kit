package dingtalk

import "encoding/json"

type TextMessage struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
	At struct {
		AtMobiles []string `json:"atMobiles"`
		AtUserIds []string `json:"atUserIds"`
		IsAtAll   bool     `json:"isAtAll"`
	} `json:"at"`
}

type TextMessageContext struct {
	message *TextMessage
}

func (m *TextMessage) Marshal() ([]byte, error) {
	m.MsgType = Text
	return json.Marshal(m)
}

func (m *TextMessage) String() string {
	b, _ := m.Marshal()
	return string(b)
}

func NewTextMessage() *TextMessage {
	return &TextMessage{
		MsgType: Text,
	}
}

func (m *TextMessage) With() *TextMessageContext {
	return &TextMessageContext{message: m}
}

func (c *TextMessageContext) Content(content string) *TextMessageContext {
	c.message.Text.Content = content
	return c
}

func (c *TextMessageContext) AtMobiles(atMobiles []string) *TextMessageContext {
	c.message.At.AtMobiles = atMobiles
	return c
}

func (c *TextMessageContext) AtUserIds(atUserIds []string) *TextMessageContext {
	c.message.At.AtUserIds = atUserIds
	return c
}

func (c *TextMessageContext) IsAtAll(isAtAll bool) *TextMessageContext {
	c.message.At.IsAtAll = isAtAll
	return c
}

func (c *TextMessageContext) Message() *TextMessage {
	return c.message
}

type MarkdownMessage struct {
	MsgType  string `json:"msgtype"`
	Markdown struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"markdown"`
	At struct {
		AtMobiles []string `json:"atMobiles"`
		AtUserIds []string `json:"atUserIds"`
		IsAtAll   bool     `json:"isAtAll"`
	} `json:"at"`
}

type MarkdownMessageContext struct {
	message *MarkdownMessage
}

func (m *MarkdownMessage) Marshal() ([]byte, error) {
	m.MsgType = Markdown
	return json.Marshal(m)
}

func (m *MarkdownMessage) String() string {
	b, _ := m.Marshal()
	return string(b)
}

func NewMarkdownMessage() *MarkdownMessage {
	return &MarkdownMessage{
		MsgType: Markdown,
	}
}

func (m *MarkdownMessage) With() *MarkdownMessageContext {
	return &MarkdownMessageContext{message: m}
}

func (c *MarkdownMessageContext) Title(title string) *MarkdownMessageContext {
	c.message.Markdown.Title = title
	return c
}

func (c *MarkdownMessageContext) Text(text string) *MarkdownMessageContext {
	c.message.Markdown.Text = text
	return c
}

func (c *MarkdownMessageContext) AtMobiles(atMobiles []string) *MarkdownMessageContext {
	c.message.At.AtMobiles = atMobiles
	return c
}

func (c *MarkdownMessageContext) AtUserIds(atUserIds []string) *MarkdownMessageContext {
	c.message.At.AtUserIds = atUserIds
	return c
}

func (c *MarkdownMessageContext) IsAtAll(isAtAll bool) *MarkdownMessageContext {
	c.message.At.IsAtAll = isAtAll
	return c
}

func (c *MarkdownMessageContext) Message() *MarkdownMessage {
	return c.message
}

type LinkMessage struct {
	MsgType string `json:"msgtype"`
	Link    struct {
		Title      string `json:"title"`
		Text       string `json:"text"`
		MessageURL string `json:"messageUrl"`
		PicURL     string `json:"picUrl"`
	} `json:"link"`
}

type LinkMessageContext struct {
	message *LinkMessage
}

func (m *LinkMessage) Marshal() ([]byte, error) {
	m.MsgType = Link
	return json.Marshal(m)
}

func (m *LinkMessage) String() string {
	b, _ := m.Marshal()
	return string(b)
}

func NewLinkMessage() *LinkMessage {
	return &LinkMessage{
		MsgType: Link,
	}
}

func (m *LinkMessage) With() *LinkMessageContext {
	return &LinkMessageContext{message: m}
}

func (c *LinkMessageContext) Title(title string) *LinkMessageContext {
	c.message.Link.Title = title
	return c
}

func (c *LinkMessageContext) Text(text string) *LinkMessageContext {
	c.message.Link.Text = text
	return c
}

func (c *LinkMessageContext) MessageURL(messageURL string) *LinkMessageContext {
	c.message.Link.MessageURL = messageURL
	return c
}

func (c *LinkMessageContext) PicURL(picURL string) *LinkMessageContext {
	c.message.Link.PicURL = picURL
	return c
}

func (c *LinkMessageContext) Message() *LinkMessage {
	return c.message
}

type ActionCardMessage struct {
	MsgType    string `json:"msgtype"`
	ActionCard struct {
		Title       string `json:"title"`
		Text        string `json:"text"`
		SingleTitle string `json:"singleTitle"`
		SingleURL   string `json:"singleURL"`
	} `json:"actionCard"`
}

type ActionCardMessageContext struct {
	message *ActionCardMessage
}

func (m *ActionCardMessage) Marshal() ([]byte, error) {
	m.MsgType = ActionCard
	return json.Marshal(m)
}

func (m *ActionCardMessage) String() string {
	b, _ := m.Marshal()
	return string(b)
}

func NewActionCardMessage() *ActionCardMessage {
	return &ActionCardMessage{
		MsgType: ActionCard,
	}
}

func (m *ActionCardMessage) With() *ActionCardMessageContext {
	return &ActionCardMessageContext{message: m}
}

func (c *ActionCardMessageContext) Title(title string) *ActionCardMessageContext {
	c.message.ActionCard.Title = title
	return c
}

func (c *ActionCardMessageContext) Text(text string) *ActionCardMessageContext {
	c.message.ActionCard.Text = text
	return c
}

func (c *ActionCardMessageContext) SingleTitle(singleTitle string) *ActionCardMessageContext {
	c.message.ActionCard.SingleTitle = singleTitle
	return c
}

func (c *ActionCardMessageContext) SingleURL(singleURL string) *ActionCardMessageContext {
	c.message.ActionCard.SingleURL = singleURL
	return c
}

func (c *ActionCardMessageContext) Message() *ActionCardMessage {
	return c.message
}

type FeedCardMessage struct {
	MsgType  string `json:"msgtype"`
	FeedCard struct {
		Links []struct {
			Title      string `json:"title"`
			MessageURL string `json:"messageURL"`
			PicURL     string `json:"picURL"`
		} `json:"links"`
	} `json:"feedCard"`
}

type FeedCardMessageContext struct {
	message *FeedCardMessage
}

func (m *FeedCardMessage) Marshal() ([]byte, error) {
	m.MsgType = FeedCard
	return json.Marshal(m)
}

func (m *FeedCardMessage) String() string {
	b, _ := m.Marshal()
	return string(b)
}

func NewFeedCardMessage() *FeedCardMessage {
	return &FeedCardMessage{
		MsgType: FeedCard,
	}
}

func (m *FeedCardMessage) With() *FeedCardMessageContext {
	return &FeedCardMessageContext{message: m}
}

func (c *FeedCardMessageContext) AddLink(title, messageURL, picURL string) *FeedCardMessageContext {
	c.message.FeedCard.Links = append(c.message.FeedCard.Links, struct {
		Title      string `json:"title"`
		MessageURL string `json:"messageURL"`
		PicURL     string `json:"picURL"`
	}{title, messageURL, picURL})
	return c
}

func (c *FeedCardMessageContext) Message() *FeedCardMessage {
	return c.message
}
