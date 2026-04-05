package async

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_SubmitAndExecute(t *testing.T) {
	p := NewPool("test", 2, 10)
	defer p.Shutdown(context.Background())

	var executed atomic.Bool
	err := p.Submit(func(ctx context.Context) {
		executed.Store(true)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if !executed.Load() {
		t.Error("task was not executed")
	}
}

func TestPool_BoundedQueue(t *testing.T) {
	p := NewPool("test-bounded", 1, 2)
	defer p.Shutdown(context.Background())

	block := make(chan struct{})
	done := make(chan struct{})

	err := p.Submit(func(ctx context.Context) {
		<-block
		close(done)
	})
	if err != nil {
		t.Fatalf("first submit failed: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	err = p.Submit(func(ctx context.Context) {})
	if err != nil {
		t.Fatalf("second submit failed: %v", err)
	}
	err = p.Submit(func(ctx context.Context) {})
	if err != nil {
		t.Fatalf("third submit failed: %v", err)
	}

	err = p.Submit(func(ctx context.Context) {})
	if err == nil {
		t.Error("expected rejection when queue is full")
	}

	close(block)
	<-done
}

func TestPool_PanicRecovery(t *testing.T) {
	p := NewPool("test-panic", 1, 10)
	defer p.Shutdown(context.Background())

	err := p.Submit(func(ctx context.Context) {
		panic("test panic")
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	stats := p.Stats()
	if stats.Panicked != 1 {
		t.Errorf("expected 1 panic, got %d", stats.Panicked)
	}

	err = p.Submit(func(ctx context.Context) {})
	if err != nil {
		t.Fatalf("pool should still accept tasks after panic: %v", err)
	}
}

func TestPool_Shutdown(t *testing.T) {
	p := NewPool("test-shutdown", 2, 10)

	var count atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		_ = p.Submit(func(ctx context.Context) {
			count.Add(1)
			wg.Done()
		})
	}

	wg.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p.Shutdown(ctx)

	stats := p.Stats()
	if stats.Completed != 5 {
		t.Errorf("expected 5 completed, got %d", stats.Completed)
	}
}

func TestPool_Stats(t *testing.T) {
	p := NewPool("test-stats", 1, 5)
	defer p.Shutdown(context.Background())

	for i := 0; i < 3; i++ {
		_ = p.Submit(func(ctx context.Context) {})
	}

	time.Sleep(50 * time.Millisecond)

	stats := p.Stats()
	if stats.Name != "test-stats" {
		t.Errorf("expected name=test-stats, got %s", stats.Name)
	}
	if stats.Submitted != 3 {
		t.Errorf("expected 3 submitted, got %d", stats.Submitted)
	}
	if stats.Completed != 3 {
		t.Errorf("expected 3 completed, got %d", stats.Completed)
	}
	if stats.Rejected != 0 {
		t.Errorf("expected 0 rejected, got %d", stats.Rejected)
	}
}

func TestRuntime_AddAndGetPool(t *testing.T) {
	r := NewRuntime()
	defer r.Shutdown(context.Background())

	p := r.AddPool("my-pool", 2, 10)
	if p == nil {
		t.Fatal("expected pool to be created")
	}

	got := r.GetPool("my-pool")
	if got == nil {
		t.Fatal("expected to retrieve pool by name")
	}
	if got != p {
		t.Error("retrieved pool does not match created pool")
	}
}

func TestRuntime_ShutdownAll(t *testing.T) {
	r := NewRuntime()

	r.AddPool("pool-a", 1, 5)
	r.AddPool("pool-b", 1, 5)

	var wg sync.WaitGroup
	wg.Add(2)
	_ = r.GetPool("pool-a").Submit(func(ctx context.Context) {
		wg.Done()
	})
	_ = r.GetPool("pool-b").Submit(func(ctx context.Context) {
		wg.Done()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r.Shutdown(ctx)

	wg.Wait()
}

func TestRuntime_Stats(t *testing.T) {
	r := NewRuntime()
	defer r.Shutdown(context.Background())

	r.AddPool("stats-a", 1, 5)
	r.AddPool("stats-b", 1, 5)

	_ = r.GetPool("stats-a").Submit(func(ctx context.Context) {})
	_ = r.GetPool("stats-b").Submit(func(ctx context.Context) {})

	time.Sleep(50 * time.Millisecond)

	stats := r.Stats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 pool stats, got %d", len(stats))
	}
}
