package chatwoot

import (
	"sync"
	"time"
)

type cbState int

const (
	cbClosed   cbState = 0
	cbOpen     cbState = 1
	cbHalfOpen cbState = 2

	cbThreshold = 5
	cbTimeout   = 30 * time.Second
)

type circuitBreaker struct {
	mu       sync.Mutex
	state    cbState
	failures int
	lastFail time.Time
}

func newCircuitBreaker() *circuitBreaker {
	return &circuitBreaker{state: cbClosed}
}

func (cb *circuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbClosed:
		return true
	case cbOpen:
		if time.Since(cb.lastFail) > cbTimeout {
			cb.state = cbHalfOpen
			return true
		}
		return false
	case cbHalfOpen:
		return true
	}
	return true
}

func (cb *circuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = cbClosed
	cb.failures = 0
}

func (cb *circuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFail = time.Now()
	if cb.state == cbHalfOpen || cb.failures >= cbThreshold {
		cb.state = cbOpen
	}
}

func (cb *circuitBreaker) State() cbState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

type cbManager struct {
	mu  sync.RWMutex
	cbs map[string]*circuitBreaker
}

func newCircuitBreakerManager() *cbManager {
	return &cbManager{cbs: make(map[string]*circuitBreaker)}
}

func (m *cbManager) get(sessionID string) *circuitBreaker {
	m.mu.RLock()
	cb, ok := m.cbs[sessionID]
	m.mu.RUnlock()
	if ok {
		return cb
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if cb, ok = m.cbs[sessionID]; ok {
		return cb
	}
	cb = newCircuitBreaker()
	m.cbs[sessionID] = cb
	return cb
}

func (m *cbManager) Allow(sessionID string) bool {
	return m.get(sessionID).Allow()
}

func (m *cbManager) RecordSuccess(sessionID string) {
	m.get(sessionID).RecordSuccess()
}

func (m *cbManager) RecordFailure(sessionID string) {
	m.get(sessionID).RecordFailure()
}
