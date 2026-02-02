package cache

import (
	"sync"
	"testing"
	"time"
)

func TestNewMemory(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	if c == nil {
		t.Fatal("NewMemory() returned nil")
	}
	if c.items == nil {
		t.Fatal("NewMemory() returned cache with nil items map")
	}
	if c.ttl != time.Minute {
		t.Errorf("NewMemory() ttl = %v, want %v", c.ttl, time.Minute)
	}
}

func TestNew(t *testing.T) {
	c := New(time.Minute)
	defer c.Stop()

	if c == nil {
		t.Fatal("New() returned nil")
	}
}

func TestMemoryCache_SetAndGet(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	c.Set("key1", "value1")

	got, ok := c.Get("key1")
	if !ok {
		t.Error("Get() returned false for existing key")
	}
	if got != "value1" {
		t.Errorf("Get() = %v, want %v", got, "value1")
	}
}

func TestMemoryCache_Get_NotFound(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	got, ok := c.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for non-existent key")
	}
	if got != nil {
		t.Errorf("Get() should return nil for non-existent key, got %v", got)
	}
}

func TestMemoryCache_Get_Expired(t *testing.T) {
	c := NewMemory(50 * time.Millisecond)
	defer c.Stop()

	c.Set("key1", "value1")

	if _, ok := c.Get("key1"); !ok {
		t.Error("Get() should return true for fresh key")
	}

	time.Sleep(60 * time.Millisecond)

	got, ok := c.Get("key1")
	if ok {
		t.Error("Get() should return false for expired key")
	}
	if got != nil {
		t.Errorf("Get() should return nil for expired key, got %v", got)
	}
}

func TestMemoryCache_SetWithTTL(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	c.SetWithTTL("key1", "value1", 50*time.Millisecond)

	if _, ok := c.Get("key1"); !ok {
		t.Error("Get() should return true for fresh key")
	}

	time.Sleep(60 * time.Millisecond)

	if _, ok := c.Get("key1"); ok {
		t.Error("Get() should return false after custom TTL expired")
	}
}

func TestMemoryCache_SetWithTTL_LongerThanDefault(t *testing.T) {
	c := NewMemory(50 * time.Millisecond)
	defer c.Stop()

	c.SetWithTTL("key1", "value1", time.Minute)

	time.Sleep(60 * time.Millisecond)

	if _, ok := c.Get("key1"); !ok {
		t.Error("Get() should return true when custom TTL hasn't expired")
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	c.Set("key1", "value1")
	c.Delete("key1")

	if _, ok := c.Get("key1"); ok {
		t.Error("Get() should return false after Delete()")
	}
}

func TestMemoryCache_Delete_NonExistent(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	c.Delete("nonexistent")
}

func TestMemoryCache_Invalidate(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	c.Set("key1", "value1")
	c.Invalidate("key1")

	if _, ok := c.Get("key1"); ok {
		t.Error("Get() should return false after Invalidate()")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	c.Clear()

	for _, key := range []string{"key1", "key2", "key3"} {
		if _, ok := c.Get(key); ok {
			t.Errorf("Get(%q) should return false after Clear()", key)
		}
	}
}

func TestMemoryCache_DifferentValueTypes(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	c.Set("string", "hello")
	if got, ok := c.Get("string"); !ok || got != "hello" {
		t.Error("Failed to store/retrieve string")
	}

	c.Set("int", 42)
	if got, ok := c.Get("int"); !ok || got != 42 {
		t.Error("Failed to store/retrieve int")
	}

	slice := []int{1, 2, 3}
	c.Set("slice", slice)
	if got, ok := c.Get("slice"); !ok {
		t.Error("Failed to retrieve slice")
	} else {
		gotSlice := got.([]int)
		if len(gotSlice) != 3 {
			t.Error("Retrieved slice has wrong length")
		}
	}

	type TestStruct struct {
		Name  string
		Value int
	}
	c.Set("struct", TestStruct{Name: "test", Value: 100})
	if got, ok := c.Get("struct"); !ok {
		t.Error("Failed to retrieve struct")
	} else {
		gotStruct := got.(TestStruct)
		if gotStruct.Name != "test" || gotStruct.Value != 100 {
			t.Error("Retrieved struct has wrong values")
		}
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Set("shared-key", idx*100+j)
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Get("shared-key")
			}
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				c.Delete("shared-key")
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()
}

func TestMemoryCache_OverwriteValue(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	c.Set("key", "value1")
	c.Set("key", "value2")

	got, ok := c.Get("key")
	if !ok {
		t.Error("Get() returned false")
	}
	if got != "value2" {
		t.Errorf("Get() = %v, want %v", got, "value2")
	}
}

func TestMemoryCache_ImplementsInterface(t *testing.T) {
	var _ Cache = (*MemoryCache)(nil)
}

func TestMemoryCache_NilValue(t *testing.T) {
	c := NewMemory(time.Minute)
	defer c.Stop()

	c.Set("nil-key", nil)

	got, ok := c.Get("nil-key")
	if !ok {
		t.Error("Get() should return true for key with nil value")
	}
	if got != nil {
		t.Errorf("Get() should return nil for nil value, got %v", got)
	}
}
