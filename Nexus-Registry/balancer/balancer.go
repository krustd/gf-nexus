package balancer

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"

	registry "github.com/krustd/nexus-registry"
)

// Balancer 负载均衡器接口
type Balancer interface {
	// Pick 从实例列表中选一个
	Pick(instances []*registry.ServiceInstance) (*registry.ServiceInstance, error)
}

// ErrNoInstance 没有可用实例
var ErrNoInstance = fmt.Errorf("nexus-registry: no available instance")

// ---------- Round Robin ----------

type roundRobin struct {
	counter uint64
}

// NewRoundRobin 创建轮询负载均衡器
func NewRoundRobin() Balancer {
	return &roundRobin{}
}

func (rr *roundRobin) Pick(instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
	n := len(instances)
	if n == 0 {
		return nil, ErrNoInstance
	}
	idx := atomic.AddUint64(&rr.counter, 1) - 1
	return instances[idx%uint64(n)], nil
}

// ---------- Random ----------

type random struct{}

// NewRandom 创建随机负载均衡器
func NewRandom() Balancer {
	return &random{}
}

func (r *random) Pick(instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
	n := len(instances)
	if n == 0 {
		return nil, ErrNoInstance
	}
	return instances[rand.Intn(n)], nil
}

// ---------- Weighted Round Robin (平滑加权轮询) ----------

type weightedNode struct {
	instance      *registry.ServiceInstance
	weight        int // 原始权重
	currentWeight int // 当前权重（动态变化）
}

type weightedRoundRobin struct {
	mu sync.Mutex
	// 缓存上一次的实例列表指纹，如果列表变了就重建 nodes
	nodes       []*weightedNode
	fingerprint string
}

// NewWeightedRoundRobin 创建平滑加权轮询负载均衡器
// 基于 Nginx 的 smooth weighted round-robin 算法
func NewWeightedRoundRobin() Balancer {
	return &weightedRoundRobin{}
}

func (w *weightedRoundRobin) Pick(instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
	n := len(instances)
	if n == 0 {
		return nil, ErrNoInstance
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查实例列表是否有变化，有变化则重建
	fp := buildFingerprint(instances)
	if fp != w.fingerprint {
		w.rebuild(instances)
		w.fingerprint = fp
	}

	// Nginx smooth weighted round-robin 算法：
	// 1. 每个节点 currentWeight += weight
	// 2. 选 currentWeight 最大的节点
	// 3. 被选中的节点 currentWeight -= totalWeight

	var (
		totalWeight int
		best        *weightedNode
	)

	for _, node := range w.nodes {
		node.currentWeight += node.weight
		totalWeight += node.weight

		if best == nil || node.currentWeight > best.currentWeight {
			best = node
		}
	}

	if best == nil {
		return nil, ErrNoInstance
	}

	best.currentWeight -= totalWeight
	return best.instance, nil
}

func (w *weightedRoundRobin) rebuild(instances []*registry.ServiceInstance) {
	w.nodes = make([]*weightedNode, len(instances))
	for i, inst := range instances {
		weight := inst.Weight
		if weight <= 0 {
			weight = 1
		}
		w.nodes[i] = &weightedNode{
			instance:      inst,
			weight:        weight,
			currentWeight: 0,
		}
	}
}

// buildFingerprint 用实例 ID 列表构建指纹，检测实例列表变化
func buildFingerprint(instances []*registry.ServiceInstance) string {
	s := ""
	for _, inst := range instances {
		s += inst.ID + ":" + fmt.Sprintf("%d", inst.Weight) + ","
	}
	return s
}
