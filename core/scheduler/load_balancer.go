package scheduler

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kochabonline/kit/log"
)

// 性能常量配置
const (
	// 默认虚拟节点数量 - 使用较大的数量以获得更好的分布
	defaultVirtualNodes = 150
	// 最大批处理大小
	maxBatchSize = 256
	// 默认健康阈值
	defaultHealthThreshold = 0.8
	// 最大缓存大小
	maxCacheSize = 1024
	// 原子操作魔数
	atomicMagic = 0x5a5a5a5a
)

var (
	ErrNoWorkersAvailable = errors.New("no workers available")
	ErrInvalidStrategy    = errors.New("invalid load balance strategy")
	ErrWorkerNotHealthy   = errors.New("worker not healthy")
)

// LoadBalancerMetrics 负载均衡器性能指标
type LoadBalancerMetrics struct {
	SelectionCount    uint64 // 选择次数
	SelectionDuration uint64 // 选择总耗时(纳秒)
	CacheHits         uint64 // 缓存命中
	CacheMisses       uint64 // 缓存未命中
	ErrorCount        uint64 // 错误次数
}

// Strategy 负载均衡策略接口
type Strategy interface {
	Select(ctx context.Context, workers []*Worker, task *Task) (*Worker, error)
	Name() string
	Reset()
}

// TaskCountProvider 提供工作节点任务计数的接口
type TaskCountProvider interface {
	getWorkerRunningTaskCount(ctx context.Context, workerID string) (int, error)
}

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	// SelectWorker 根据策略选择最佳工作节点
	SelectWorker(ctx context.Context, workers []*Worker, task *Task) (*Worker, error)
	// SelectWorkers 批量选择工作节点
	SelectWorkers(ctx context.Context, workers []*Worker, tasks []*Task) ([]*Worker, error)
	// GetStrategy 获取当前使用的策略
	GetStrategy() LoadBalanceStrategy
	// SetStrategy 设置负载均衡策略
	SetStrategy(strategy LoadBalanceStrategy)
	// GetMetrics 获取性能指标
	GetMetrics() *LoadBalancerMetrics
	// Reset 重置内部状态（如轮询计数器等）
	Reset()
	// Close 关闭负载均衡器
	Close() error
}

// WorkerCache 工作节点缓存
type WorkerCache struct {
	taskCount int
	timestamp int64
	hash      uint64
}

// FastRoundRobin 轮询计数器
type FastRoundRobin struct {
	counter uint64
	_       [7]uint64 // 缓存行填充，避免false sharing
}

// Next 获取下一个索引
func (r *FastRoundRobin) Next(size int) int {
	if size <= 0 {
		return 0
	}
	return int(atomic.AddUint64(&r.counter, 1) % uint64(size))
}

// Reset 重置计数器
func (r *FastRoundRobin) Reset() {
	atomic.StoreUint64(&r.counter, 0)
}

// BalancerPool 负载均衡器对象池
type BalancerPool struct {
	workerSlicePool sync.Pool
	indexSlicePool  sync.Pool
	sortablePool    sync.Pool
}

// newBalancerPool 创建对象池
func newBalancerPool() *BalancerPool {
	return &BalancerPool{
		workerSlicePool: sync.Pool{
			New: func() any {
				return make([]*Worker, 0, 64)
			},
		},
		indexSlicePool: sync.Pool{
			New: func() any {
				return make([]int, 0, 64)
			},
		},
		sortablePool: sync.Pool{
			New: func() any {
				return &sortableWorkers{}
			},
		},
	}
}

// GetWorkerSlice 获取工作节点切片
func (p *BalancerPool) GetWorkerSlice() []*Worker {
	return p.workerSlicePool.Get().([]*Worker)[:0]
}

// PutWorkerSlice 归还工作节点切片
func (p *BalancerPool) PutWorkerSlice(slice []*Worker) {
	if cap(slice) <= 128 {
		p.workerSlicePool.Put(&slice)
	}
}

