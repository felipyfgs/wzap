package async

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"wzap/internal/logger"
)

type Task func(ctx context.Context)

type Pool struct {
	name      string
	workers   int
	queue     chan Task
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	submitted atomic.Int64
	rejected  atomic.Int64
	completed atomic.Int64
	panicked  atomic.Int64
}

func NewPool(name string, workers int, queueSize int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		name:    name,
		workers: workers,
		queue:   make(chan Task, queueSize),
		ctx:     ctx,
		cancel:  cancel,
	}
	for range workers {
		p.wg.Add(1)
		go p.worker()
	}
	logger.Info().Str("pool", name).Int("workers", workers).Int("queue", queueSize).Msg("async pool started")
	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.queue:
			if !ok {
				return
			}
			p.execute(task)
		}
	}
}

func (p *Pool) execute(task Task) {
	defer func() {
		if r := recover(); r != nil {
			p.panicked.Add(1)
			logger.Error().
				Str("pool", p.name).
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("async pool task panicked")
		}
	}()

	task(p.ctx)
	p.completed.Add(1)
}

func (p *Pool) Submit(task Task) error {
	select {
	case p.queue <- task:
		p.submitted.Add(1)
		return nil
	default:
		p.rejected.Add(1)
		logger.Warn().Str("pool", p.name).Msg("async pool queue full, task rejected")
		return fmt.Errorf("pool %s: queue full", p.name)
	}
}

func (p *Pool) Shutdown(ctx context.Context) {
	p.cancel()
	close(p.queue)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info().Str("pool", p.name).Msg("async pool drained")
	case <-ctx.Done():
		logger.Warn().Str("pool", p.name).Msg("async pool shutdown timed out")
	}
}

func (p *Pool) Stats() PoolStats {
	return PoolStats{
		Name:      p.name,
		Submitted: p.submitted.Load(),
		Completed: p.completed.Load(),
		Rejected:  p.rejected.Load(),
		Panicked:  p.panicked.Load(),
		Queued:    int64(len(p.queue)),
	}
}

type PoolStats struct {
	Name      string `json:"name"`
	Submitted int64  `json:"submitted"`
	Completed int64  `json:"completed"`
	Rejected  int64  `json:"rejected"`
	Panicked  int64  `json:"panicked"`
	Queued    int64  `json:"queued"`
}

type Runtime struct {
	pools []*Pool
	mu    sync.RWMutex
}

func NewRuntime() *Runtime {
	return &Runtime{}
}

func (r *Runtime) AddPool(name string, workers int, queueSize int) *Pool {
	p := NewPool(name, workers, queueSize)
	r.mu.Lock()
	r.pools = append(r.pools, p)
	r.mu.Unlock()
	return p
}

func (r *Runtime) GetPool(name string) *Pool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.pools {
		if p.name == name {
			return p
		}
	}
	return nil
}

func (r *Runtime) Shutdown(ctx context.Context) {
	r.mu.RLock()
	pools := make([]*Pool, len(r.pools))
	copy(pools, r.pools)
	r.mu.RUnlock()

	for _, p := range pools {
		p.Shutdown(ctx)
	}
}

func (r *Runtime) Stats() []PoolStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stats := make([]PoolStats, 0, len(r.pools))
	for _, p := range r.pools {
		stats = append(stats, p.Stats())
	}
	return stats
}
