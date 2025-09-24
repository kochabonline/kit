package redis

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
)

var (
	ErrNoWorkersAvailable = errors.New("no workers available")
	ErrInvalidStrategy    = errors.New("invalid load balance strategy")
	ErrWorkerNotHealthy   = errors.New("worker not healthy")
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	// SelectWorker 根据策略选择最佳工作节点
	SelectWorker(ctx context.Context, workers []*Worker, task *Task) (*Worker, error)

	// GetStrategy 获取当前使用的策略
	GetStrategy() LoadBalanceStrategy

	// SetStrategy 设置负载均衡策略
	SetStrategy(strategy LoadBalanceStrategy)

	// Reset 重置内部状态（如轮询计数器等）
	Reset()
}

// TaskCountProvider 提供工作节点任务计数的接口
type TaskCountProvider interface {
	GetWorkerRunningTaskCount(ctx context.Context, workerID string) (int, error)
}

// redisLoadBalancer Redis负载均衡器实现
type redisLoadBalancer struct {
	strategy      LoadBalanceStrategy
	countProvider TaskCountProvider

	// 轮询状态
	roundRobinIndex int64

	// 一致性哈希环
	hashRing *ConsistentHashRing

	// 随机数生成器
	rand *rand.Rand
	mu   sync.Mutex
}