// GetIndexSlice 获取索引切片
func (p *BalancerPool) GetIndexSlice() []int {
	return p.indexSlicePool.Get().([]int)[:0]
}

// PutIndexSlice 归还索引切片
func (p *BalancerPool) PutIndexSlice(slice []int) {
	if cap(slice) <= 128 {
		p.indexSlicePool.Put(&slice)
	}
}

// GetSortable 获取可排序对象
func (p *BalancerPool) GetSortable() *sortableWorkers {
	return p.sortablePool.Get().(*sortableWorkers)
}

// PutSortable 归还可排序对象
func (p *BalancerPool) PutSortable(s *sortableWorkers) {
	s.workers = s.workers[:0]
	s.counts = s.counts[:0]
	p.sortablePool.Put(s)
}

// loadBalancer 负载均衡器实现
type loadBalancer struct {
	// 策略和状态
	strategy          atomic.Value // LoadBalanceStrategy
	taskCountProvider TaskCountProvider

	// 轮询状态
	roundRobin FastRoundRobin

	// 随机数生成器
	randPool sync.Pool

	// 对象池
	pool *BalancerPool

	// 性能指标
	metrics LoadBalancerMetrics

	// 工作节点缓存
	cache struct {
		sync.RWMutex
		data map[string]*WorkerCache
	}

	// 一致性哈希环
	hashRing atomic.Value // *ConsistentHashRing

	// 节点追踪
	lastWorkerSet atomic.Value // map[string]bool - 用于检测节点变化

	// 配置参数
	healthThreshold   float64
	enableCache       bool
	cacheTimeout      time.Duration
	virtualNodes      int
	enableMetrics     bool
	batchSize         int
	enableBatching    bool
	enableHealthAware bool

	// 关闭信号
	closed int32
}

// ConsistentHashRing 一致性哈希环
type ConsistentHashRing struct {
	nodes []HashNode
	count int
}

// HashNode 哈希节点
type HashNode struct {
	hash     uint64
	workerID string
	weight   int
}

// sortableWorkers 可排序的工作节点
type sortableWorkers struct {
	workers []*Worker
	counts  []int
}

func (s *sortableWorkers) Len() int { return len(s.workers) }
func (s *sortableWorkers) Swap(i, j int) {
	s.workers[i], s.workers[j] = s.workers[j], s.workers[i]
	s.counts[i], s.counts[j] = s.counts[j], s.counts[i]
}
func (s *sortableWorkers) Less(i, j int) bool {
	if s.counts[i] == s.counts[j] {
		if s.workers[i].Weight == s.workers[j].Weight {
			// 权重相同时，按工作节点ID排序，确保结果可预测但分布均匀
			return s.workers[i].ID < s.workers[j].ID
		}
		return s.workers[i].Weight > s.workers[j].Weight
	}
	return s.counts[i] < s.counts[j]
}

// LoadBalancerConfig 负载均衡器配置
type LoadBalancerConfig struct {
	HealthThreshold    float64
	EnableCache        bool
	CacheTimeout       time.Duration
	VirtualNodes       int
	EnableMetrics      bool
	BatchSize          int
	EnableBatching     bool
	EnableHealthAware  bool
	MaxConcurrentTasks int
	TaskTimeoutSeconds int
}

// DefaultLoadBalancerConfig 默认配置
func DefaultLoadBalancerConfig() *LoadBalancerConfig {
	return &LoadBalancerConfig{
		HealthThreshold:    defaultHealthThreshold,
		EnableCache:        true,
		CacheTimeout:       5 * time.Second,
		VirtualNodes:       defaultVirtualNodes,
		EnableMetrics:      true,
		BatchSize:          32,
		EnableBatching:     true,
		EnableHealthAware:  true,
		MaxConcurrentTasks: 100,
		TaskTimeoutSeconds: 300,
	}
}

