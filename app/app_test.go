package app

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/transport"
	"github.com/kochabonline/kit/transport/http"
)

func TestApp(t *testing.T) {
	httpServer := http.NewServer("", gin.New())
	app := NewApp([]transport.Server{httpServer})
	app.Run()
}
