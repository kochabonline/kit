package bot

import (
	"net/http"
)

type Bot interface {
	Send(sendable Sendable) (*http.Response, error)
}

type Sendable interface {
	Marshal() ([]byte, error)
}
