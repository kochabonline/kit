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

func (m *TextMessage) Marshal() ([]byte, error) {
	m.MsgType = "text"
	return json.Marshal(m)
}

func (m *TextMessage) String() string {
	b, _ := m.Marshal()
	return string(b)
}

func NewTextMessage() *TextMessage {
	return &TextMessage{
		MsgType: "text",
	}
}

func (m *TextMessage) Content(content string) *TextMessage {
	m.Text.Content = content
	return m
}

func (m *TextMessage) AtMobiles(atMobiles []string) *TextMessage {
	m.At.AtMobiles = atMobiles
	return m
}

func (m *TextMessage) AtUserIds(atUserIds []string) *TextMessage {
	m.At.AtUserIds = atUserIds
	return m
}

func (m *TextMessage) IsAtAll(isAtAll bool) *TextMessage {
	m.At.IsAtAll = isAtAll
	return m
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

func (m *MarkdownMessage) Marshal() ([]byte, error) {
	m.MsgType = "markdown"
	return json.Marshal(m)
}

func (m *MarkdownMessage) String() string {
	b, _ := m.Marshal()
	return string(b)
}

func NewMarkdownMessage() *MarkdownMessage {
	return &MarkdownMessage{
		MsgType: "markdown",
	}
}

func (m *MarkdownMessage) Title(title string) *MarkdownMessage {
	m.Markdown.Title = title
	return m
}

func (m *MarkdownMessage) Text(text string) *MarkdownMessage {
	m.Markdown.Text = text
	return m
}

func (m *MarkdownMessage) AtMobiles(atMobiles []string) *MarkdownMessage {
	m.At.AtMobiles = atMobiles
	return m
}

func (m *MarkdownMessage) AtUserIds(atUserIds []string) *MarkdownMessage {
	m.At.AtUserIds = atUserIds
	return m
}

func (m *MarkdownMessage) IsAtAll(isAtAll bool) *MarkdownMessage {
	m.At.IsAtAll = isAtAll
	return m
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

func (m *LinkMessage) Marshal() ([]byte, error) {
	m.MsgType = "link"
	return json.Marshal(m)
}

func (m *LinkMessage) String() string {
	b, _ := m.Marshal()
	return string(b)
}

func NewLinkMessage() *LinkMessage {
	return &LinkMessage{
		MsgType: "link",
	}
}

func (m *LinkMessage) Title(title string) *LinkMessage {
	m.Link.Title = title
	return m
}

func (m *LinkMessage) Text(text string) *LinkMessage {
	m.Link.Text = text
	return m
}

func (m *LinkMessage) MessageURL(messageURL string) *LinkMessage {
	m.Link.MessageURL = messageURL
	return m
}

func (m *LinkMessage) PicURL(picURL string) *LinkMessage {
	m.Link.PicURL = picURL
	return m
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

func (m *ActionCardMessage) Marshal() ([]byte, error) {
	m.MsgType = "actionCard"
	return json.Marshal(m)
}

func (m *ActionCardMessage) String() string {
	b, _ := m.Marshal()
	return string(b)
}

func NewActionCardMessage() *ActionCardMessage {
	return &ActionCardMessage{
		MsgType: "actionCard",
	}
}

func (m *ActionCardMessage) Title(title string) *ActionCardMessage {
	m.ActionCard.Title = title
	return m
}

func (m *ActionCardMessage) Text(text string) *ActionCardMessage {
	m.ActionCard.Text = text
	return m
}

func (m *ActionCardMessage) SingleTitle(singleTitle string) *ActionCardMessage {
	m.ActionCard.SingleTitle = singleTitle
	return m
}

func (m *ActionCardMessage) SingleURL(singleURL string) *ActionCardMessage {
	m.ActionCard.SingleURL = singleURL
	return m
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

func (m *FeedCardMessage) Marshal() ([]byte, error) {
	m.MsgType = "feedCard"
	return json.Marshal(m)
}

func (m *FeedCardMessage) String() string {
	b, _ := m.Marshal()
	return string(b)
}

func NewFeedCardMessage() *FeedCardMessage {
	return &FeedCardMessage{
		MsgType: "feedCard",
	}
}

func (m *FeedCardMessage) AddLink(title, messageURL, picURL string) *FeedCardMessage {
	m.FeedCard.Links = append(m.FeedCard.Links, struct {
		Title      string `json:"title"`
		MessageURL string `json:"messageURL"`
		PicURL     string `json:"picURL"`
	}{title, messageURL, picURL})
	return m
}