// NewLoadBalancer 创建负载均衡器
func NewLoadBalancer(strategy LoadBalanceStrategy, provider TaskCountProvider, config *LoadBalancerConfig) LoadBalancer {
	if config == nil {
		config = DefaultLoadBalancerConfig()
	}

	lb := &loadBalancer{
		taskCountProvider: provider,
		pool:              newBalancerPool(),
		healthThreshold:   config.HealthThreshold,
		enableCache:       config.EnableCache,
		cacheTimeout:      config.CacheTimeout,
		virtualNodes:      config.VirtualNodes,
		enableMetrics:     config.EnableMetrics,
		batchSize:         config.BatchSize,
		enableBatching:    config.EnableBatching,
		enableHealthAware: config.EnableHealthAware,
	}

	// 初始化随机数池
	lb.randPool = sync.Pool{
		New: func() any {
			return rand.New(rand.NewSource(time.Now().UnixNano()))
		},
	}

	// 初始化缓存
	lb.cache.data = make(map[string]*WorkerCache, maxCacheSize)

	// 设置策略
	lb.strategy.Store(strategy)

	// 初始化一致性哈希环
	lb.hashRing.Store(&ConsistentHashRing{})

	// 初始化节点追踪
	lb.lastWorkerSet.Store(make(map[string]bool))

	return lb
}

// SelectWorker 选择最佳工作节点
func (lb *loadBalancer) SelectWorker(ctx context.Context, workers []*Worker, task *Task) (*Worker, error) {
	startTime := time.Now()
	defer func() {
		if lb.enableMetrics {
			atomic.AddUint64(&lb.metrics.SelectionCount, 1)
			atomic.AddUint64(&lb.metrics.SelectionDuration, uint64(time.Since(startTime).Nanoseconds()))
		}
	}()

	if atomic.LoadInt32(&lb.closed) != 0 {
		return nil, errors.New("load balancer is closed")
	}

	if len(workers) == 0 {
		atomic.AddUint64(&lb.metrics.ErrorCount, 1)
		return nil, ErrNoWorkersAvailable
	}

	// 检测节点变化并清理缓存
	lb.handleWorkerSetChange(workers)

	if len(workers) == 1 {
		worker := workers[0]
		if lb.isWorkerAvailable(ctx, worker) {
			return worker, nil
		}
		return nil, ErrNoWorkersAvailable
	}

	// 过滤可用工作节点
	availableWorkers := lb.filterAvailableWorkers(ctx, workers)
	if len(availableWorkers) == 0 {
		atomic.AddUint64(&lb.metrics.ErrorCount, 1)
		return nil, ErrNoWorkersAvailable
	}

	// 根据策略选择工作节点
	strategy := lb.strategy.Load().(LoadBalanceStrategy)
	return lb.selectByStrategy(ctx, availableWorkers, task, strategy)
}

// SelectWorkers 批量选择工作节点
func (lb *loadBalancer) SelectWorkers(ctx context.Context, workers []*Worker, tasks []*Task) ([]*Worker, error) {
	if !lb.enableBatching || len(tasks) == 1 {
		// 退化为单个选择
		results := make([]*Worker, len(tasks))
		for i, task := range tasks {
			worker, err := lb.SelectWorker(ctx, workers, task)
			if err != nil {
				return nil, err
			}
			results[i] = worker
		}
		return results, nil
	}

	return lb.batchSelect(ctx, workers, tasks)
}

// batchSelect 批量选择实现
func (lb *loadBalancer) batchSelect(ctx context.Context, workers []*Worker, tasks []*Task) ([]*Worker, error) {
	availableWorkers := lb.filterAvailableWorkers(ctx, workers)
	if len(availableWorkers) == 0 {
		return nil, ErrNoWorkersAvailable
	}

	results := make([]*Worker, len(tasks))
	strategy := lb.strategy.Load().(LoadBalanceStrategy)

	switch strategy {
	case StrategyRoundRobin:
		for i := range tasks {
			idx := lb.roundRobin.Next(len(availableWorkers))
			results[i] = availableWorkers[idx]
		}
	case StrategyRandom:
		rng := lb.randPool.Get().(*rand.Rand)
		defer lb.randPool.Put(rng)
		for i := range tasks {
			idx := rng.Intn(len(availableWorkers))
			results[i] = availableWorkers[idx]
		}
	default:
		// 其他策略退化为单个选择
		for i, task := range tasks {
			worker, err := lb.selectByStrategy(ctx, availableWorkers, task, strategy)
			if err != nil {
				return nil, err
			}
			results[i] = worker
		}
	}

	return results, nil
}

