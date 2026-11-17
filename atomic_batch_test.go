package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// batchCapableStorage 声明了 CapabilityAtomicBatch，但没有覆盖 CompareAndSwapBatch，
// 因此会走到嵌入 Storage 的默认拒绝实现。
type batchCapableStorage struct {
	*mockStorage
}

func newBatchCapableStorage() *batchCapableStorage {
	return &batchCapableStorage{mockStorage: &mockStorage{
		capabilities: []StorageCapability{CapabilityCAS, CapabilityReliableTime},
	}}
}

func (b *batchCapableStorage) Capabilities() []StorageCapability {
	caps := append([]StorageCapability{}, b.mockStorage.capabilities...)
	return append(caps, CapabilityAtomicBatch)
}

// batchImplStorage 同时声明能力并覆盖了 CompareAndSwapBatch。
type batchImplStorage struct {
	*batchCapableStorage
	ops []CASOp
}

func (b *batchImplStorage) CompareAndSwapBatch(_ context.Context, ops []CASOp) error {
	b.ops = append(b.ops, ops...)
	return nil
}

// TestAtomicBatchDefaultRejects 验证默认实现必须拒绝，
// 避免声明了 CapabilityAtomicBatch 却没真正实现的存储被当成支持跨 key 原子写。
func TestAtomicBatchDefaultRejects(t *testing.T) {
	store := newBatchCapableStorage()
	assert.True(t, HasCapability(store, CapabilityAtomicBatch))

	err := CompareAndSwapBatchUnsupported(context.Background(), []CASOp{{LockID: "a"}})
	assert.ErrorIs(t, err, ErrAtomicBatchUnsupported)
}

// TestAtomicBatchInterfaceDispatch 验证覆盖实现的存储会走自己的实现，
// 而不是默认拒绝。
func TestAtomicBatchInterfaceDispatch(t *testing.T) {
	store := &batchImplStorage{batchCapableStorage: newBatchCapableStorage()}
	var target AtomicBatchStorage = store
	err := target.CompareAndSwapBatch(context.Background(), []CASOp{
		{LockID: "a", ExpectedVersion: 0, NewVersion: 1, Information: &LockInformation{LockId: "a"}},
	})
	assert.NoError(t, err)
	assert.Len(t, store.ops, 1)
	assert.Equal(t, "a", store.ops[0].LockID)
}

// TestNotAtomicBatchStorageDoesNotSatisfy 验证不声明能力的存储不能被断言成
// AtomicBatchStorage。上层据此拒绝跨 key 全有全无锁。
func TestNotAtomicBatchStorageDoesNotSatisfy(t *testing.T) {
	plain := &mockStorage{capabilities: []StorageCapability{CapabilityCAS}}
	_, ok := Storage(plain).(AtomicBatchStorage)
	assert.False(t, ok, "普通单 key 存储不能被视为原子批量存储")
}

// TestAtomicBatchInheritedFromEmbeddedStorage 验证嵌入的通用实现
// 不会因为接口方法集而意外满足 AtomicBatchStorage。
func TestAtomicBatchInheritedFromEmbeddedStorage(t *testing.T) {
	store := newBatchCapableStorage()
	_, isBatch := Storage(store).(AtomicBatchStorage)
	assert.False(t, isBatch, "只声明能力、没有实现方法的存储不能被视为原子批量存储")
}
