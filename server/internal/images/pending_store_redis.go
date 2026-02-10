package images

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	defaultPendingRedisPrefix  = "pending-upload:"
	defaultPendingRedisTimeout = 2 * time.Second
)

// RedisPendingStore keeps approved uploads in Redis until user confirms Save.
type RedisPendingStore struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// NewRedisPendingStore creates a Redis-backed pending upload store.
func NewRedisPendingStore(client *redis.Client, ttl time.Duration) *RedisPendingStore {
	return NewRedisPendingStoreWithPrefix(client, ttl, defaultPendingRedisPrefix)
}

// NewRedisPendingStoreWithPrefix creates a Redis-backed pending store with explicit key prefix.
func NewRedisPendingStoreWithPrefix(client *redis.Client, ttl time.Duration, prefix string) *RedisPendingStore {
	if ttl <= 0 {
		ttl = defaultPendingTTL
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = defaultPendingRedisPrefix
	}

	return &RedisPendingStore{
		client: client,
		ttl:    ttl,
		prefix: prefix,
	}
}

func (s *RedisPendingStore) key(uploadID string) string {
	return s.prefix + uploadID
}

// Put stores an approved upload and returns its token id.
func (s *RedisPendingStore) Put(upload PendingUpload) string {
	if s.client == nil {
		return ""
	}

	id := uuid.NewString()
	upload.ID = id
	upload.ExpiresAt = time.Now().Add(s.ttl)

	payload, err := json.Marshal(upload)
	if err != nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultPendingRedisTimeout)
	defer cancel()

	if err := s.client.Set(ctx, s.key(id), payload, s.ttl).Err(); err != nil {
		return ""
	}

	return id
}

// Get fetches an upload token scoped to a user.
func (s *RedisPendingStore) Get(ownerUserID, uploadID string) (*PendingUpload, bool) {
	if s.client == nil {
		return nil, false
	}

	uploadID = strings.TrimSpace(uploadID)
	if uploadID == "" {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultPendingRedisTimeout)
	defer cancel()

	payload, err := s.client.Get(ctx, s.key(uploadID)).Bytes()
	if err != nil {
		return nil, false
	}

	var upload PendingUpload
	if err := json.Unmarshal(payload, &upload); err != nil {
		return nil, false
	}
	if upload.OwnerUserID != ownerUserID {
		return nil, false
	}
	if upload.ID == "" {
		upload.ID = uploadID
	}

	return &upload, true
}

// Delete removes an upload token.
func (s *RedisPendingStore) Delete(uploadID string) {
	if s.client == nil {
		return
	}

	uploadID = strings.TrimSpace(uploadID)
	if uploadID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultPendingRedisTimeout)
	defer cancel()

	s.client.Del(ctx, s.key(uploadID))
}

var _ PendingStore = (*RedisPendingStore)(nil)
