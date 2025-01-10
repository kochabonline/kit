package lark

import "testing"

func TestLark(t *testing.T) {
	lark := New("", "")
	resp, err := lark.Send(NewCardMessage().With().CardElement(Markdown, "**hello world**").Message())
	if err != nil {
		t.Error(err)
	}
	t.Log(resp)
}