// selectByStrategy 根据策略选择工作节点
func (lb *loadBalancer) selectByStrategy(ctx context.Context, workers []*Worker, task *Task, strategy LoadBalanceStrategy) (*Worker, error) {
	switch strategy {
	case StrategyLeastTasks:
		return lb.selectByLeastTasks(ctx, workers)
	case StrategyRoundRobin:
		return lb.selectByRoundRobin(workers)
	case StrategyWeightedRoundRobin:
		return lb.selectByWeightedRoundRobin(workers)
	case StrategyRandom:
		return lb.selectByRandom(workers)
	case StrategyConsistentHash:
		return lb.selectByConsistentHash(workers, task)
	case StrategyLeastConnections:
		return lb.selectByLeastTasks(ctx, workers) // 等同于最少任务
	default:
		return lb.selectByLeastTasks(ctx, workers)
	}
}

// filterAvailableWorkers 高性能过滤可用工作节点
func (lb *loadBalancer) filterAvailableWorkers(ctx context.Context, workers []*Worker) []*Worker {
	result := make([]*Worker, 0, len(workers))

	for _, worker := range workers {
		if lb.isWorkerAvailable(ctx, worker) {
			result = append(result, worker)
		}
	}

	return result
}

// isWorkerAvailable 检查工作节点是否可用
func (lb *loadBalancer) isWorkerAvailable(ctx context.Context, worker *Worker) bool {
	// 检查工作节点状态
	if worker.Status != WorkerStatusOnline && worker.Status != WorkerStatusIdle {
		return false
	}

	// 健康检查
	if lb.enableHealthAware {
		if worker.HealthStatus.CPU > lb.healthThreshold ||
			worker.HealthStatus.Memory > lb.healthThreshold {
			return false
		}
	}

	// 检查并发限制
	if worker.RunningTaskNum >= int(worker.MaxConcurrency) {
		return false
	}

	// 使用缓存的任务计数或实时查询
	if lb.enableCache {
		if count, ok := lb.getCachedTaskCount(worker.ID); ok {
			return count < int(worker.MaxConcurrency)
		}
	}

	if lb.taskCountProvider == nil {
		return true // 如果没有提供者，则认为可用
	}

	taskCount, err := lb.taskCountProvider.getWorkerRunningTaskCount(ctx, worker.ID)
	if err != nil {
		log.Warn().Err(err).Str("workerId", worker.ID).Msg("failed to get worker task count")
		return false
	}

	// 更新缓存
	if lb.enableCache {
		lb.setCachedTaskCount(worker.ID, taskCount)
	}

	return taskCount < int(worker.MaxConcurrency)
}

// getCachedTaskCount 获取缓存的任务计数
func (lb *loadBalancer) getCachedTaskCount(workerID string) (int, bool) {
	lb.cache.RLock()
	defer lb.cache.RUnlock()

	if cache, exists := lb.cache.data[workerID]; exists {
		if time.Now().UnixNano()-cache.timestamp < lb.cacheTimeout.Nanoseconds() {
			atomic.AddUint64(&lb.metrics.CacheHits, 1)
			return cache.taskCount, true
		}
	}

	atomic.AddUint64(&lb.metrics.CacheMisses, 1)
	return 0, false
}

