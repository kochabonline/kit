# Log 日志模块

基于 [zerolog](https://github.com/rs/zerolog) 的高性能日志库，提供了结构化日志记录、日志轮转、数据脱敏等功能。

## 特性

- 🚀 **高性能**: 基于 zerolog，零分配的JSON日志记录
- 🔄 **日志轮转**: 支持按时间和大小进行日志轮转
- 🔒 **数据脱敏**: 内置敏感数据脱敏功能，保护隐私信息
- 📝 **多种输出**: 支持控制台、文件、多路输出
- 🎯 **结构化日志**: 支持结构化字段记录
- 📊 **调用栈**: 可选的调用栈信息记录
- 🌐 **全局日志**: 提供全局日志实例，方便使用

## 快速开始

### 基本使用

```go
package main

import (
    "github.com/kochabonline/kit/log"
)

func main() {
    // 使用默认控制台日志
    logger := log.New()
    
    logger.Info().Msg("Hello, World!")
    logger.Debug().Str("key", "value").Msg("Debug with field")
    logger.Error().Err(err).Msg("Error occurred")
    
    // 使用全局日志
    log.Info().Msg("Global info log")
    log.Error().Err(err).Msg("Global error log")
}
```

### 文件日志

```go
config := log.Config{
    RotateMode: log.RotateModeSize,
    Filepath:   "logs",
    Filename:   "app",
    FileExt:    "log",
    LumberjackConfig: log.LumberjackConfig{
        MaxSize:    100,  // MB
        MaxBackups: 5,
        MaxAge:     30,   // days
        Compress:   true,
    },
}

logger := log.NewFile(config)
logger.Info().Msg("File log message")
```

### 同时输出到文件和控制台

```go
logger := log.NewMulti(config)
logger.Info().Msg("Multi output log")
```

## 配置说明

### Config 结构体

```go
type Config struct {
    RotateMode       RotateMode       // 轮转模式：RotateModeTime(按时间) 或 RotateModeSize(按大小)
    Filepath         string           // 日志文件路径，默认: "log"
    Filename         string           // 日志文件名，默认: "app"
    FileExt          string           // 日志文件扩展名，默认: "log"
    RotatelogsConfig RotatelogsConfig // 按时间轮转配置
    LumberjackConfig LumberjackConfig // 按大小轮转配置
}
```

### 按时间轮转配置

```go
type RotatelogsConfig struct {
    MaxAge       int // 日志保留时间(小时)，默认: 24
    RotationTime int // 轮转时间间隔(小时)，默认: 1
}
```

### 按大小轮转配置

```go
type LumberjackConfig struct {
    MaxSize    int  // 单个日志文件最大大小(MB)，默认: 100
    MaxBackups int  // 保留的旧日志文件数量，默认: 5
    MaxAge     int  // 日志文件保留天数，默认: 30
    Compress   bool // 是否压缩旧日志文件，默认: false
}
```

## 高级功能

### 调用栈信息

```go
logger := log.New(log.WithCaller())
logger.Info().Msg("Log with caller info")

// 或者指定跳过的帧数
logger := log.New(log.WithCallerSkip(1))
```

### 数据脱敏

创建带脱敏功能的日志器：

```go
// 创建脱敏钩子
hook := log.NewDesensitizeHook()

// 添加手机号脱敏规则
err := hook.AddContentRule("phone", `1[3-9]\d{9}`, "1****5678")
if err != nil {
    panic(err)
}

// 添加邮箱脱敏规则
err = hook.AddContentRule("email", `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, "***@***.com")
if err != nil {
    panic(err)
}

// 添加JSON字段脱敏规则
err = hook.AddFieldRule("password", "password", ".+", "***")
if err != nil {
    panic(err)
}

// 创建带脱敏功能的日志器
logger := log.New(log.WithDesensitize(hook))

logger.Info().Str("phone", "13812345678").Msg("User info")  // 输出: "13812345678" -> "1****5678"
```

### 全局日志配置

```go
// 设置全局日志级别
log.SetGlobalLevel(zerolog.InfoLevel)

// 设置自定义的全局日志器
logger := log.NewFile(config, log.WithCaller())
log.SetGlobalLogger(logger)
```

## 脱敏功能详解

### 内容规则 (ContentRule)

基于正则表达式匹配日志内容进行脱敏：

```go
hook := log.NewDesensitizeHook()

// 手机号脱敏
hook.AddContentRule("phone", `1[3-9]\d{9}`, "1****5678")

