package balancer

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/krustd/nexus-sdk/registry"
)

// ErrNoInstance 没有可用实例
var ErrNoInstance = fmt.Errorf("nexus: no available instance")

// ==================== Round Robin ====================

type roundRobin struct {
	counter uint64
}

// NewRoundRobin 轮询
func NewRoundRobin() registry.Picker {
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

// ==================== Random ====================

type random struct{}

// NewRandom 随机
func NewRandom() registry.Picker {
	return &random{}
}

func (r *random) Pick(instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
	n := len(instances)
	if n == 0 {
		return nil, ErrNoInstance
	}
	return instances[rand.Intn(n)], nil
}

// ==================== Weighted Round Robin ====================
// Nginx 平滑加权轮询算法

type weightedNode struct {
	instance      *registry.ServiceInstance
	weight        int
	currentWeight int
}

type weightedRoundRobin struct {
	mu          sync.Mutex
	nodes       []*weightedNode
	fingerprint string
}

// NewWeightedRoundRobin 加权轮询
func NewWeightedRoundRobin() registry.Picker {
	return &weightedRoundRobin{}
}

func (w *weightedRoundRobin) Pick(instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
	n := len(instances)
	if n == 0 {
		return nil, ErrNoInstance
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	fp := fingerprint(instances)
	if fp != w.fingerprint {
		w.rebuild(instances)
		w.fingerprint = fp
	}

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
		wt := inst.Weight
		if wt <= 0 {
			wt = 1
		}
		w.nodes[i] = &weightedNode{instance: inst, weight: wt}
	}
}

func fingerprint(instances []*registry.ServiceInstance) string {
	s := ""
	for _, inst := range instances {
		s += fmt.Sprintf("%s:%d,", inst.ID, inst.Weight)
	}
	return s
}
