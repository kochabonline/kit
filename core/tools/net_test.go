package tools

import (
	"testing"
)

type Response struct {
	Status string `json:"status"`
}

func TestHttp(t *testing.T) {
	url := Url("http://localhost:8080", WithUrlRefs("health"))
	var response Response
	client := New()
	err := client.Request("GET", url, nil, WithRequestResponse(&response))
	if err != nil {
		t.Error(err)
	}
	t.Log(response.Status)
}
