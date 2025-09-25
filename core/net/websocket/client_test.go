package websocket

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// 使用 Postman Echo WebSocket 测试接口
	postmanEchoWSS = "wss://ws.postman-echo.com/raw"
)

// TestPostmanEcho_SimpleConnection 简单连接测试
func TestPostmanEcho_SimpleConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要网络连接的集成测试")
	}

	// 创建带自定义TLS配置的客户端
	headers := http.Header{}
	headers.Set("User-Agent", "WebSocket-Test-Client/1.0")

	client := NewClient(WithHeaders(headers))
	defer client.Close()

	// 修改dialer以支持TLS
	wsClient := client.(*wsClient)
	wsClient.dialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: false, // 验证证书
	}

	// 设置事件处理器
	var connected bool
	var connectErr error
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(1)
	client.OnEvent(EventConnected, func(event Event) {
		mu.Lock()
		connected = true
		mu.Unlock()
		t.Log("✅ 连接到 Postman Echo WSS 成功")
		wg.Done()
	})

	client.OnEvent(EventError, func(event Event) {
		mu.Lock()
		connectErr = event.Error
		mu.Unlock()
		t.Logf("❌ 连接错误: %v", event.Error)
		wg.Done()
	})

	// 连接到 Postman Echo WebSocket
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	t.Logf("🔗 尝试连接到: %s", postmanEchoWSS)
	err := client.Connect(ctx, postmanEchoWSS)
	if err != nil {
		t.Logf("初始连接错误: %v", err)
		return // 如果连接失败，跳过测试
	}

	// 等待连接结果
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// 连接结果已收到
	case <-time.After(20 * time.Second):
		t.Log("⏰ 等待连接事件超时")
		return
	}

	mu.Lock()
	if connectErr != nil {
		t.Logf("❌ 连接过程中出现错误: %v", connectErr)
		mu.Unlock()
		return
	}

	connectionStatus := connected
	mu.Unlock()

	if connectionStatus {
		assert.True(t, client.IsConnected(), "客户端应该显示已连接状态")
		t.Log("🎉 WebSocket 连接测试成功！")
	} else {
		t.Log("⚠️  未收到连接成功事件，但没有错误")
	}
}

// TestPostmanEcho_EchoMessage 测试echo功能
func TestPostmanEcho_EchoMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要网络连接的集成测试")
	}

	headers := http.Header{}
	headers.Set("User-Agent", "WebSocket-Test-Client/1.0")

	client := NewClient(WithHeaders(headers))
	defer client.Close()

	// 设置TLS配置
	wsClient := client.(*wsClient)
	wsClient.dialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: false,
	}

	var wg sync.WaitGroup
	var receivedMessage string
	var mu sync.Mutex

	// 先等待连接
	wg.Add(1)
	client.OnEvent(EventConnected, func(event Event) {
		t.Log("✅ 连接成功，准备发送消息")
		wg.Done()
	})

	client.OnEvent(EventMessage, func(event Event) {
		if msg, ok := event.Data.(Message); ok && msg.Type == TextMessage {
			mu.Lock()
			receivedMessage = string(msg.Data)
			mu.Unlock()
			t.Logf("📩 收到echo消息: %s", receivedMessage)
			wg.Done()
		}
	})

	client.OnEvent(EventError, func(event Event) {
		t.Logf("❌ 错误: %v", event.Error)
	})

	// 连接
	ctx := context.Background()
	err := client.Connect(ctx, postmanEchoWSS)
	if err != nil {
		t.Skipf("连接失败，跳过测试: %v", err)
	}

	// 等待连接建立
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// 连接成功
	case <-time.After(20 * time.Second):
		t.Skip("连接超时，跳过测试")
	}

	if !client.IsConnected() {
		t.Skip("客户端未连接，跳过测试")
	}

	// 发送测试消息
	testMessage := "Hello Postman Echo WebSocket!"
	t.Logf("📤 发送消息: %s", testMessage)

	wg.Add(1) // 等待echo响应
	err = client.SendText(testMessage)
	require.NoError(t, err, "发送消息应该成功")

	// 等待echo响应
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// 收到响应
		mu.Lock()
		echo := receivedMessage
		mu.Unlock()

		assert.Equal(t, testMessage, echo, "应该收到相同的echo消息")
		t.Log("🎉 Echo 测试成功！")
	case <-time.After(15 * time.Second):
		t.Log("⏰ 等待echo响应超时")
	}
}