// 身份证号脱敏
hook.AddContentRule("idcard", `\d{17}[\dXx]`, "****")

// 银行卡号脱敏
hook.AddContentRule("bankcard", `\d{16,19}`, "****")
```

### 字段规则 (FieldRule)

基于JSON字段名进行脱敏：

```go
hook := log.NewDesensitizeHook()

// 密码字段脱敏
hook.AddFieldRule("password", "password", ".+", "***")

// 令牌字段脱敏
hook.AddFieldRule("token", "token", ".+", "***")

// 邮箱字段脱敏
hook.AddFieldRule("email", "email", `(.+)@(.+)`, "$1***@***.com")
```

### 规则管理

```go
hook := log.NewDesensitizeHook()

// 添加规则
hook.AddContentRule("phone", `1[3-9]\d{9}`, "1****5678")

// 禁用规则
hook.SetRuleEnabled("phone", false)

// 启用规则
hook.SetRuleEnabled("phone", true)

// 移除规则
hook.RemoveRule("phone")

// 清空所有规则
hook.Clear()
```

## 最佳实践

### 1. 生产环境配置

```go
config := log.Config{
    RotateMode: log.RotateModeSize,
    Filepath:   "/var/log/myapp",
    Filename:   "app",
    FileExt:    "log",
    LumberjackConfig: log.LumberjackConfig{
        MaxSize:    100,  // 100MB
        MaxBackups: 10,   // 保留10个备份文件
        MaxAge:     30,   // 保留30天
        Compress:   true, // 压缩旧日志
    },
}

// 生产环境建议使用文件日志并配置脱敏
hook := log.NewDesensitizeHook()
hook.AddContentRule("phone", `1[3-9]\d{9}`, "1****5678")
hook.AddFieldRule("password", "password", ".+", "***")

logger := log.NewFile(config, 
    log.WithCaller(),
    log.WithDesensitize(hook),
)

log.SetGlobalLogger(logger)
log.SetGlobalLevel(zerolog.InfoLevel)
```

### 2. 开发环境配置

```go
// 开发环境使用控制台输出，便于调试
logger := log.New(log.WithCaller())
log.SetGlobalLogger(logger)
log.SetGlobalLevel(zerolog.DebugLevel)
```

### 3. 结构化日志记录

```go
log.Info().
    Str("user_id", "12345").
    Int("age", 25).
    Dur("elapsed", time.Since(start)).
    Msg("User operation completed")
```

### 4. 错误日志记录

```go
if err != nil {
    log.Error().
        Err(err).
        Str("operation", "database_query").
        Str("table", "users").
        Msg("Database operation failed")
    return err
}
```

## API 参考

### 创建日志器

- `New(opts ...Option) *Logger` - 创建控制台日志器
- `NewFile(config Config, opts ...Option) *Logger` - 创建文件日志器  
- `NewMulti(config Config, opts ...Option) *Logger` - 创建多路输出日志器

### 选项函数

- `WithCaller()` - 添加调用栈信息
- `WithCallerSkip(skip int)` - 添加调用栈信息并跳过指定帧数
- `WithDesensitize(hook *DesensitizeHook)` - 添加脱敏功能

### 全局日志函数

- `Debug() *zerolog.Event` - Debug级别日志
- `Info() *zerolog.Event` - Info级别日志  
- `Warn() *zerolog.Event` - Warn级别日志
- `Error() *zerolog.Event` - Error级别日志(带栈信息)
- `Fatal() *zerolog.Event` - Fatal级别日志(带栈信息)
- `Panic() *zerolog.Event` - Panic级别日志(带栈信息)

### 脱敏相关

- `NewDesensitizeHook() *DesensitizeHook` - 创建脱敏钩子
- `AddContentRule(name, pattern, replacement string) error` - 添加内容脱敏规则
- `AddFieldRule(name, fieldName, pattern, replacement string) error` - 添加字段脱敏规则
- `SetRuleEnabled(name string, enabled bool)` - 启用/禁用脱敏规则
- `RemoveRule(name string)` - 移除脱敏规则
- `Clear()` - 清空所有脱敏规则

## 依赖

- [github.com/rs/zerolog](https://github.com/rs/zerolog) - 高性能日志库
- [github.com/lestrrat-go/file-rotatelogs](https://github.com/lestrrat-go/file-rotatelogs) - 按时间轮转
- [gopkg.in/natefinch/lumberjack.v2](https://gopkg.in/natefinch/lumberjack.v2) - 按大小轮转
