package storage

import (
	"context"
	"testing"
)

// TestDsnConnectionManagerTakeRetryOnFailure 钉死漏洞4：sql.Open 首次失败后不缓存 err，
// 下次 Take 可重试。修复前 sync.Once 永久缓存 err，Take 永远返回同一错误对象无法恢复（liveness）。
// 用未注册的 driver 名让 sql.Open 失败。验证两次 Take 返回的是不同的 err 对象（说明重新尝试了），
// 而非同一个缓存的 err。
func TestDsnConnectionManagerTakeRetryOnFailure(t *testing.T) {
	cm := NewDsnConnectionManager("nonexistent-driver-xyz", "whatever-dsn")

	db1, err1 := cm.Take(context.Background())
	if err1 == nil {
		t.Fatal("未注册 driver 应使 sql.Open 失败，实际 err=nil")
	}
	if db1 != nil {
		t.Fatal("失败时 db 应为 nil")
	}

	db2, err2 := cm.Take(context.Background())
	if err2 == nil {
		t.Fatal("第二次 Take 仍应失败（driver 仍未注册），实际 err=nil")
	}
	if db2 != nil {
		t.Fatal("失败时 db 应为 nil")
	}

	// 漏洞4 核心：err1 与 err2 是不同的错误对象，说明第二次 Take 重新调用了 sql.Open
	// （而非返回 sync.Once 缓存的同一个 err）。修复前两者是同一个指针。
	if err1 == err2 {
		t.Fatalf("失败应不缓存，两次 Take 应返回不同 err 对象（漏洞4：可重试），实际相同: %v", err1)
	}
}
