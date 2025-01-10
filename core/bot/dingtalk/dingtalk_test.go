package dingtalk

import "testing"

func TestSend(t *testing.T) {
	dt := New("", "")
	_, err := dt.Send(NewMarkdownMessage().With().Title("test").Text("test").Message())
	if err != nil {
		t.Fatal(err)
	}
}
