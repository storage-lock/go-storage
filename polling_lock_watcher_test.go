package storage

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang-infrastructure/go-iterator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pollingMockStorage 手写的内存 mock Storage，供 PollingLockWatcher 测试使用。
type pollingMockStorage struct {
	mu     sync.Mutex
	data   map[string]string
	getErr error
}

func newPollingMockStorage() *pollingMockStorage {
	return &pollingMockStorage{data: make(map[string]string)}
}

func (x *pollingMockStorage) GetName() string { return "polling-mock" }

func (x *pollingMockStorage) Capabilities() []StorageCapability {
	return []StorageCapability{CapabilityCAS, CapabilityReliableTime}
}

func (x *pollingMockStorage) Init(ctx context.Context) error { return nil }

func (x *pollingMockStorage) UpdateWithVersion(ctx context.Context, lockId string, exceptedVersion, newVersion Version, lockInformation *LockInformation) error {
	return nil
}

func (x *pollingMockStorage) CreateWithVersion(ctx context.Context, lockId string, version Version, lockInformation *LockInformation) error {
	return nil
}

func (x *pollingMockStorage) DeleteWithVersion(ctx context.Context, lockId string, exceptedVersion Version, lockInformation *LockInformation) error {
	return nil
}

func (x *pollingMockStorage) Get(ctx context.Context, lockId string) (string, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.getErr != nil {
		return "", x.getErr
	}
	return x.data[lockId], nil
}

func (x *pollingMockStorage) GetTime(ctx context.Context) (time.Time, error) {
	return time.Now(), nil
}

func (x *pollingMockStorage) Close(ctx context.Context) error { return nil }

func (x *pollingMockStorage) List(ctx context.Context) (iterator.Iterator[*LockInformation], error) {
	return nil, nil
}

func (x *pollingMockStorage) put(lockId string, info *LockInformation) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.data[lockId] = info.ToJsonString()
}

func TestNewPollingLockWatcher(t *testing.T) {
	s := newPollingMockStorage()
	w := NewPollingLockWatcher(s, 10*time.Millisecond)
	require.NotNil(t, w)
	require.NotNil(t, w.closed)
	assert.NoError(t, w.Close())
}

func TestPollingLockWatcher_Watch_EmptyLockId(t *testing.T) {
	s := newPollingMockStorage()
	w := NewPollingLockWatcher(s, 10*time.Millisecond)
	defer func() { _ = w.Close() }()
	ch, err := w.Watch(context.Background(), "")
	assert.Error(t, err)
	assert.Nil(t, ch)
}

func TestPollingLockWatcher_Watch_VersionChange(t *testing.T) {
	s := newPollingMockStorage()
	w := NewPollingLockWatcher(s, 10*time.Millisecond)
	defer func() { _ = w.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info := &LockInformation{LockId: "l1", OwnerId: "o1", Version: 1, LockCount: 1, LockBeginTime: time.Now(), LeaseExpireTime: time.Now().Add(time.Minute)}
	s.put("l1", info)

	ch, err := w.Watch(ctx, "l1")
	require.NoError(t, err)
	require.NotNil(t, ch)

	select {
	case got := <-ch:
		require.NotNil(t, got)
		assert.Equal(t, Version(1), got.Version)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for first version push")
	}

	// 版本变更后应再次推送
	info2 := &LockInformation{LockId: "l1", OwnerId: "o1", Version: 2, LockCount: 1, LockBeginTime: time.Now(), LeaseExpireTime: time.Now().Add(time.Minute)}
	s.put("l1", info2)

	select {
	case got := <-ch:
		require.NotNil(t, got)
		assert.Equal(t, Version(2), got.Version)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for second version push")
	}
}

func TestPollingLockWatcher_Watch_GetErrorSkipped(t *testing.T) {
	s := newPollingMockStorage()
	w := NewPollingLockWatcher(s, 10*time.Millisecond)
	defer func() { _ = w.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 先让 Get 报错，watcher 应跳过而不是退出
	s.getErr = assert.AnError
	ch, err := w.Watch(ctx, "l1")
	require.NoError(t, err)

	// 写入坏 JSON，watcher 应跳过解析失败
	s.getErr = nil
	s.mu.Lock()
	s.data["l1"] = "not-json{{{"
	s.mu.Unlock()
	time.Sleep(50 * time.Millisecond)

	// 恢复正常数据，应能收到推送
	info := &LockInformation{LockId: "l1", OwnerId: "o1", Version: 7, LockCount: 1, LockBeginTime: time.Now(), LeaseExpireTime: time.Now().Add(time.Minute)}
	s.put("l1", info)

	select {
	case got := <-ch:
		require.NotNil(t, got)
		assert.Equal(t, Version(7), got.Version)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for push after error recovery")
	}
}

func TestPollingLockWatcher_Watch_ContextCancel(t *testing.T) {
	s := newPollingMockStorage()
	w := NewPollingLockWatcher(s, 10*time.Millisecond)
	defer func() { _ = w.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := w.Watch(ctx, "l1")
	require.NoError(t, err)

	cancel()
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed after ctx cancel")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for channel close after ctx cancel")
	}
}

func TestPollingLockWatcher_Close_Idempotent(t *testing.T) {
	s := newPollingMockStorage()
	w := NewPollingLockWatcher(s, 10*time.Millisecond)

	assert.NoError(t, w.Close())
	// 第二次 Close 不应 panic（sync.Once 保证）
	assert.NotPanics(t, func() {
		assert.NoError(t, w.Close())
	})

	// Close 后正在 Watch 的 goroutine 应退出
	ctx := context.Background()
	ch, err := w.Watch(ctx, "l1")
	require.NoError(t, err)
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed after watcher Close")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for channel close after watcher Close")
	}
}

