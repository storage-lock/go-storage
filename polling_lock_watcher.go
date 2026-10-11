package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PollingLockWatcher 对不支持原生 Watch 的存储提供轮询降级。
type PollingLockWatcher struct {
	storage    Storage
	pollPeriod time.Duration
	closeOnce  sync.Once
	closed     chan struct{}
}

// NewPollingLockWatcher 创建一个轮询 watcher。
func NewPollingLockWatcher(storage Storage, pollPeriod time.Duration) *PollingLockWatcher {
	return &PollingLockWatcher{
		storage:    storage,
		pollPeriod: pollPeriod,
		closed:     make(chan struct{}),
	}
}

// Watch 监听指定锁的状态变化。
func (x *PollingLockWatcher) Watch(ctx context.Context, lockId string) (<-chan *LockInformation, error) {
	if lockId == "" {
		return nil, fmt.Errorf("lockId is empty")
	}

	out := make(chan *LockInformation)
	go func() {
		defer close(out)
		ticker := time.NewTicker(x.pollPeriod)
		defer ticker.Stop()

		var lastVersion Version
		for {
			select {
			case <-ctx.Done():
				return
			case <-x.closed:
				return
			case <-ticker.C:
			}

			jsonStr, err := x.storage.Get(ctx, lockId)
			if err != nil {
				continue
			}
			if jsonStr == "" {
				continue
			}
			info, err := LockInformationFromJsonString(jsonStr)
			if err != nil {
				continue
			}
			if info.Version == lastVersion {
				continue
			}
			lastVersion = info.Version
			select {
			case <-ctx.Done():
				return
			case <-x.closed:
				return
			case out <- info:
			}
		}
	}()

	return out, nil
}

// Close 关闭监听器并释放底层资源。
func (x *PollingLockWatcher) Close() error {
	x.closeOnce.Do(func() {
		close(x.closed)
	})
	return nil
}