// NewRedisLoadBalancer 创建Redis负载均衡器
func NewRedisLoadBalancer(strategy LoadBalanceStrategy, countProvider TaskCountProvider) LoadBalancer {
	return &redisLoadBalancer{
		strategy:      strategy,
		countProvider: countProvider,
		hashRing:      NewConsistentHashRing(),
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SelectWorker 选择工作节点
func (lb *redisLoadBalancer) SelectWorker(ctx context.Context, workers []*Worker, task *Task) (*Worker, error) {
	if len(workers) == 0 {
		return nil, ErrNoWorkersAvailable
	}

	// 过滤可用工作节点
	availableWorkers := lb.filterAvailableWorkers(workers)
	if len(availableWorkers) == 0 {
		return nil, ErrNoWorkersAvailable
	}

	switch lb.strategy {
	case StrategyLeastTasks:
		return lb.selectByLeastTasks(ctx, availableWorkers)
	case StrategyRoundRobin:
		return lb.selectByRoundRobin(availableWorkers)
	case StrategyWeightedRoundRobin:
		return lb.selectByWeightedRoundRobin(availableWorkers)
	case StrategyRandom:
		return lb.selectByRandom(availableWorkers)
	case StrategyConsistentHash:
		return lb.selectByConsistentHash(availableWorkers, task)
	case StrategyLeastConnections:
		return lb.selectByLeastConnections(ctx, availableWorkers)
	default:
		return nil, ErrInvalidStrategy
	}
}

// GetStrategy 获取当前策略
func (lb *redisLoadBalancer) GetStrategy() LoadBalanceStrategy {
	return lb.strategy
}

// SetStrategy 设置策略
func (lb *redisLoadBalancer) SetStrategy(strategy LoadBalanceStrategy) {
	lb.strategy = strategy
	if strategy == StrategyConsistentHash {
		// 重建一致性哈希环
		lb.rebuildHashRing(nil) // 这里需要传入workers，实际使用时需要调整
	}
}

// Reset 重置内部状态
func (lb *redisLoadBalancer) Reset() {
	atomic.StoreInt64(&lb.roundRobinIndex, 0)
	lb.hashRing = NewConsistentHashRing()
}

// filterAvailableWorkers 过滤可用的工作节点
func (lb *redisLoadBalancer) filterAvailableWorkers(workers []*Worker) []*Worker {
	var available []*Worker

	for _, worker := range workers {
		// 检查工作节点是否在线且未满负荷
		if worker.Status == WorkerStatusOnline && worker.CurrentTasks < worker.MaxTasks {
			available = append(available, worker)
		}
	}

	return available
}

// selectByLeastTasks 按最少任务数选择
func (lb *redisLoadBalancer) selectByLeastTasks(ctx context.Context, workers []*Worker) (*Worker, error) {
	if len(workers) == 1 {
		return workers[0], nil
	}

	var bestWorker *Worker
	minTasks := math.MaxInt32

	for _, worker := range workers {
		// 获取实时任务数
		currentTasks := worker.CurrentTasks
		if lb.countProvider != nil {
			if count, err := lb.countProvider.GetWorkerRunningTaskCount(ctx, worker.ID); err == nil {
				currentTasks = count
			}
		}

		if currentTasks < minTasks {
			minTasks = currentTasks
			bestWorker = worker
		}
	}

	return bestWorker, nil
}

// selectByRoundRobin 轮询选择
func (lb *redisLoadBalancer) selectByRoundRobin(workers []*Worker) (*Worker, error) {
	index := atomic.AddInt64(&lb.roundRobinIndex, 1) - 1
	return workers[index%int64(len(workers))], nil
}

// selectByWeightedRoundRobin 加权轮询选择
func (lb *redisLoadBalancer) selectByWeightedRoundRobin(workers []*Worker) (*Worker, error) {
	totalWeight := 0
	for _, worker := range workers {
		if worker.Weight <= 0 {
			worker.Weight = 1 // 默认权重
		}
		totalWeight += worker.Weight
	}

	lb.mu.Lock()
	target := lb.rand.Intn(totalWeight)
	lb.mu.Unlock()

	currentWeight := 0
	for _, worker := range workers {
		currentWeight += worker.Weight
		if currentWeight > target {
			return worker, nil
		}
	}

	return workers[len(workers)-1], nil
}

// selectByRandom 随机选择
func (lb *redisLoadBalancer) selectByRandom(workers []*Worker) (*Worker, error) {
	lb.mu.Lock()
	index := lb.rand.Intn(len(workers))
	lb.mu.Unlock()

	return workers[index], nil
}

// selectByConsistentHash 一致性哈希选择
func (lb *redisLoadBalancer) selectByConsistentHash(workers []*Worker, task *Task) (*Worker, error) {
	// 重建哈希环（如果需要）
	lb.rebuildHashRing(workers)

	// 使用任务ID作为哈希键
	hashKey := task.ID
	if hashKey == "" {
		// 如果任务ID为空，使用任务名称
		hashKey = task.Name
	}

	workerID := lb.hashRing.Get(hashKey)

	// 找到对应的工作节点
	for _, worker := range workers {
		if worker.ID == workerID {
			return worker, nil
		}
	}

	// 如果没找到，回退到轮询策略
	return lb.selectByRoundRobin(workers)
}

// selectByLeastConnections 按最少连接数选择
func (lb *redisLoadBalancer) selectByLeastConnections(ctx context.Context, workers []*Worker) (*Worker, error) {
	// 对于任务调度器，连接数等同于当前任务数
	return lb.selectByLeastTasks(ctx, workers)
}

// rebuildHashRing 重建一致性哈希环
func (lb *redisLoadBalancer) rebuildHashRing(workers []*Worker) {
	if workers == nil {
		return
	}

	lb.hashRing = NewConsistentHashRing()
	for _, worker := range workers {
		weight := worker.Weight
		if weight <= 0 {
			weight = 1
		}
		lb.hashRing.Add(worker.ID, weight)
	}
}

// ConsistentHashRing 一致性哈希环
type ConsistentHashRing struct {
	nodes   []HashNode
	nodeMap map[string]bool
	mu      sync.RWMutex
}

// HashNode 哈希节点
type HashNode struct {
	Hash   uint64
	NodeID string
}

// NewConsistentHashRing 创建一致性哈希环
func NewConsistentHashRing() *ConsistentHashRing {
	return &ConsistentHashRing{
		nodes:   make([]HashNode, 0),
		nodeMap: make(map[string]bool),
	}
}

// Add 添加节点到哈希环
func (c *ConsistentHashRing) Add(nodeID string, weight int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.nodeMap[nodeID] {
		return // 节点已存在
	}

	virtualNodes := weight * 150 // 每个权重对应150个虚拟节点
	for i := 0; i < virtualNodes; i++ {
		hash := c.hashKey(fmt.Sprintf("%s:%d", nodeID, i))
		node := HashNode{
			Hash:   hash,
			NodeID: nodeID,
		}
		c.nodes = append(c.nodes, node)
	}

	c.nodeMap[nodeID] = true
	sort.Slice(c.nodes, func(i, j int) bool {
		return c.nodes[i].Hash < c.nodes[j].Hash
	})
}

// Remove 从哈希环移除节点
func (c *ConsistentHashRing) Remove(nodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.nodeMap[nodeID] {
		return // 节点不存在
	}

	var newNodes []HashNode
	for _, node := range c.nodes {
		if node.NodeID != nodeID {
			newNodes = append(newNodes, node)
		}
	}

	c.nodes = newNodes
	delete(c.nodeMap, nodeID)
}

// Get 根据key获取节点
func (c *ConsistentHashRing) Get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.nodes) == 0 {
		return ""
	}

	hash := c.hashKey(key)

	// 二分查找第一个hash值大于等于key hash的节点
	idx := sort.Search(len(c.nodes), func(i int) bool {
		return c.nodes[i].Hash >= hash
	})

	// 如果没找到，选择第一个节点（环形特性）
	if idx == len(c.nodes) {
		idx = 0
	}

	return c.nodes[idx].NodeID
}

// hashKey 计算key的哈希值
func (c *ConsistentHashRing) hashKey(key string) uint64 {
	h := sha1.New()
	h.Write([]byte(key))
	hash := h.Sum(nil)

	// 取前8字节转换为uint64
	return binary.BigEndian.Uint64(hash[:8])
}

// IsEmpty 检查哈希环是否为空
func (c *ConsistentHashRing) IsEmpty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.nodes) == 0
}

// NodeCount 获取节点数量
func (c *ConsistentHashRing) NodeCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.nodeMap)
}

// redisTaskCountProvider Redis任务计数提供者实现
type redisTaskCountProvider struct {
	workerManager *WorkerManager
}

// NewRedisTaskCountProvider 创建Redis任务计数提供者
func NewRedisTaskCountProvider(workerManager *WorkerManager) TaskCountProvider {
	return &redisTaskCountProvider{
		workerManager: workerManager,
	}
}

// GetWorkerRunningTaskCount 获取工作节点运行任务数
func (p *redisTaskCountProvider) GetWorkerRunningTaskCount(ctx context.Context, workerID string) (int, error) {
	worker, err := p.workerManager.GetWorker(ctx, workerID)
	if err != nil {
		return 0, err
	}

	return worker.CurrentTasks, nil
}
