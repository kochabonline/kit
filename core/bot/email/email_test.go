package email

import "testing"

func TestEmail(t *testing.T) {
	e := New(SmtpPlainAuth{
		Username: "",
		Password: "",
		Host:     "",
		Port:     25,
	})
	if _, err := e.Send(NewMessage().With().
		Type(Markdown).
		From("kit").
		To([]string{""}).
		Subject("你有一条新的信息").
		Body("# hello world").
		Message()); err != nil {
		t.Fatal(err)
	}
}
