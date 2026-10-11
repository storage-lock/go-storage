package storage

import (
	"context"
	"testing"
	"time"

	"github.com/golang-infrastructure/go-iterator"
	"github.com/stretchr/testify/assert"
)

// mockStorage 用于测试能力检查的 mock 实现
type mockStorage struct {
	capabilities []StorageCapability
}

func (m *mockStorage) Capabilities() []StorageCapability {
	return m.capabilities
}

func (m *mockStorage) CreateWithVersion(ctx context.Context, lockId string, version Version, lockInformation *LockInformation) error {
	return nil
}

func (m *mockStorage) UpdateWithVersion(ctx context.Context, lockId string, expectedVersion, newVersion Version, lockInformation *LockInformation) error {
	return nil
}

func (m *mockStorage) DeleteWithVersion(ctx context.Context, lockId string, expectedVersion Version, lockInformation *LockInformation) error {
	return nil
}

func (m *mockStorage) Get(ctx context.Context, lockId string) (string, error) {
	return "", nil
}

func (m *mockStorage) GetTime(ctx context.Context) (time.Time, error) {
	return time.Time{}, nil
}

func (m *mockStorage) List(ctx context.Context) (iterator.Iterator[*LockInformation], error) {
	return nil, nil
}

func (m *mockStorage) Close(ctx context.Context) error {
	return nil
}

func (m *mockStorage) Init(ctx context.Context) error {
	return nil
}

func (m *mockStorage) GetName() string {
	return ""
}

func TestHasCapability(t *testing.T) {
	// 测试存储支持指定能力
	s := &mockStorage{capabilities: []StorageCapability{CapabilityCAS, CapabilityReliableTime}}
	assert.True(t, HasCapability(s, CapabilityCAS))
	assert.True(t, HasCapability(s, CapabilityReliableTime))

	// 测试存储不支持指定能力
	assert.False(t, HasCapability(s, CapabilityAtomicDelete))

	// 测试空能力列表
	s2 := &mockStorage{capabilities: []StorageCapability{}}
	assert.False(t, HasCapability(s2, CapabilityCAS))
}

func TestCheckCapabilities(t *testing.T) {
	// 测试满足所有必要条件
	s1 := &mockStorage{capabilities: []StorageCapability{CapabilityCAS, CapabilityReliableTime}}
	missing1 := CheckCapabilities(s1)
	assert.Empty(t, missing1)

	// 测试缺少 CAS
	s2 := &mockStorage{capabilities: []StorageCapability{CapabilityReliableTime}}
	missing2 := CheckCapabilities(s2)
	assert.Len(t, missing2, 1)
	assert.Equal(t, CapabilityCAS, missing2[0])

	// 测试缺少可靠时间
	s3 := &mockStorage{capabilities: []StorageCapability{CapabilityCAS}}
	missing3 := CheckCapabilities(s3)
	assert.Len(t, missing3, 1)
	assert.Equal(t, CapabilityReliableTime, missing3[0])

	// 测试缺少所有必要条件
	s4 := &mockStorage{capabilities: []StorageCapability{CapabilityAtomicDelete}}
	missing4 := CheckCapabilities(s4)
	assert.Len(t, missing4, 2)
}

func TestSupportsAtomicDelete(t *testing.T) {
	// 测试支持原子删除
	s1 := &mockStorage{capabilities: []StorageCapability{CapabilityCAS, CapabilityReliableTime, CapabilityAtomicDelete}}
	assert.True(t, SupportsAtomicDelete(s1))

	// 测试不支持原子删除
	s2 := &mockStorage{capabilities: []StorageCapability{CapabilityCAS, CapabilityReliableTime}}
	assert.False(t, SupportsAtomicDelete(s2))

	// 测试只支持原子删除但不支持其他能力
	s3 := &mockStorage{capabilities: []StorageCapability{CapabilityAtomicDelete}}
	assert.True(t, SupportsAtomicDelete(s3))
}

func TestCapabilityWatch(t *testing.T) {
	// 验证 CapabilityWatch 常量存在且值正确
	assert.Equal(t, StorageCapability("watch"), CapabilityWatch)

	// 测试存储声明 CapabilityWatch
	s1 := &mockStorage{capabilities: []StorageCapability{CapabilityCAS, CapabilityReliableTime, CapabilityWatch}}
	assert.True(t, HasCapability(s1, CapabilityWatch))

	// 测试存储未声明 CapabilityWatch
	s2 := &mockStorage{capabilities: []StorageCapability{CapabilityCAS, CapabilityReliableTime}}
	assert.False(t, HasCapability(s2, CapabilityWatch))
}