// setCachedTaskCount 设置缓存的任务计数
func (lb *loadBalancer) setCachedTaskCount(workerID string, count int) {
	lb.cache.Lock()
	defer lb.cache.Unlock()

	// 限制缓存大小
	if len(lb.cache.data) >= maxCacheSize {
		// 简单的LRU淘汰策略
		oldestKey := ""
		oldestTime := int64(math.MaxInt64)
		for key, cache := range lb.cache.data {
			if cache.timestamp < oldestTime {
				oldestTime = cache.timestamp
				oldestKey = key
			}
		}
		if oldestKey != "" {
			delete(lb.cache.data, oldestKey)
		}
	}

	lb.cache.data[workerID] = &WorkerCache{
		taskCount: count,
		timestamp: time.Now().UnixNano(),
		hash:      atomicMagic,
	}
}

// selectByLeastTasks 最少任务数策略
func (lb *loadBalancer) selectByLeastTasks(ctx context.Context, workers []*Worker) (*Worker, error) {
	if lb.taskCountProvider == nil {
		// 如果没有任务计数提供者，退化为轮询
		return lb.selectByRoundRobin(workers)
	}

	sortable := lb.pool.GetSortable()
	defer lb.pool.PutSortable(sortable)

	sortable.workers = append(sortable.workers, workers...)
	sortable.counts = make([]int, len(workers))

	// 获取所有工作节点的任务计数
	for i, worker := range workers {
		if count, ok := lb.getCachedTaskCount(worker.ID); ok {
			sortable.counts[i] = count
		} else {
			taskCount, err := lb.taskCountProvider.getWorkerRunningTaskCount(ctx, worker.ID)
			if err != nil {
				sortable.counts[i] = int(worker.MaxConcurrency) // 惩罚无法获取计数的节点
			} else {
				sortable.counts[i] = taskCount
				if lb.enableCache {
					lb.setCachedTaskCount(worker.ID, taskCount)
				}
			}
		}
	}

	// 排序找到最小值
	sort.Sort(sortable)

	// 找到所有具有最少任务数的工作节点
	minTasks := sortable.counts[0]
	minTaskWorkers := make([]*Worker, 0, len(workers))

	for i, count := range sortable.counts {
		if count == minTasks {
			minTaskWorkers = append(minTaskWorkers, sortable.workers[i])
		} else {
			break // 因为已排序，后续任务数只会更多
		}
	}

	// 如果只有一个工作节点有最少任务数，直接返回
	if len(minTaskWorkers) == 1 {
		return minTaskWorkers[0], nil
	}

	// 如果有多个工作节点有相同的最少任务数，使用轮询选择
	idx := lb.roundRobin.Next(len(minTaskWorkers))
	return minTaskWorkers[idx], nil
}

// selectByRoundRobin 轮询策略
func (lb *loadBalancer) selectByRoundRobin(workers []*Worker) (*Worker, error) {
	if len(workers) == 0 {
		return nil, ErrNoWorkersAvailable
	}

	// 为了一致性，按ID排序
	sortedWorkers := make([]*Worker, len(workers))
	copy(sortedWorkers, workers)
	sort.Slice(sortedWorkers, func(i, j int) bool {
		return sortedWorkers[i].ID < sortedWorkers[j].ID
	})

	// 安全的索引计算，避免节点数量变化时的问题
	idx := lb.roundRobin.Next(len(sortedWorkers))
	if idx >= len(sortedWorkers) {
		idx = 0
		lb.roundRobin.Reset() // 重置计数器以避免持续越界
	}
	return sortedWorkers[idx], nil
}

