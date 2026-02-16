package middleware

import (
	"sync"
	"time"

	"github.com/krustd/nexus-gateway/config"
)

type circuitState int

const (
	stateClosed   circuitState = iota // 正常，请求通过
	stateOpen                         // 熔断，快速失败
	stateHalfOpen                     // 半开，允许探测请求
)

type serviceCircuit struct {
	mu       sync.Mutex
	state    circuitState
	failures int
	total    int
	openedAt time.Time

	// 滑动窗口
	windowStart time.Time
}

// CircuitBreakerManager 按服务名管理熔断状态
type CircuitBreakerManager struct {
	mu       sync.RWMutex
	circuits map[string]*serviceCircuit
	cfg      config.CircuitConfig
}

func NewCircuitBreakerManager(cfg config.CircuitConfig) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		circuits: make(map[string]*serviceCircuit),
		cfg:      cfg,
	}
}

func (m *CircuitBreakerManager) getCircuit(serviceName string) *serviceCircuit {
	m.mu.RLock()
	sc, ok := m.circuits[serviceName]
	m.mu.RUnlock()
	if ok {
		return sc
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if sc, ok := m.circuits[serviceName]; ok {
		return sc
	}
	sc = &serviceCircuit{
		state:       stateClosed,
		windowStart: time.Now(),
	}
	m.circuits[serviceName] = sc
	return sc
}

// Allow 判断是否允许请求通过
func (m *CircuitBreakerManager) Allow(serviceName string) bool {
	sc := m.getCircuit(serviceName)
	sc.mu.Lock()
	defer sc.mu.Unlock()

	switch sc.state {
	case stateClosed:
		return true

	case stateOpen:
		cooldown := time.Duration(m.cfg.CooldownSec) * time.Second
		if time.Since(sc.openedAt) >= cooldown {
			// 冷却结束，进入半开状态
			sc.state = stateHalfOpen
			sc.failures = 0
			sc.total = 0
			sc.windowStart = time.Now()
			return true
		}
		return false

	case stateHalfOpen:
		// 半开状态允许有限请求通过探测
		return true
	}

	return true
}

// RecordSuccess 记录成功请求
func (m *CircuitBreakerManager) RecordSuccess(serviceName string) {
	sc := m.getCircuit(serviceName)
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// 重置过期窗口（与 RecordFailure 保持一致）
	window := time.Duration(m.cfg.WindowSec) * time.Second
	if time.Since(sc.windowStart) > window {
		sc.failures = 0
		sc.total = 0
		sc.windowStart = time.Now()
	}

	sc.total++

	if sc.state == stateHalfOpen {
		// 半开状态收到成功 → 恢复关闭
		sc.state = stateClosed
		sc.failures = 0
		sc.total = 0
		sc.windowStart = time.Now()
	}
}

// RecordFailure 记录失败请求
func (m *CircuitBreakerManager) RecordFailure(serviceName string) {
	sc := m.getCircuit(serviceName)
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// 重置过期窗口
	window := time.Duration(m.cfg.WindowSec) * time.Second
	if time.Since(sc.windowStart) > window {
		sc.failures = 0
		sc.total = 0
		sc.windowStart = time.Now()
	}

	sc.failures++
	sc.total++

	if sc.state == stateHalfOpen {
		// 半开状态收到失败 → 重新熔断
		sc.state = stateOpen
		sc.openedAt = time.Now()
		return
	}

	// 关闭状态：检查是否需要熔断
	if sc.total >= m.cfg.MinRequests {
		errorRatio := float64(sc.failures) / float64(sc.total)
		if errorRatio >= m.cfg.ErrorThreshold {
			sc.state = stateOpen
			sc.openedAt = time.Now()
		}
	}
}

// GetState 获取服务的熔断状态（用于指标上报）
func (m *CircuitBreakerManager) GetState(serviceName string) circuitState {
	sc := m.getCircuit(serviceName)
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.state
}
