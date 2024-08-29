package telegram

import (
	"testing"
	"time"
)

func TestSend(t *testing.T) {
	tg := New("7371112917:AAEysG0bv4CEEnRrdMHK0LNWR6KUTHRjiuw")
	resp, err := tg.Send(NewSendMessage().
		With().
		ChatId(-4592139533).
		Text("Hello, World").
		ParseMode(MarkdownV2).
		Message())
	if err != nil {
		t.Error(err)
	}
	t.Log(resp)
}

func TestHandleCommands(t *testing.T) {
	tg := New("7371112917:AAEysG0bv4CEEnRrdMHK0LNWR6KUTHRjiuw")
	tg.HandleCommands()
	defer tg.Close()
	time.Sleep(20 * time.Second)
}