func TestPollingLockWatcher_Watch_GetErrorSkippedPersistently(t *testing.T) {
	s := newPollingMockStorage()
	w := NewPollingLockWatcher(s, 10*time.Millisecond)
	defer func() { _ = w.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get 持续报错多个轮询周期，watcher 应一直跳过而不退出
	s.mu.Lock()
	s.getErr = assert.AnError
	s.mu.Unlock()
	ch, err := w.Watch(ctx, "l1")
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	// 恢复正常后应能收到推送
	s.mu.Lock()
	s.getErr = nil
	s.mu.Unlock()
	info := &LockInformation{LockId: "l1", OwnerId: "o1", Version: 3, LockCount: 1, LockBeginTime: time.Now(), LeaseExpireTime: time.Now().Add(time.Minute)}
	s.put("l1", info)

	select {
	case got := <-ch:
		require.NotNil(t, got)
		assert.Equal(t, Version(3), got.Version)
	case <-time.After(5 * time.Second):
		t.Fatal("Get 恢复正常后应收到推送")
	}
}

func TestPollingLockWatcher_Watch_EmptyStringSkipped(t *testing.T) {
	s := newPollingMockStorage()
	w := NewPollingLockWatcher(s, 10*time.Millisecond)
	defer func() { _ = w.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// "l1" 尚无数据，Get 返回空串，watcher 应跳过
	ch, err := w.Watch(ctx, "l1")
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	info := &LockInformation{LockId: "l1", OwnerId: "o1", Version: 5, LockCount: 1, LockBeginTime: time.Now(), LeaseExpireTime: time.Now().Add(time.Minute)}
	s.put("l1", info)

	select {
	case got := <-ch:
		require.NotNil(t, got)
		assert.Equal(t, Version(5), got.Version)
	case <-time.After(5 * time.Second):
		t.Fatal("写入数据后应收到推送")
	}
}

func TestPollingLockWatcher_Watch_SameVersionDeduped(t *testing.T) {
	s := newPollingMockStorage()
	w := NewPollingLockWatcher(s, 10*time.Millisecond)
	defer func() { _ = w.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	info := &LockInformation{LockId: "l1", OwnerId: "o1", Version: 7, LockCount: 1, LockBeginTime: time.Now(), LeaseExpireTime: time.Now().Add(time.Minute)}
	s.put("l1", info)

	ch, err := w.Watch(ctx, "l1")
	require.NoError(t, err)

	select {
	case got := <-ch:
		require.NotNil(t, got)
		assert.Equal(t, Version(7), got.Version)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for first push")
	}

	// 版本不变，后续轮询不应重复推送
	select {
	case got, ok := <-ch:
		if ok {
			t.Fatalf("相同版本不应重复推送，却收到版本 %d", got.Version)
		}
	case <-time.After(100 * time.Millisecond):
		// 预期：无新推送
	}
}

func TestPollingLockWatcher_Watch_BlockedPushCtxCancel(t *testing.T) {
	s := newPollingMockStorage()
	w := NewPollingLockWatcher(s, time.Millisecond)
	defer func() { _ = w.Close() }()

	// 持续变更版本但不读取，buffer（容量1）填满后推送阻塞
	info := &LockInformation{LockId: "l1", OwnerId: "o1", Version: 1, LockCount: 1, LockBeginTime: time.Now(), LeaseExpireTime: time.Now().Add(time.Minute)}
	s.put("l1", info)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := w.Watch(ctx, "l1")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	cancel()

	timeout := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-timeout:
			t.Fatal("ctx 取消后，阻塞在推送上的 Watch 未退出")
		}
	}
}

func TestPollingLockWatcher_Watch_BlockedPushClose(t *testing.T) {
	s := newPollingMockStorage()
	w := NewPollingLockWatcher(s, time.Millisecond)

	info := &LockInformation{LockId: "l1", OwnerId: "o1", Version: 1, LockCount: 1, LockBeginTime: time.Now(), LeaseExpireTime: time.Now().Add(time.Minute)}
	s.put("l1", info)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := w.Watch(ctx, "l1")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.NoError(t, w.Close())

	timeout := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-timeout:
			t.Fatal("Close 后，阻塞在推送上的 Watch 未退出")
		}
	}
}
