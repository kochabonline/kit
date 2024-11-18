package bot

import (
	"net/http"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Bot interface {
	Send(msg Message) (*http.Response, error)
}

type Message interface {
	Marshal() ([]byte, error)
}
