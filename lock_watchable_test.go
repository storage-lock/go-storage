package storage

import (
	"context"
	"testing"
	"time"

	"github.com/golang-infrastructure/go-iterator"
	"github.com/stretchr/testify/assert"
)

// mockWatchableStorage 用于测试 LockWatchable 接口
type mockWatchableStorage struct {
	events []*LockInformation
	index  int
}

func newMockWatchableStorage(events ...*LockInformation) *mockWatchableStorage {
	return &mockWatchableStorage{events: events}
}

func (m *mockWatchableStorage) Capabilities() []StorageCapability {
	return []StorageCapability{CapabilityCAS, CapabilityReliableTime, CapabilityWatch}
}

func (m *mockWatchableStorage) CreateWithVersion(ctx context.Context, lockId string, version Version, lockInformation *LockInformation) error {
	return nil
}

func (m *mockWatchableStorage) UpdateWithVersion(ctx context.Context, lockId string, expectedVersion, newVersion Version, lockInformation *LockInformation) error {
	return nil
}

func (m *mockWatchableStorage) DeleteWithVersion(ctx context.Context, lockId string, expectedVersion Version, lockInformation *LockInformation) error {
	return nil
}

func (m *mockWatchableStorage) Get(ctx context.Context, lockId string) (string, error) {
	return "", nil
}

func (m *mockWatchableStorage) GetTime(ctx context.Context) (time.Time, error) {
	return time.Time{}, nil
}

func (m *mockWatchableStorage) List(ctx context.Context) (iterator.Iterator[*LockInformation], error) {
	return nil, nil
}

func (m *mockWatchableStorage) Close(ctx context.Context) error {
	return nil
}

func (m *mockWatchableStorage) Init(ctx context.Context) error {
	return nil
}

func (m *mockWatchableStorage) GetName() string {
	return "mock-watchable-storage"
}

func (m *mockWatchableStorage) Watch(ctx context.Context, lockId string) (<-chan *LockInformation, error) {
	ch := make(chan *LockInformation, len(m.events))
	go func() {
		defer close(ch)
		for _, info := range m.events {
			select {
			case <-ctx.Done():
				return
			case ch <- info:
			}
		}
	}()
	return ch, nil
}

func TestLockWatchable_TypeAssertion(t *testing.T) {
	var s Storage = newMockWatchableStorage()
	_, ok := s.(LockWatchable)
	assert.True(t, ok, "mockWatchableStorage 应实现 LockWatchable 接口")
}

func TestLockWatchable_Watch(t *testing.T) {
	events := []*LockInformation{
		{LockId: "lock-1", OwnerId: "owner-1", Version: 1, LockCount: 1},
		{LockId: "lock-1", OwnerId: "owner-1", Version: 2, LockCount: 1},
	}
	storage := newMockWatchableStorage(events...)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := storage.Watch(ctx, "lock-1")
	assert.NoError(t, err)
	assert.NotNil(t, ch)

	count := 0
	for info := range ch {
		assert.NotNil(t, info)
		count++
	}
	assert.Equal(t, 2, count)
}
