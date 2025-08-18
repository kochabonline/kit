package http

import (
	"sync"

	"github.com/gin-gonic/gin"
)

// GinRegister 注册到Gin路由器接口
type GinRegister interface {
	Register(r gin.IRouter)
}

type GinHandler struct {
	pool []GinRegister
	mu   sync.RWMutex
}

// NewHandler 创建一个新的GinHandler实例
func NewHandler() *GinHandler {
	return &GinHandler{
		pool: make([]GinRegister, 0),
	}
}

// Register 将所有处理器注册到给定的路由组
func (h *GinHandler) Register(rg *gin.RouterGroup) {
	if rg == nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, handler := range h.pool {
		if handler != nil {
			handler.Register(rg)
		}
	}
}

// Add 添加一个或多个处理器到处理器池
func (h *GinHandler) Add(handlers ...GinRegister) {
	if len(handlers) == 0 {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// 过滤nil处理器
	validHandlers := make([]GinRegister, 0, len(handlers))
	for _, handler := range handlers {
		if handler != nil {
			validHandlers = append(validHandlers, handler)
		}
	}

	h.pool = append(h.pool, validHandlers...)
}

// Count 返回当前处理器的数量
func (h *GinHandler) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.pool)
}

// Clear 清空所有处理器
func (h *GinHandler) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pool = h.pool[:0]
}
