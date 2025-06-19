package log

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/rs/zerolog"
)

// DesensitizeRule 脱敏规则接口
type DesensitizeRule interface {
	// Name 返回规则名称
	Name() string
	// Enabled 返回规则是否启用
	Enabled() bool
	// SetEnabled 设置规则启用状态
	SetEnabled(enabled bool)
	// Process 对文本进行脱敏处理
	Process(text string) string
}

// ContentRule 基于内容匹配的脱敏规则
type ContentRule struct {
	name        string
	pattern     *regexp.Regexp
	replacement string
	enabled     int32 // 使用原子操作避免锁竞争
}

// NewContentRule 创建基于内容匹配的脱敏规则
func NewContentRule(name, pattern, replacement string) (*ContentRule, error) {
	if name == "" {
		return nil, fmt.Errorf("rule name cannot be empty")
	}
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern '%s': %w", pattern, err)
	}

	return &ContentRule{
		name:        name,
		pattern:     regex,
		replacement: replacement,
		enabled:     1,
	}, nil
}

func (r *ContentRule) Name() string {
	return r.name
}

func (r *ContentRule) Enabled() bool {
	return atomic.LoadInt32(&r.enabled) == 1
}

func (r *ContentRule) SetEnabled(enabled bool) {
	if enabled {
		atomic.StoreInt32(&r.enabled, 1)
	} else {
		atomic.StoreInt32(&r.enabled, 0)
	}
}

func (r *ContentRule) Process(text string) string {
	if !r.Enabled() {
		return text
	}
	return r.pattern.ReplaceAllString(text, r.replacement)
}

// FieldRule 基于字段名匹配的脱敏规则
type FieldRule struct {
	name         string
	fieldName    string
	fieldPattern *regexp.Regexp
	replacement  string
	jsonPattern  *regexp.Regexp // 预编译的JSON字段匹配模式
	enabled      int32
}

// NewFieldRule 创建基于字段名匹配的脱敏规则
func NewFieldRule(name, fieldName, pattern, replacement string) (*FieldRule, error) {
	if name == "" {
		return nil, fmt.Errorf("rule name cannot be empty")
	}
	if fieldName == "" {
		return nil, fmt.Errorf("field name cannot be empty")
	}
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	fieldPattern, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid field pattern '%s': %w", pattern, err)
	}

	// 预编译JSON字段匹配模式
	jsonPatternStr := fmt.Sprintf(`"%s"\s*:\s*"([^"]*)"`, regexp.QuoteMeta(fieldName))
	jsonPattern, err := regexp.Compile(jsonPatternStr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile json pattern: %w", err)
	}

	return &FieldRule{
		name:         name,
		fieldName:    fieldName,
		fieldPattern: fieldPattern,
		replacement:  replacement,
		jsonPattern:  jsonPattern,
		enabled:      1,
	}, nil
}

func (r *FieldRule) Name() string {
	return r.name
}

func (r *FieldRule) Enabled() bool {
	return atomic.LoadInt32(&r.enabled) == 1
}

func (r *FieldRule) SetEnabled(enabled bool) {
	if enabled {
		atomic.StoreInt32(&r.enabled, 1)
	} else {
		atomic.StoreInt32(&r.enabled, 0)
	}
}

func (r *FieldRule) Process(text string) string {
	if !r.Enabled() {
		return text
	}

	return r.jsonPattern.ReplaceAllStringFunc(text, func(match string) string {
		submatches := r.jsonPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		fieldValue := submatches[1]
		newValue := r.fieldPattern.ReplaceAllString(fieldValue, r.replacement)

		// 保持字段名和引号，只替换值
		return fmt.Sprintf(`"%s":"%s"`, r.fieldName, newValue)
	})
}

// DesensitizeHook 脱敏钩子
type DesensitizeHook struct {
	mu    sync.RWMutex
	rules map[string]DesensitizeRule
}

// NewDesensitizeHook 创建新的脱敏钩子
func NewDesensitizeHook() *DesensitizeHook {
	return &DesensitizeHook{
		rules: make(map[string]DesensitizeRule),
	}
}

// AddContentRule 添加基于内容匹配的脱敏规则 (简化API)
func (h *DesensitizeHook) AddContentRule(name, pattern, replacement string) error {
	rule, err := NewContentRule(name, pattern, replacement)
	if err != nil {
		return err
	}

	h.mu.Lock()
	h.rules[name] = rule
	h.mu.Unlock()

	return nil
}

