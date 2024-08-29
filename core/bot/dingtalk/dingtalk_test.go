package dingtalk

import "testing"

func TestSend(t *testing.T) {
	dt := New("", "")
	_, err := dt.Send(NewMarkdownMessage().Title("test").Text("test"))
	if err != nil {
		t.Fatal(err)
	}
}