// selectByWeightedRoundRobin 加权轮询策略
func (lb *loadBalancer) selectByWeightedRoundRobin(workers []*Worker) (*Worker, error) {
	if len(workers) == 0 {
		return nil, ErrNoWorkersAvailable
	}

	// 为了一致性，按ID排序
	sortedWorkers := make([]*Worker, len(workers))
	copy(sortedWorkers, workers)
	sort.Slice(sortedWorkers, func(i, j int) bool {
		return sortedWorkers[i].ID < sortedWorkers[j].ID
	})

	weightedWorkers := lb.pool.GetWorkerSlice()
	defer lb.pool.PutWorkerSlice(weightedWorkers)

	// 根据权重创建加权列表
	for _, worker := range sortedWorkers {
		weight := worker.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			weightedWorkers = append(weightedWorkers, worker)
		}
	}

	if len(weightedWorkers) == 0 {
		return nil, ErrNoWorkersAvailable
	}

	// 安全的索引计算
	idx := lb.roundRobin.Next(len(weightedWorkers))
	if idx >= len(weightedWorkers) {
		idx = 0
		lb.roundRobin.Reset()
	}
	return weightedWorkers[idx], nil
}

// selectByRandom 随机策略
func (lb *loadBalancer) selectByRandom(workers []*Worker) (*Worker, error) {
	if len(workers) == 0 {
		return nil, ErrNoWorkersAvailable
	}

	rng := lb.randPool.Get().(*rand.Rand)
	defer lb.randPool.Put(rng)

	idx := rng.Intn(len(workers))
	return workers[idx], nil
}

// selectByConsistentHash 一致性哈希策略
func (lb *loadBalancer) selectByConsistentHash(workers []*Worker, task *Task) (*Worker, error) {
	if len(workers) == 0 {
		return nil, ErrNoWorkersAvailable
	}

	if task == nil || task.ID == "" {
		// 无任务ID时退化为随机选择
		return lb.selectByRandom(workers)
	}

	// 获取或构建哈希环
	ring := lb.getOrBuildHashRing(workers)
	if ring.count == 0 {
		return lb.selectByRandom(workers)
	}

	// 计算任务哈希
	taskHash := lb.fastHash(task.ID)

	// 创建工作节点映射以快速查找
	workerMap := make(map[string]*Worker, len(workers))
	for _, worker := range workers {
		workerMap[worker.ID] = worker
	}

	// 从哈希环中查找，支持多次尝试以处理不可用的节点
	maxAttempts := 3 // 最多尝试3次
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// 每次尝试时微调哈希值，避免总是选择同一个不可用节点
		adjustedHash := taskHash + uint64(attempt*131) // 使用质数增加随机性

		workerID := lb.findNodeInRing(ring, adjustedHash)
		if workerID == "" {
			continue
		}

		// 查找对应的工作节点
		if worker, exists := workerMap[workerID]; exists {
			return worker, nil
		}
	}

	// 如果所有尝试都失败，降级为随机选择
	return lb.selectByRandom(workers)
}

// fastHash 哈希函数
func (lb *loadBalancer) fastHash(s string) uint64 {
	h := sha1.Sum([]byte(s))
	// 使用SHA-1的前8个字节，通过binary包转换为uint64
	return binary.BigEndian.Uint64(h[:8])
}

// getOrBuildHashRing 获取或构建哈希环
func (lb *loadBalancer) getOrBuildHashRing(workers []*Worker) *ConsistentHashRing {
	ring := lb.hashRing.Load().(*ConsistentHashRing)

	// 检查是否需要重建哈希环
	needRebuild := ring.count == 0

	if !needRebuild && len(workers) > 0 {
		// 计算当前工作节点的预期虚拟节点总数
		expectedVirtualNodes := 0
		for _, worker := range workers {
			weight := worker.Weight
			if weight <= 0 {
				weight = 1
			}
			expectedVirtualNodes += lb.virtualNodes * weight
		}

		// 检查虚拟节点总数是否匹配
		if ring.count != expectedVirtualNodes {
			needRebuild = true
		} else {
			// 检查工作节点ID集合是否发生变化
			existingWorkerIDs := make(map[string]bool)
			for _, node := range ring.nodes {
				existingWorkerIDs[node.workerID] = true
			}

			currentWorkerIDs := make(map[string]bool)
			for _, worker := range workers {
				currentWorkerIDs[worker.ID] = true
			}

			// 比较两个集合是否相等
			if len(existingWorkerIDs) != len(currentWorkerIDs) {
				needRebuild = true
			} else {
				for workerID := range currentWorkerIDs {
					if !existingWorkerIDs[workerID] {
						needRebuild = true
						break
					}
				}
			}
		}
	} else if len(workers) == 0 {
		// 没有工作节点时，返回空环
		return &ConsistentHashRing{nodes: make([]HashNode, 0), count: 0}
	}

	if needRebuild {
		ring = lb.buildHashRing(workers)
		lb.hashRing.Store(ring)
	}

	return ring
}