// AddFieldRule 添加基于字段名匹配的脱敏规则
func (h *DesensitizeHook) AddFieldRule(name, fieldName, pattern, replacement string) error {
	rule, err := NewFieldRule(name, fieldName, pattern, replacement)
	if err != nil {
		return err
	}

	h.mu.Lock()
	h.rules[name] = rule
	h.mu.Unlock()

	return nil
}

// RemoveRule 移除脱敏规则
func (h *DesensitizeHook) RemoveRule(name string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	_, exists := h.rules[name]
	if exists {
		delete(h.rules, name)
	}
	return exists
}

// EnableRule 启用规则
func (h *DesensitizeHook) EnableRule(name string) bool {
	h.mu.RLock()
	rule, exists := h.rules[name]
	h.mu.RUnlock()

	if exists {
		rule.SetEnabled(true)
	}
	return exists
}

// DisableRule 禁用规则
func (h *DesensitizeHook) DisableRule(name string) bool {
	h.mu.RLock()
	rule, exists := h.rules[name]
	h.mu.RUnlock()

	if exists {
		rule.SetEnabled(false)
	}
	return exists
}

// GetRule 获取指定规则
func (h *DesensitizeHook) GetRule(name string) (DesensitizeRule, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	rule, exists := h.rules[name]
	return rule, exists
}

// GetRules 列出所有规则名称
func (h *DesensitizeHook) GetRules() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	names := make([]string, 0, len(h.rules))
	for name := range h.rules {
		names = append(names, name)
	}
	return names
}

// RuleCount 返回规则数量
func (h *DesensitizeHook) RuleCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rules)
}

// Clear 清空所有规则
func (h *DesensitizeHook) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rules = make(map[string]DesensitizeRule)
}

// Run 实现zerolog.Hook接口
func (h *DesensitizeHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// zerolog的Hook机制在消息输出后调用，无法修改消息内容
	// 实际的脱敏处理在Writer层面进行
}

// Desensitize 对文本进行脱敏处理 (公开方法，便于测试)
func (h *DesensitizeHook) Desensitize(text string) string {
	if text == "" {
		return text
	}

	h.mu.RLock()
	if len(h.rules) == 0 {
		h.mu.RUnlock()
		return text
	}

	// 获取所有启用的规则
	rules := make([]DesensitizeRule, 0, len(h.rules))
	for _, rule := range h.rules {
		if rule.Enabled() {
			rules = append(rules, rule)
		}
	}
	h.mu.RUnlock()

	// 如果没有启用的规则，直接返回
	if len(rules) == 0 {
		return text
	}

	// 按顺序应用所有启用的规则
	result := text
	for _, rule := range rules {
		result = rule.Process(result)
	}

	return result
}

// DesensitizeWriter 包装writer以支持脱敏
type DesensitizeWriter struct {
	writer io.Writer
	hook   *DesensitizeHook
	buffer *bytes.Buffer // 复用缓冲区
	mu     sync.Mutex    // 保护buffer
}

// NewDesensitizeWriter 创建脱敏writer
func NewDesensitizeWriter(writer io.Writer, hook *DesensitizeHook) *DesensitizeWriter {
	if writer == nil {
		panic("writer cannot be nil")
	}
	if hook == nil {
		panic("hook cannot be nil")
	}

	return &DesensitizeWriter{
		writer: writer,
		hook:   hook,
		buffer: &bytes.Buffer{},
	}
}

// Write 实现io.Writer接口
func (w *DesensitizeWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// 如果没有规则，直接写入
	if w.hook.RuleCount() == 0 {
		return w.writer.Write(p)
	}

	// 使用unsafeString将字节切片转换为字符串
	text := unsafeString(p)
	desensitized := w.hook.Desensitize(text)

	// 如果内容没有变化，直接写入原始数据
	if desensitized == text {
		return w.writer.Write(p)
	}

	// 使用缓冲区减少内存分配
	w.mu.Lock()
	w.buffer.Reset()
	w.buffer.WriteString(desensitized)
	data := w.buffer.Bytes()
	w.mu.Unlock()

	// 写入脱敏后的数据，返回脱敏后数据的长度
	return w.writer.Write(data)
}

// unsafeString 将字节切片转换为字符串而不复制数据
func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
