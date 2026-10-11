package storage

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDsnConnectionManager_Name_ReturnsConstant(t *testing.T) {
	cm := NewDsnConnectionManager("sqlite3", ":memory:")
	assert.Equal(t, DSNConnectionManagerName, cm.Name())
}

func TestDsnConnectionManager_Return_AlwaysNilError(t *testing.T) {
	cm := NewDsnConnectionManager("sqlite3", ":memory:")
	err := cm.Return(context.Background(), nil)
	assert.NoError(t, err)
}

func TestDsnConnectionManager_Shutdown_NoErrorWhenNil(t *testing.T) {
	cm := NewDsnConnectionManager("sqlite3", "bad-dsn")
	err := cm.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestDsnConnectionManager_Take_ReusesCachedDb(t *testing.T) {
	cm := NewDsnConnectionManager("sqlite3", ":memory:")

	fakeDB := &sql.DB{}
	cm.db = fakeDB
	cm.initialized = true

	first, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, fakeDB, first)

	second, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, fakeDB, second)
}

func TestDsnConnectionManager_Take_ErrorOnInvalidDriver(t *testing.T) {
	cm := NewDsnConnectionManager("nonexistent-driver-xyz", "whatever-dsn")
	_, err := cm.Take(context.Background())
	assert.Error(t, err)
}

func TestDsnConnectionManager_Take_RetryOnFailure(t *testing.T) {
	cm := NewDsnConnectionManager("nonexistent-driver-xyz", "whatever-dsn")

	_, err1 := cm.Take(context.Background())
	assert.Error(t, err1)

	_, err2 := cm.Take(context.Background())
	assert.Error(t, err2)

	// 漏洞4 核心：失败不缓存，两次 Take 应返回不同的 err 对象（说明重新尝试了）
	if err1 == err2 {
		t.Fatalf("失败应不缓存，两次 Take 应返回不同 err 对象（漏洞4：可重试），实际相同: %v", err1)
	}
}

func TestDsnConnectionManager_Take_ErrorDoesNotCache(t *testing.T) {
	cm := NewDsnConnectionManager("invalid-driver", "dsn")

	_, err := cm.Take(context.Background())
	assert.Error(t, err)
	assert.False(t, cm.initialized, "失败不应缓存 initialized 状态")
}

func TestDsnConnectionManager_Take_SuccessAndReuse(t *testing.T) {
	cm := NewDsnConnectionManager("sqlite3", ":memory:")

	fakeDB := &sql.DB{}
	cm.db = fakeDB
	cm.initialized = true

	first, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, fakeDB, first)

	second, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, first, second)
}


func TestDsnConnectionManager_Take_ContextCancelledBeforeSuccess(t *testing.T) {
	cm := NewDsnConnectionManager("invalid-driver", "dsn")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := cm.Take(ctx)
	assert.Error(t, err)
	assert.False(t, cm.initialized)
}

func TestDsnConnectionManager_Return_WithNilContext(t *testing.T) {
	cm := NewDsnConnectionManager("sqlite3", ":memory:")

	// Return 方法内部不使用 context，验证即使传入 nil context 也不会 panic
	err := cm.Return(nil, nil)
	assert.NoError(t, err)
}

func TestDsnConnectionManager_Shutdown_NilDbAfterFailedTake(t *testing.T) {
	cm := NewDsnConnectionManager("invalid-driver", "dsn")

	_, err := cm.Take(context.Background())
	assert.Error(t, err)

	// Shutdown 应当能够安全处理 nil db
	err = cm.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestDsnConnectionManager_Take_ConcurrentAccess(t *testing.T) {
	cm := NewDsnConnectionManager("invalid-driver", "dsn")

	// 并发调用 Take 不应出现 data race
	done := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		go func() {
			defer func() {
				done <- struct{}{}
			}()
			_, _ = cm.Take(context.Background())
		}()
	}

	<-done
	<-done
}
