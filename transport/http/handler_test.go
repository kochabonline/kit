package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// MockGinRegister 用于测试的模拟注册器
type MockGinRegister struct {
	RegisterCalled bool
	TestPath       string
}

func NewMockGinRegister(testPath string) *MockGinRegister {
	return &MockGinRegister{
		TestPath: testPath,
	}
}

func (m *MockGinRegister) Register(r gin.IRouter) {
	m.RegisterCalled = true
	// 添加一个简单的路由用于测试
	r.GET(m.TestPath, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})
}

func TestNewHandler(t *testing.T) {
	handler := NewHandler()

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.pool)
	assert.Equal(t, 0, len(handler.pool))
	assert.Equal(t, 0, handler.Count())
}

func TestGinHandler_Add(t *testing.T) {
	tests := []struct {
		name          string
		handlers      []GinRegister
		expectedCount int
		description   string
	}{
		{
			name:          "添加单个处理器",
			handlers:      []GinRegister{NewMockGinRegister("/test1")},
			expectedCount: 1,
			description:   "应该成功添加一个处理器",
		},
		{
			name:          "添加多个处理器",
			handlers:      []GinRegister{NewMockGinRegister("/test1"), NewMockGinRegister("/test2"), NewMockGinRegister("/test3")},
			expectedCount: 3,
			description:   "应该成功添加多个处理器",
		},
		{
			name:          "添加空处理器列表",
			handlers:      []GinRegister{},
			expectedCount: 0,
			description:   "添加空列表应该不改变处理器数量",
		},
		{
			name:          "添加包含nil的处理器列表",
			handlers:      []GinRegister{NewMockGinRegister("/test1"), nil, NewMockGinRegister("/test2")},
			expectedCount: 2,
			description:   "应该过滤掉nil处理器，只添加有效处理器",
		},
		{
			name:          "添加全部为nil的处理器列表",
			handlers:      []GinRegister{nil, nil, nil},
			expectedCount: 0,
			description:   "全部为nil的处理器列表应该不添加任何处理器",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler()
			handler.Add(tt.handlers...)

			assert.Equal(t, tt.expectedCount, handler.Count(), tt.description)
		})
	}
}

func TestGinHandler_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("正常注册处理器", func(t *testing.T) {
		handler := NewHandler()
		mockHandler1 := NewMockGinRegister("/test1")
		mockHandler2 := NewMockGinRegister("/test2")

		handler.Add(mockHandler1, mockHandler2)

		router := gin.New()
		rg := router.Group("/api")

		handler.Register(rg)

		assert.True(t, mockHandler1.RegisterCalled, "第一个处理器应该被调用注册")
		assert.True(t, mockHandler2.RegisterCalled, "第二个处理器应该被调用注册")
	})

	t.Run("传入nil路由组", func(t *testing.T) {
		handler := NewHandler()
		mockHandler := NewMockGinRegister("/test")
		handler.Add(mockHandler)

		// 应该不会panic，也不会调用处理器的Register方法
		assert.NotPanics(t, func() {
			handler.Register(nil)
		})

		assert.False(t, mockHandler.RegisterCalled, "当路由组为nil时，处理器不应该被调用")
	})

	t.Run("空处理器池注册", func(t *testing.T) {
		handler := NewHandler()
		router := gin.New()
		rg := router.Group("/api")

		// 应该不会panic
		assert.NotPanics(t, func() {
			handler.Register(rg)
		})
	})

	t.Run("注册后路由可访问", func(t *testing.T) {
		handler := NewHandler()
		mockHandler := NewMockGinRegister("/test")
		handler.Add(mockHandler)

		router := gin.New()
		rg := router.Group("/api")
		handler.Register(rg)

		// 测试路由是否正确注册
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "test")
	})
}

func TestGinHandler_Count(t *testing.T) {
	handler := NewHandler()

	// 初始计数应该为0
	assert.Equal(t, 0, handler.Count())

	// 添加处理器后计数应该增加
	handler.Add(NewMockGinRegister("/test1"))
	assert.Equal(t, 1, handler.Count())

	handler.Add(NewMockGinRegister("/test2"), NewMockGinRegister("/test3"))
	assert.Equal(t, 3, handler.Count())

	// 添加nil处理器不应该增加计数
	handler.Add(nil)
	assert.Equal(t, 3, handler.Count())
}

func TestGinHandler_Clear(t *testing.T) {
	handler := NewHandler()

	// 添加一些处理器
	handler.Add(NewMockGinRegister("/test1"), NewMockGinRegister("/test2"), NewMockGinRegister("/test3"))
	assert.Equal(t, 3, handler.Count())

	// 清空处理器
	handler.Clear()
	assert.Equal(t, 0, handler.Count())

	// 再次清空应该不会出错
	handler.Clear()
	assert.Equal(t, 0, handler.Count())
}

func TestGinHandler_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 集成测试：测试完整的工作流程
	handler := NewHandler()

	// 创建多个模拟处理器
	mockHandler1 := NewMockGinRegister("/test1")
	mockHandler2 := NewMockGinRegister("/test2")
	mockHandler3 := NewMockGinRegister("/test3")

	// 分批添加处理器
	handler.Add(mockHandler1)
	assert.Equal(t, 1, handler.Count())

	handler.Add(mockHandler2, mockHandler3)
	assert.Equal(t, 3, handler.Count())

	// 创建路由并注册
	router := gin.New()
	rg := router.Group("/api/v1")
	handler.Register(rg)

	// 验证所有处理器都被注册
	assert.True(t, mockHandler1.RegisterCalled)
	assert.True(t, mockHandler2.RegisterCalled)
	assert.True(t, mockHandler3.RegisterCalled)

	// 测试路由功能
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 清空处理器
	handler.Clear()
	assert.Equal(t, 0, handler.Count())

	// 再次添加新处理器并测试
	newMockHandler := NewMockGinRegister("/newtest")
	handler.Add(newMockHandler)
	handler.Register(rg)

	assert.True(t, newMockHandler.RegisterCalled)
	assert.Equal(t, 1, handler.Count())
}
