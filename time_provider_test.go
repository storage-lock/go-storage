package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// testTimeProvider 用于测试 TimeProvider 接口的简单实现
type testTimeProvider struct {
	fixedTime time.Time
}

func (t *testTimeProvider) GetTime(ctx context.Context) (time.Time, error) {
	if t.fixedTime.IsZero() {
		return time.Now(), nil
	}
	return t.fixedTime, nil
}

func TestTimeProvider_Interface_Satisfied(t *testing.T) {
	// 验证 testTimeProvider 实现了 TimeProvider 接口
	var _ TimeProvider = &testTimeProvider{}
	var _ TimeProvider = (*testTimeProvider)(nil)
}

func TestTimeProvider_GetTime_ReturnsTime(t *testing.T) {
	provider := &testTimeProvider{fixedTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)}
	result, err := provider.GetTime(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), result)
}

func TestTimeProvider_GetTime_ContextIgnored(t *testing.T) {
	provider := &testTimeProvider{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// 当前简单实现忽略 context 状态
	result, err := provider.GetTime(ctx)
	assert.NoError(t, err)
	assert.False(t, result.IsZero())
}
