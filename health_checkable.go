package storage

import (
	"context"
)

// HealthCheckable 是可选的健康检查能力接口
// Storage 实现可以不实现此接口；调用方可通过类型断言检查存储是否支持健康检查
type HealthCheckable interface {
	// Health 检查存储是否可用，返回 nil 表示健康
	// 实现应快速返回（如 ping / lightweight query），不执行实际业务操作
	Health(ctx context.Context) error
}
