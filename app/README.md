# Quick Start

```go
import (
    "github.com/gin-gonic/gin"
    
    "github.com/kochabonline/kit/app"
    "github.com/kochabonline/kit/transport"
    "github.com/kochabonline/kit/transport/http"
)

func main() {
    r := gin.Default()
    httpServer := http.NewServer("", r)
    application := app.NewApp([]transport.Server{httpServer})
    application.Run()
}
```