// buildHashRing 构建哈希环
func (lb *loadBalancer) buildHashRing(workers []*Worker) *ConsistentHashRing {
	if len(workers) == 0 {
		return &ConsistentHashRing{nodes: make([]HashNode, 0), count: 0}
	}

	ring := &ConsistentHashRing{
		nodes: make([]HashNode, 0),
	}

	// 每个权重单位对应80个虚拟节点，增加更多虚拟节点以获得更好分布
	replicaCount := 80

	for _, worker := range workers {
		weight := worker.Weight
		if weight <= 0 {
			weight = 1
		}

		// 为每个权重单位创建虚拟节点
		for w := 0; w < weight; w++ {
			for i := 0; i < replicaCount; i++ {
				// 节点ID + 副本编号, 确保良好的分布
				key := fmt.Sprintf("%s:%d:%d", worker.ID, w, i)
				hash := lb.fastHash(key)

				ring.nodes = append(ring.nodes, HashNode{
					hash:     hash,
					workerID: worker.ID,
					weight:   weight,
				})
			}
		}
	}

	// 按哈希值排序所有虚拟节点
	sort.Slice(ring.nodes, func(i, j int) bool {
		return ring.nodes[i].hash < ring.nodes[j].hash
	})

	ring.count = len(ring.nodes)
	return ring
}

// findNodeInRing 在哈希环中查找节点
func (lb *loadBalancer) findNodeInRing(ring *ConsistentHashRing, hash uint64) string {
	if ring.count == 0 {
		return ""
	}

	// 使用二分查找找到第一个哈希值大于等于目标哈希值的节点
	idx := sort.Search(ring.count, func(i int) bool {
		return ring.nodes[i].hash >= hash
	})

	// 如果没有找到，则选择第一个节点（环形结构）
	if idx >= ring.count {
		idx = 0
	}

	return ring.nodes[idx].workerID
}

// handleWorkerSetChange 处理节点集合变化
func (lb *loadBalancer) handleWorkerSetChange(workers []*Worker) {
	// 构建当前工作节点集合
	currentWorkerSet := make(map[string]bool, len(workers))
	for _, worker := range workers {
		currentWorkerSet[worker.ID] = true
	}

	// 获取上次的工作节点集合
	lastWorkerSetInterface := lb.lastWorkerSet.Load()
	if lastWorkerSetInterface == nil {
		// 首次调用，保存当前集合
		lb.lastWorkerSet.Store(currentWorkerSet)
		return
	}

	lastWorkerSet := lastWorkerSetInterface.(map[string]bool)

	// 检查是否有变化
	hasChanges := len(currentWorkerSet) != len(lastWorkerSet)
	if !hasChanges {
		for workerID := range currentWorkerSet {
			if !lastWorkerSet[workerID] {
				hasChanges = true
				break
			}
		}
	}

	if hasChanges {
		// 节点集合发生变化，清理相关状态
		lb.cleanupOnWorkerSetChange(currentWorkerSet, lastWorkerSet)
		lb.lastWorkerSet.Store(currentWorkerSet)
	}
}

