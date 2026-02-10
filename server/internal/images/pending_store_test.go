package images

import (
	"testing"
	"time"
)

func TestInMemoryPendingStoreEvictsOldestByEntryLimit(t *testing.T) {
	store := NewInMemoryPendingStoreWithLimits(time.Hour, 2, 1024)

	id1 := store.Put(PendingUpload{OwnerUserID: "user-1", ImageBytes: []byte{1}})
	id2 := store.Put(PendingUpload{OwnerUserID: "user-1", ImageBytes: []byte{2}})
	id3 := store.Put(PendingUpload{OwnerUserID: "user-1", ImageBytes: []byte{3}})

	if _, ok := store.Get("user-1", id1); ok {
		t.Fatalf("expected oldest upload %q to be evicted", id1)
	}
	if _, ok := store.Get("user-1", id2); !ok {
		t.Fatalf("expected second upload %q to remain", id2)
	}
	if _, ok := store.Get("user-1", id3); !ok {
		t.Fatalf("expected newest upload %q to remain", id3)
	}
}

func TestInMemoryPendingStoreEvictsOldestByByteLimit(t *testing.T) {
	store := NewInMemoryPendingStoreWithLimits(time.Hour, 10, 6)

	id1 := store.Put(PendingUpload{
		OwnerUserID: "user-1",
		ImageBytes:  []byte{1, 2, 3, 4},
	})
	id2 := store.Put(PendingUpload{
		OwnerUserID: "user-1",
		ImageBytes:  []byte{5, 6, 7, 8},
	})

	if _, ok := store.Get("user-1", id1); ok {
		t.Fatalf("expected upload %q to be evicted once byte limit exceeded", id1)
	}
	if _, ok := store.Get("user-1", id2); !ok {
		t.Fatalf("expected newest upload %q to remain", id2)
	}
}
