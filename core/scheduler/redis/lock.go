package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/kochabonline/kit/log"
	"github.com/redis/go-redis/v9"
)

// redisLock Redis分布式锁实现
type redisLock struct {
	client redis.Cmdable
	script *redis.Script
}

// NewRedisLock 创建Redis分布式锁
func NewRedisLock(client redis.Cmdable) DistributedLock {
	// Lua脚本确保原子性释放锁
	unlockScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	return &redisLock{
		client: client,
		script: redis.NewScript(unlockScript),
	}
}

// TryLock 尝试获取锁，非阻塞
func (r *redisLock) TryLock(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	result, err := r.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	return result, nil
}

// Lock 获取锁，阻塞直到获取成功或超时
func (r *redisLock) Lock(ctx context.Context, key string, value string, ttl time.Duration, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		// 检查上下文是否已取消
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// 检查是否超时
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout acquiring lock after %v", timeout)
		}

		// 尝试获取锁
		acquired, err := r.TryLock(ctx, key, value, ttl)
		if err != nil {
			return err
		}

		if acquired {
			return nil
		}

		// 等待一段时间后重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// 继续循环
		}
	}
}

// Unlock 释放锁
func (r *redisLock) Unlock(ctx context.Context, key string, value string) error {
	result, err := r.script.Run(ctx, r.client, []string{key}, value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result.(int64) == 0 {
		return fmt.Errorf("lock not held by this value")
	}

	return nil
}

// Refresh 续期锁
func (r *redisLock) Refresh(ctx context.Context, key string, value string, ttl time.Duration) error {
	// 使用Lua脚本确保原子性续期
	refreshScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("expire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	script := redis.NewScript(refreshScript)
	result, err := script.Run(ctx, r.client, []string{key}, value, int64(ttl.Seconds())).Result()
	if err != nil {
		return fmt.Errorf("failed to refresh lock: %w", err)
	}

	if result.(int64) == 0 {
		return fmt.Errorf("lock not held by this value")
	}

	return nil
}

// generateLockValue 生成唯一的锁值
func generateLockValue() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// redisLeaderElector Redis Leader选举器实现
type redisLeaderElector struct {
	client    redis.Cmdable
	lock      DistributedLock
	nodeID    string
	lockKey   string
	lockValue string

	// 状态管理
	mu       sync.RWMutex
	isLeader bool

	// 控制通道
	stopChan     chan struct{}
	eventChan    chan LeaderElectionEvent
	refreshTimer *time.Timer

	// 配置
	leaseDuration   time.Duration
	refreshInterval time.Duration
}

// NewRedisLeaderElector 创建Redis Leader选举器
func NewRedisLeaderElector(client redis.Cmdable, nodeID string, lockKey string, leaseDuration time.Duration) LeaderElector {
	return &redisLeaderElector{
		client:          client,
		lock:            NewRedisLock(client),
		nodeID:          nodeID,
		lockKey:         lockKey,
		lockValue:       generateLockValue(),
		stopChan:        make(chan struct{}),
		eventChan:       make(chan LeaderElectionEvent, 10),
		leaseDuration:   leaseDuration,
		refreshInterval: leaseDuration / 3, // 每1/3租约时长刷新一次
	}
}

// CampaignLeader 竞选Leader
func (r *redisLeaderElector) CampaignLeader(ctx context.Context) error {
	// 发布竞选事件
	r.publishEvent(LeaderElectionEvent{
		Type:      "campaigning",
		LeaderID:  "",
		NodeID:    r.nodeID,
		Timestamp: time.Now().Unix(),
	})

	// 尝试获取Leader锁
	acquired, err := r.lock.TryLock(ctx, r.lockKey, r.lockValue, r.leaseDuration)
	if err != nil {
		return fmt.Errorf("failed to campaign for leader: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if acquired {
		r.isLeader = true
		log.Info().Str("nodeId", r.nodeID).Msg("elected as leader")

		// 启动续期协程
		go r.refreshLeaderLock(ctx)

		// 发布选举成功事件
		r.publishEvent(LeaderElectionEvent{
			Type:      "elected",
			LeaderID:  r.nodeID,
			NodeID:    r.nodeID,
			Timestamp: time.Now().Unix(),
		})
	}

	return nil
}

// IsLeader 检查是否为Leader
func (r *redisLeaderElector) IsLeader() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isLeader
}

// ResignLeader 主动放弃Leader
func (r *redisLeaderElector) ResignLeader(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isLeader {
		return nil
	}

	// 释放锁
	if err := r.lock.Unlock(ctx, r.lockKey, r.lockValue); err != nil {
		log.Error().Err(err).Msg("failed to release leader lock")
	}

	r.isLeader = false

	// 停止续期定时器
	if r.refreshTimer != nil {
		r.refreshTimer.Stop()
	}

	log.Info().Str("nodeId", r.nodeID).Msg("resigned from leader")

	// 发布Leader丢失事件
	r.publishEvent(LeaderElectionEvent{
		Type:      "lost",
		LeaderID:  r.nodeID,
		NodeID:    r.nodeID,
		Timestamp: time.Now().Unix(),
	})

	return nil
}

// GetLeaderID 获取当前Leader ID
func (r *redisLeaderElector) GetLeaderID(ctx context.Context) (string, error) {
	result, err := r.client.Get(ctx, r.lockKey).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // 没有Leader
		}
		return "", fmt.Errorf("failed to get leader: %w", err)
	}

	// 这里简化处理，实际应用中锁值应该包含nodeID信息
	// 或者使用额外的键来存储Leader信息
	return result, nil
}

// WatchLeaderElection 监听Leader选举事件
func (r *redisLeaderElector) WatchLeaderElection(ctx context.Context) <-chan LeaderElectionEvent {
	return r.eventChan
}

