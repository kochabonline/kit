package http

import (
	"testing"
)

type Response struct {
	Status string `json:"status"`
}

func TestHttp(t *testing.T) {
	url := Url("http://localhost:8080", WithUrlRefs("health"))
	var response Response
	http := New()
	err := http.Request(MethodGet, url, nil, WithResponse(&response))
	if err != nil {
		t.Error(err)
	}
	t.Log(response.Status)
}
