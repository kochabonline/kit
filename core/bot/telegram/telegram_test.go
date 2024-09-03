package telegram

import (
	"testing"
	"time"
)

func TestSend(t *testing.T) {
	tg := New("7371112917:xx")
	resp, err := tg.Send(NewSendMessage().
		With().
		ChatId(-12345678).
		Text("Hello, World").
		ParseMode(MarkdownV2).
		Message())
	if err != nil {
		t.Error(err)
	}
	t.Log(resp)
}

func TestHandleCommands(t *testing.T) {
	tg := New("7023209709:xx")
	tg.HandleCommands()
	defer tg.Close()
	time.Sleep(20 * time.Second)
}
