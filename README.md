# Kit - Go微服务工具包

Kit是一个功能丰富的Go语言微服务工具包，提供了构建生产级微服务所需的各种组件和工具。

## 项目特性

- 🚀 **应用框架**: 优雅的服务生命周期管理
- 🔐 **认证授权**: JWT、MFA多因子认证支持
- 🔒 **加密算法**: BCrypt、ECIES、HMAC加密工具
- 📊 **监控指标**: Prometheus集成
- 🗄️ **存储支持**: GORM、Redis、MongoDB、Etcd、Kafka
- ⚡ **限流器**: 令牌桶、滑动窗口算法
- 🌐 **HTTP服务**: 基于Gin的HTTP服务器
- 📝 **日志系统**: 结构化日志与脱敏功能
- 🔍 **参数验证**: 通用验证器支持
- 🎯 **任务调度**: 事件总线和负载均衡

## 快速开始

### 安装

```bash
go get github.com/kochabonline/kit
```

### App模块 - 创建和管理服务

App模块是Kit工具包的核心，提供了完整的应用生命周期管理，支持多服务器运行、优雅关闭和资源清理。

#### 基本使用

```go
package main

import (
    "context"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/kochabonline/kit/app"
    "github.com/kochabonline/kit/transport/http"
)

func main() {
    // 创建Gin引擎
    engine := gin.New()
    engine.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    // 创建HTTP服务器
    httpServer := http.NewServer(":8080", engine)

    // 创建应用实例
    application := app.New(
        app.WithServer(httpServer),
        app.WithShutdownTimeout(30*time.Second),
    )

    // 启动应用
    if err := application.Start(); err != nil {
        panic(err)
    }
}
```

#### 高级配置

```go
package main

import (
    "context"
    "database/sql"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/kochabonline/kit/app"
    "github.com/kochabonline/kit/transport/http"
)

func main() {
    // 创建多个服务
    adminEngine := gin.New()
    adminEngine.GET("/admin/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"service": "admin"})
    })
    adminServer := http.NewServer(":8081", adminEngine)

    apiEngine := gin.New()
    apiEngine.GET("/api/v1/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"service": "api"})
    })
    apiServer := http.NewServer(":8080", apiEngine)

    // 模拟数据库连接
    var db *sql.DB // 实际项目中需要初始化

    // 创建应用实例，支持多服务器和资源清理
    application := app.New(
        // 添加多个服务器
        app.WithServers(adminServer, apiServer),
        
        // 设置自定义上下文
        app.WithContext(context.Background()),
        
        // 配置关闭超时
        app.WithShutdownTimeout(30*time.Second),
        app.WithCleanupTimeout(10*time.Second),
        
        // 添加资源清理函数
        app.WithCleanup("database", func(ctx context.Context) error {
            if db != nil {
                return db.Close()
            }
            return nil
        }, 5*time.Second),
        
        app.WithCleanup("cache", func(ctx context.Context) error {
            // 清理缓存逻辑
            return nil
        }, 3*time.Second),
    )

    // 运行时添加服务器
    metricsEngine := gin.New()
    metricsEngine.GET("/metrics", func(c *gin.Context) {
        c.String(200, "metrics data")
    })
    metricsServer := http.NewServer(":9090", metricsEngine)
    
    if err := application.AddServer(metricsServer); err != nil {
        panic(err)
    }

    // 运行时添加清理函数
    if err := application.AddCleanup("metrics", func(ctx context.Context) error {
        // 清理指标收集器
        return nil
    }, 2*time.Second); err != nil {
        panic(err)
    }

    // 启动应用
    if err := application.Start(); err != nil {
        panic(err)
    }
}
```

#### App模块特性

- **多服务器支持**: 同时运行多个HTTP/gRPC服务器
- **优雅关闭**: 接收系统信号自动优雅关闭所有服务
- **资源清理**: 支持注册清理函数，确保资源正确释放
- **超时控制**: 可配置服务关闭和清理函数的超时时间
- **并发安全**: 线程安全的服务器和清理函数管理
- **错误处理**: 完整的错误处理和日志记录

#### 配置选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `WithContext` | 设置应用根上下文 | `context.Background()` |
| `WithServer` | 添加单个服务器 | - |
| `WithServers` | 添加多个服务器 | - |
| `WithShutdownTimeout` | 设置服务关闭超时时间 | `30s` |
| `WithCleanupTimeout` | 设置清理函数默认超时时间 | `10s` |
| `WithSignals` | 设置自定义关闭信号 | `SIGINT, SIGTERM, SIGQUIT` |
| `WithCleanup` | 添加资源清理函数 | - |

#### 运行时管理

```go
// 获取应用信息
info := application.Info()
fmt.Printf("服务器数量: %d\n", info.ServerCount)
fmt.Printf("清理函数数量: %d\n", info.CleanupCount)
fmt.Printf("是否已启动: %t\n", info.Started)

// 手动停止应用
application.Stop()
```

### 其他模块示例

#### HTTP中间件

```go
import "github.com/kochabonline/kit/transport/http/middleware"

engine.Use(middleware.Logger())
engine.Use(middleware.Recovery())
engine.Use(middleware.CORS())
```

#### JWT认证

```go
import "github.com/kochabonline/kit/core/auth/jwt"

jwtManager := jwt.New(jwt.Config{
    Secret: "your-secret-key",
    Expire: time.Hour * 24,
})

token, err := jwtManager.GenerateToken("user123", map[string]interface{}{
    "role": "admin",
})
```

#### Redis存储

```go
import "github.com/kochabonline/kit/store/redis"

client := redis.New(redis.Config{
    Addr: "localhost:6379",
    DB:   0,
})
```

## 项目结构

```
├── app/              # 应用框架和生命周期管理
├── config/           # 配置管理
├── core/            # 核心功能组件
│   ├── auth/        # 认证相关
│   ├── crypto/      # 加密算法
│   ├── http/        # HTTP工具
│   ├── rate/        # 限流器
│   └── ...
├── errors/          # 错误处理
├── log/             # 日志系统
├── store/           # 存储适配器
├── transport/       # 传输层
└── ...
```
