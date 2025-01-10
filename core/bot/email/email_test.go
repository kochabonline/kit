package email

import "testing"

func TestEmail(t *testing.T) {
	e := New(SmtpPlainAuth{})
	if _, err := e.Send(NewMessage().With().
		Type(Markdown).
		To([]string{""}).
		Subject("test").
		Body("# hello world\n### hello world").
		Message()); err != nil {
		t.Fatal(err)
	}
}
