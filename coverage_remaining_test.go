package storage

import (
	"context"
	"testing"
	"time"
)

func TestCoverageRemainingStorageBranches(t *testing.T) {
	if info, err := LockInformationFromJsonString("null"); err != nil || info != nil {
		t.Fatalf("null info=%v err=%v", info, err)
	}
	bad := &LockInformation{LockBeginTime: time.Date(0, 0, 0, 0, 0, 0, 0, time.FixedZone("bad", 0))}
	if _, err := bad.ToJsonStringE(); err == nil {
		t.Fatal("invalid time serialized")
	}

	store := newPollingMockStorage()
	watcher := NewPollingLockWatcher(store, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := watcher.Watch(ctx, "full")
	if err != nil {
		t.Fatal(err)
	}
	store.put("full", &LockInformation{LockId: "full", Version: 1})
	deadline := time.After(time.Second)
	for {
		select {
		case <-ch:
			goto filled
		case <-deadline:
			t.Fatal("first event missing")
		}
	}
filled:
	if err := watcher.Close(); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("closed watcher delivered another event")
		}
	case <-time.After(time.Second):
		t.Fatal("watcher did not exit")
	}
}

type blockingSendStorage struct{ *pollingMockStorage }

func (x *blockingSendStorage) Get(ctx context.Context, id string) (string, error) {
	x.mu.Lock()
	value := x.data[id]
	x.mu.Unlock()
	if value == "" {
		<-ctx.Done()
		return "", ctx.Err()
	}
	x.mu.Lock()
	x.data[id] = ""
	x.mu.Unlock()
	return value, nil
}

func TestPollingWatchSendObservesCancellation(t *testing.T) {
	store := &blockingSendStorage{newPollingMockStorage()}
	watcher := NewPollingLockWatcher(store, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := watcher.Watch(ctx, "blocked")
	if err != nil {
		t.Fatal(err)
	}
	store.pollingMockStorage.put("blocked", &LockInformation{LockId: "blocked", Version: 1})
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("canceled watcher did not exit")
	}
}

func TestPollingWatchBlockedSendObservesCloseAndCancel(t *testing.T) {
	store := newPollingMockStorage()
	store.put("blocked", &LockInformation{LockId: "blocked", Version: 1})
	closed := NewPollingLockWatcher(store, time.Millisecond)
	closedCh, err := closed.Watch(context.Background(), "blocked")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case _, ok := <-closedCh:
		if ok {
			t.Fatal("closed watcher delivered event")
		}
	case <-time.After(time.Second):
		t.Fatal("closed watcher blocked")
	}

	canceled := NewPollingLockWatcher(store, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	canceledCh, err := canceled.Watch(ctx, "blocked")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case _, ok := <-canceledCh:
		if ok {
			t.Fatal("canceled watcher delivered event")
		}
	case <-time.After(time.Second):
		t.Fatal("canceled watcher blocked")
	}
}
