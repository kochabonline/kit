# Kit

### Quick Start

```golang
import (
    "github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/transport"
	"github.com/kochabonline/kit/transport/http"
)

func main() {
    r := gin.Default()
    httpServer := http.NewServer("", r)
	app := NewApp([]transport.Server{httpServer})
	app.Run()
}
```