// cleanupOnWorkerSetChange 清理节点变化时的状态
func (lb *loadBalancer) cleanupOnWorkerSetChange(currentSet, lastSet map[string]bool) {
	// 找出被移除的节点
	removedWorkers := make([]string, 0)
	for workerID := range lastSet {
		if !currentSet[workerID] {
			removedWorkers = append(removedWorkers, workerID)
		}
	}

	// 清理已移除节点的缓存
	if len(removedWorkers) > 0 && lb.enableCache {
		lb.cache.Lock()
		for _, workerID := range removedWorkers {
			delete(lb.cache.data, workerID)
		}
		lb.cache.Unlock()
	}

	// 重置轮询计数器以确保均匀分布
	lb.roundRobin.Reset()

	// 重置哈希环（在getOrBuildHashRing中会自动重建）
	lb.hashRing.Store(&ConsistentHashRing{})
}

// GetStrategy 获取当前策略
func (lb *loadBalancer) GetStrategy() LoadBalanceStrategy {
	return lb.strategy.Load().(LoadBalanceStrategy)
}

// SetStrategy 设置负载均衡策略
func (lb *loadBalancer) SetStrategy(strategy LoadBalanceStrategy) {
	lb.strategy.Store(strategy)
	lb.Reset()
}

// GetMetrics 获取性能指标
func (lb *loadBalancer) GetMetrics() *LoadBalancerMetrics {
	return &LoadBalancerMetrics{
		SelectionCount:    atomic.LoadUint64(&lb.metrics.SelectionCount),
		SelectionDuration: atomic.LoadUint64(&lb.metrics.SelectionDuration),
		CacheHits:         atomic.LoadUint64(&lb.metrics.CacheHits),
		CacheMisses:       atomic.LoadUint64(&lb.metrics.CacheMisses),
		ErrorCount:        atomic.LoadUint64(&lb.metrics.ErrorCount),
	}
}

// Reset 重置内部状态
func (lb *loadBalancer) Reset() {
	lb.roundRobin.Reset()

	// 清空缓存
	lb.cache.Lock()
	for k := range lb.cache.data {
		delete(lb.cache.data, k)
	}
	lb.cache.Unlock()

	// 重置哈希环
	lb.hashRing.Store(&ConsistentHashRing{})

	// 重置节点追踪
	lb.lastWorkerSet.Store(make(map[string]bool))
}

// Close 关闭负载均衡器
func (lb *loadBalancer) Close() error {
	atomic.StoreInt32(&lb.closed, 1)
	return nil
}

// PlacementStrategy 任务放置策略
type PlacementStrategy struct {
	affinityRules     map[string]string
	antiAffinityRules map[string][]string
	resourceLimits    map[string]float64
}

// NewPlacementStrategy 创建任务放置策略
func NewPlacementStrategy() *PlacementStrategy {
	return &PlacementStrategy{
		affinityRules:     make(map[string]string),
		antiAffinityRules: make(map[string][]string),
		resourceLimits:    make(map[string]float64),
	}
}

// Select 根据放置策略选择工作节点
func (ps *PlacementStrategy) Select(ctx context.Context, workers []*Worker, task *Task) (*Worker, error) {
	// 实现任务亲和性和反亲和性逻辑
	for _, worker := range workers {
		if ps.canPlace(worker, task) {
			return worker, nil
		}
	}
	return nil, ErrNoWorkersAvailable
}

func (ps *PlacementStrategy) Name() string {
	return "placement"
}

func (ps *PlacementStrategy) Reset() {}

func (ps *PlacementStrategy) canPlace(worker *Worker, task *Task) bool {
	// 检查亲和性规则
	if preferred, exists := ps.affinityRules[task.ID]; exists && worker.ID != preferred {
		return false
	}

	// 检查反亲和性规则
	if forbidden, exists := ps.antiAffinityRules[task.ID]; exists {
		for _, forbiddenWorker := range forbidden {
			if worker.ID == forbiddenWorker {
				return false
			}
		}
	}

	// 检查资源限制
	for resource, limit := range ps.resourceLimits {
		switch resource {
		case "cpu":
			if worker.HealthStatus.CPU > limit {
				return false
			}
		case "memory":
			if worker.HealthStatus.Memory > limit {
				return false
			}
		}
	}

	return true
}
