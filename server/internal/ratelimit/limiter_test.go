package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	limiter := New(time.Second)
	if limiter == nil {
		t.Fatal("New() returned nil")
	}
	if limiter.hosts == nil {
		t.Fatal("New() returned limiter with nil hosts map")
	}
	if limiter.minInterval != time.Second {
		t.Errorf("New() minInterval = %v, want %v", limiter.minInterval, time.Second)
	}
}

func TestAllow_FirstRequest(t *testing.T) {
	limiter := New(100 * time.Millisecond)

	if !limiter.Allow("example.com") {
		t.Error("Allow() should return true for first request to a host")
	}
}

func TestAllow_SecondRequestTooSoon(t *testing.T) {
	limiter := New(100 * time.Millisecond)

	limiter.Allow("example.com")
	if limiter.Allow("example.com") {
		t.Error("Allow() should return false for second request before minInterval")
	}
}

func TestAllow_SecondRequestAfterInterval(t *testing.T) {
	limiter := New(50 * time.Millisecond)

	limiter.Allow("example.com")
	time.Sleep(60 * time.Millisecond)

	if !limiter.Allow("example.com") {
		t.Error("Allow() should return true after minInterval has passed")
	}
}

func TestAllow_DifferentHosts(t *testing.T) {
	limiter := New(100 * time.Millisecond)

	limiter.Allow("example.com")
	if !limiter.Allow("other.com") {
		t.Error("Allow() should return true for different host")
	}
}

func TestWait_FirstRequest(t *testing.T) {
	limiter := New(50 * time.Millisecond)

	start := time.Now()
	limiter.Wait("example.com")
	elapsed := time.Since(start)

	if elapsed >= 50*time.Millisecond {
		t.Error("Wait() should not wait for first request")
	}
}

func TestWait_SecondRequestWaits(t *testing.T) {
	limiter := New(50 * time.Millisecond)

	limiter.Wait("example.com")
	start := time.Now()
	limiter.Wait("example.com")
	elapsed := time.Since(start)

	// Should wait close to 50ms (allow some tolerance)
	if elapsed < 40*time.Millisecond {
		t.Errorf("Wait() should wait for minInterval, elapsed: %v", elapsed)
	}
}

func TestWait_DifferentHostsNoWait(t *testing.T) {
	limiter := New(100 * time.Millisecond)

	limiter.Wait("example.com")
	start := time.Now()
	limiter.Wait("other.com")
	elapsed := time.Since(start)

	if elapsed >= 50*time.Millisecond {
		t.Error("Wait() should not wait for different host")
	}
}

func TestReset(t *testing.T) {
	limiter := New(100 * time.Millisecond)

	limiter.Allow("example.com")
	if limiter.Allow("example.com") {
		t.Fatal("Second Allow() should return false before reset")
	}

	limiter.Reset("example.com")

	if !limiter.Allow("example.com") {
		t.Error("Allow() should return true after Reset()")
	}
}

func TestResetAll(t *testing.T) {
	limiter := New(100 * time.Millisecond)

	limiter.Allow("example.com")
	limiter.Allow("other.com")

	limiter.ResetAll()

	if !limiter.Allow("example.com") {
		t.Error("Allow() should return true after ResetAll()")
	}
	if !limiter.Allow("other.com") {
		t.Error("Allow() should return true after ResetAll()")
	}
}

func TestConcurrentAccess(t *testing.T) {
	limiter := New(10 * time.Millisecond)
	var wg sync.WaitGroup

	// Spawn multiple goroutines accessing the same host
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				limiter.Allow("example.com")
				limiter.Reset("example.com")
			}
		}()
	}

	// Spawn multiple goroutines accessing different hosts
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			host := "host" + string(rune('a'+idx)) + ".com"
			limiter.Wait(host)
		}(i)
	}

	wg.Wait()
	// If we get here without race conditions, test passes
}

func TestWait_PartialIntervalElapsed(t *testing.T) {
	limiter := New(100 * time.Millisecond)

	limiter.Wait("example.com")
	time.Sleep(30 * time.Millisecond) // Wait part of the interval

	start := time.Now()
	limiter.Wait("example.com")
	elapsed := time.Since(start)

	// Should wait approximately 70ms (100ms - 30ms already elapsed)
	if elapsed < 60*time.Millisecond || elapsed > 90*time.Millisecond {
		t.Errorf("Wait() should wait for remaining interval, elapsed: %v", elapsed)
	}
}

func TestAllow_UpdatesTimestamp(t *testing.T) {
	limiter := New(50 * time.Millisecond)

	limiter.Allow("example.com")
	time.Sleep(30 * time.Millisecond)
	limiter.Allow("example.com") // Should fail but not update timestamp

	time.Sleep(30 * time.Millisecond) // 60ms total from first Allow

	if !limiter.Allow("example.com") {
		t.Error("Allow() should return true after original minInterval has passed")
	}
}

func TestReset_NonExistentHost(t *testing.T) {
	limiter := New(time.Second)

	// Should not panic
	limiter.Reset("nonexistent.com")

	// And Allow should work normally
	if !limiter.Allow("nonexistent.com") {
		t.Error("Allow() should return true for host after Reset()")
	}
}

func TestLimiter_ZeroInterval(t *testing.T) {
	limiter := New(0)

	// All requests should be allowed immediately
	for i := 0; i < 10; i++ {
		if !limiter.Allow("example.com") {
			t.Errorf("Allow() should always return true with zero interval, iteration %d", i)
		}
	}
}
