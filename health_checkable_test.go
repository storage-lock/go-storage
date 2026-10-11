package storage

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockHealthyStorage 实现了 HealthCheckable，总是返回健康
type mockHealthyStorage struct{}

func (m *mockHealthyStorage) Health(_ context.Context) error {
	return nil
}

// mockUnhealthyStorage 实现了 HealthCheckable，总是返回错误
type mockUnhealthyStorage struct{}

func (m *mockUnhealthyStorage) Health(_ context.Context) error {
	return assert.AnError
}

// nonHealthCheckableStorage 未实现 HealthCheckable
type nonHealthCheckableStorage struct{}

func TestHealthCheckable_Healthy(t *testing.T) {
	var s HealthCheckable = &mockHealthyStorage{}
	err := s.Health(context.Background())
	assert.NoError(t, err)
}

func TestHealthCheckable_Unhealthy(t *testing.T) {
	var s HealthCheckable = &mockUnhealthyStorage{}
	err := s.Health(context.Background())
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestHealthCheckable_TypeAssertion(t *testing.T) {
	var healthy HealthCheckable = &mockHealthyStorage{}
	var unhealthy HealthCheckable = &mockUnhealthyStorage{}
	nonHealth := &nonHealthCheckableStorage{}

	// 支持健康检查的类型断言成功
	if h, ok := healthy.(HealthCheckable); ok {
		assert.NoError(t, h.Health(context.Background()))
	} else {
		t.Fatal("healthy 应该实现 HealthCheckable")
	}

	// 支持但不健康的类型断言成功
	if h, ok := unhealthy.(HealthCheckable); ok {
		assert.Error(t, h.Health(context.Background()))
	} else {
		t.Fatal("unhealthy 应该实现 HealthCheckable")
	}

	// 不支持的类型断言失败
	nonHealthValue := reflect.ValueOf(nonHealth)
	_, ok := nonHealthValue.Interface().(HealthCheckable)
	assert.False(t, ok, "nonHealthCheckableStorage 不应该实现 HealthCheckable")
}

func TestHealthCheckable_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var s HealthCheckable = &mockHealthyStorage{}
	// 即使 context 已取消，mock 仍返回 nil；真实实现应尊重 context
	err := s.Health(ctx)
	assert.NoError(t, err)
}
