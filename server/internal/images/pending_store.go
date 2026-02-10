package images

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryPendingStore keeps approved uploads in memory until user confirms Save.
type InMemoryPendingStore struct {
	mu          sync.RWMutex
	uploads     map[string]PendingUpload
	order       []string
	ttl         time.Duration
	maxEntries  int
	maxBytes    int64
	currentSize int64
}

const (
	defaultPendingTTL        = 10 * time.Minute
	defaultPendingMaxEntries = 100
	defaultPendingMaxBytes   = 100 * 1024 * 1024 // 100MB
)

// NewInMemoryPendingStore creates a pending store with the provided TTL.
func NewInMemoryPendingStore(ttl time.Duration) *InMemoryPendingStore {
	return NewInMemoryPendingStoreWithLimits(ttl, defaultPendingMaxEntries, defaultPendingMaxBytes)
}

// NewInMemoryPendingStoreWithLimits creates a pending store with explicit entry/byte limits.
func NewInMemoryPendingStoreWithLimits(ttl time.Duration, maxEntries int, maxBytes int64) *InMemoryPendingStore {
	if ttl <= 0 {
		ttl = defaultPendingTTL
	}
	if maxEntries <= 0 {
		maxEntries = defaultPendingMaxEntries
	}
	if maxBytes <= 0 {
		maxBytes = defaultPendingMaxBytes
	}

	return &InMemoryPendingStore{
		uploads:    make(map[string]PendingUpload),
		order:      make([]string, 0, maxEntries),
		ttl:        ttl,
		maxEntries: maxEntries,
		maxBytes:   maxBytes,
	}
}

// Put stores an approved upload and returns its token id.
func (s *InMemoryPendingStore) Put(upload PendingUpload) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.cleanupLocked(now)
	s.evictForCapacityLocked(int64(len(upload.ImageBytes)))

	id := uuid.NewString()
	upload.ID = id
	upload.ExpiresAt = now.Add(s.ttl)
	s.uploads[id] = upload
	s.order = append(s.order, id)
	s.currentSize += int64(len(upload.ImageBytes))

	return id
}

// Get fetches an upload token scoped to a user.
func (s *InMemoryPendingStore) Get(ownerUserID, uploadID string) (*PendingUpload, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.cleanupLocked(now)

	upload, ok := s.uploads[uploadID]
	if !ok {
		return nil, false
	}
	if upload.OwnerUserID != ownerUserID {
		return nil, false
	}

	copyUpload := upload
	return &copyUpload, true
}

// Delete removes an upload token.
func (s *InMemoryPendingStore) Delete(uploadID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteLocked(uploadID)
}

func (s *InMemoryPendingStore) cleanupLocked(now time.Time) {
	ids := append([]string(nil), s.order...)
	for _, id := range ids {
		upload, ok := s.uploads[id]
		if !ok {
			continue
		}
		if now.After(upload.ExpiresAt) {
			s.deleteLocked(id)
		}
	}
}

func (s *InMemoryPendingStore) evictForCapacityLocked(incomingSize int64) {
	for s.shouldEvictLocked(incomingSize) {
		if len(s.order) == 0 {
			return
		}
		s.deleteLocked(s.order[0])
	}
}

func (s *InMemoryPendingStore) shouldEvictLocked(incomingSize int64) bool {
	if s.maxEntries > 0 && len(s.uploads) >= s.maxEntries {
		return true
	}
	if s.maxBytes > 0 && len(s.uploads) > 0 && (s.currentSize+incomingSize) > s.maxBytes {
		return true
	}
	return false
}

func (s *InMemoryPendingStore) deleteLocked(uploadID string) {
	upload, ok := s.uploads[uploadID]
	if !ok {
		return
	}
	delete(s.uploads, uploadID)
	s.currentSize -= int64(len(upload.ImageBytes))
	if s.currentSize < 0 {
		s.currentSize = 0
	}

	for i, id := range s.order {
		if id == uploadID {
			s.order = append(s.order[:i], s.order[i+1:]...)
			return
		}
	}
}
