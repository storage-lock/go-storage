package storage

import "context"

// LockWatchable 是可选的能力接口，表示存储介质支持对锁状态的监听/通知。
// 上层 Lock/FairLock 可通过类型断言检查存储是否实现该接口，并据此选择：
//   - 实现：直接使用存储原生的 Watch/Notify 能力
//   - 未实现：降级为轮询（由上层封装），不影响基本功能
//
// 典型实现：
//   - etcd: etcd.Watch
//   - Redis: Pub/Sub / Keyspace Notifications
//   - PostgreSQL: LISTEN/NOTIFY
//   - MongoDB: Change Streams
type LockWatchable interface {
	// Watch 监听指定锁的状态变化，并在锁信息变更时推送最新的 LockInformation。
	// channel 关闭或 ctx 取消时，监听结束。
	// 返回的 channel 不应为 nil；若底层无法监听，可直接返回 error。
	Watch(ctx context.Context, lockId string) (<-chan *LockInformation, error)
}
