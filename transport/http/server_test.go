package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServerRun(t *testing.T) {
	s := NewServer("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	go func() {
		err := s.Run()
		assert.NoError(t, err)
	}()

	time.Sleep(time.Second) // give server time to start

	resp, err := http.Get("http://localhost:8080")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