// refreshLeaderLock 续期Leader锁
func (r *redisLeaderElector) refreshLeaderLock(ctx context.Context) {
	ticker := time.NewTicker(r.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-ticker.C:
			// 检查是否还是Leader
			r.mu.RLock()
			isLeader := r.isLeader
			r.mu.RUnlock()

			if !isLeader {
				return
			}

			// 续期锁
			if err := r.lock.Refresh(ctx, r.lockKey, r.lockValue, r.leaseDuration); err != nil {
				log.Error().Err(err).Msg("failed to refresh leader lock")

				// 刷新失败，可能已经不是Leader了
				r.mu.Lock()
				r.isLeader = false
				r.mu.Unlock()

				// 发布Leader丢失事件
				r.publishEvent(LeaderElectionEvent{
					Type:      "lost",
					LeaderID:  r.nodeID,
					NodeID:    r.nodeID,
					Timestamp: time.Now().Unix(),
				})

				return
			}

			log.Debug().Str("nodeId", r.nodeID).Msg("refreshed leader lock")
		}
	}
}

// publishEvent 发布选举事件
func (r *redisLeaderElector) publishEvent(event LeaderElectionEvent) {
	select {
	case r.eventChan <- event:
	default:
		// 如果通道已满，丢弃事件
		log.Warn().Msg("election event channel full, dropping event")
	}
}

// stop 停止选举器
func (r *redisLeaderElector) stop() {
	close(r.stopChan)
	close(r.eventChan)
}

// RedisLockManager Redis锁管理器 - 支持多个锁的管理
type RedisLockManager struct {
	client redis.Cmdable
	locks  sync.Map // key: lockKey, value: *lockInfo
}

// lockInfo 锁信息
type lockInfo struct {
	key      string
	value    string
	ttl      time.Duration
	timer    *time.Timer
	stopChan chan struct{}
	isHeld   bool
	mu       sync.RWMutex
}

// NewRedisLockManager 创建Redis锁管理器
func NewRedisLockManager(client redis.Cmdable) *RedisLockManager {
	return &RedisLockManager{
		client: client,
	}
}

// AcquireLock 获取锁并自动续期
func (m *RedisLockManager) AcquireLock(ctx context.Context, key string, ttl time.Duration) (string, error) {
	value := generateLockValue()
	lock := NewRedisLock(m.client)

	// 尝试获取锁
	acquired, err := lock.TryLock(ctx, key, value, ttl)
	if err != nil {
		return "", err
	}

	if !acquired {
		return "", ErrLockAlreadyHeld
	}

	// 创建锁信息
	info := &lockInfo{
		key:      key,
		value:    value,
		ttl:      ttl,
		stopChan: make(chan struct{}),
		isHeld:   true,
	}

	// 启动自动续期
	go m.autoRenew(ctx, lock, info)

	m.locks.Store(key, info)

	return value, nil
}

// ReleaseLock 释放锁
func (m *RedisLockManager) ReleaseLock(ctx context.Context, key string, value string) error {
	lockValue, exists := m.locks.Load(key)
	if !exists {
		return fmt.Errorf("lock not found: %s", key)
	}

	info := lockValue.(*lockInfo)
	info.mu.Lock()
	defer info.mu.Unlock()

	if info.value != value {
		return fmt.Errorf("lock value mismatch")
	}

	if !info.isHeld {
		return fmt.Errorf("lock already released")
	}

	// 停止自动续期
	close(info.stopChan)
	if info.timer != nil {
		info.timer.Stop()
	}

	// 释放Redis锁
	lock := NewRedisLock(m.client)
	if err := lock.Unlock(ctx, key, value); err != nil {
		log.Error().Err(err).Str("key", key).Msg("failed to release redis lock")
	}

	info.isHeld = false
	m.locks.Delete(key)

	return nil
}

// autoRenew 自动续期锁
func (m *RedisLockManager) autoRenew(ctx context.Context, lock DistributedLock, info *lockInfo) {
	renewInterval := info.ttl / 3 // 每1/3 TTL时间续期一次
	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-info.stopChan:
			return
		case <-ticker.C:
			info.mu.RLock()
			if !info.isHeld {
				info.mu.RUnlock()
				return
			}
			key := info.key
			value := info.value
			ttl := info.ttl
			info.mu.RUnlock()

			if err := lock.Refresh(ctx, key, value, ttl); err != nil {
				log.Error().Err(err).Str("key", key).Msg("failed to refresh lock")

				// 续期失败，标记锁为失效
				info.mu.Lock()
				info.isHeld = false
				info.mu.Unlock()

				m.locks.Delete(key)
				return
			}
		}
	}
}

// IsLockHeld 检查锁是否被持有
func (m *RedisLockManager) IsLockHeld(key string) bool {
	lockValue, exists := m.locks.Load(key)
	if !exists {
		return false
	}

	info := lockValue.(*lockInfo)
	info.mu.RLock()
	defer info.mu.RUnlock()

	return info.isHeld
}

// Close 关闭锁管理器，释放所有锁
func (m *RedisLockManager) Close(ctx context.Context) error {
	lock := NewRedisLock(m.client)

	m.locks.Range(func(key, value interface{}) bool {
		info := value.(*lockInfo)
		info.mu.Lock()
		defer info.mu.Unlock()

		if info.isHeld {
			// 停止自动续期
			close(info.stopChan)
			if info.timer != nil {
				info.timer.Stop()
			}

			// 释放Redis锁
			if err := lock.Unlock(ctx, info.key, info.value); err != nil {
				log.Error().Err(err).Str("key", info.key).Msg("failed to release lock during close")
			}

			info.isHeld = false
		}

		m.locks.Delete(key)
		return true
	})

	return nil
}
