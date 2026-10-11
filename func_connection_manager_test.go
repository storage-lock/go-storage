package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFuncConnectionProvider_ReturnsNonNil(t *testing.T) {
	cm := NewFuncConnectionProvider[string]()
	assert.NotNil(t, cm)
}

func TestFuncConnectionProvider_SetName_Chainable(t *testing.T) {
	cm := NewFuncConnectionProvider[string]()
	result := cm.SetName("test-name")
	assert.Equal(t, cm, result, "SetName 应返回自身以支持链式调用")
	assert.Equal(t, "test-name", cm.Name())
}

func TestFuncConnectionProvider_SetName_DefaultEmpty(t *testing.T) {
	cm := NewFuncConnectionProvider[string]()
	assert.Empty(t, cm.Name())
}

func TestFuncConnectionProvider_SetTakeFunc_Chainable(t *testing.T) {
	cm := NewFuncConnectionProvider[string]()
	result := cm.SetTakeFunc(func(ctx context.Context) (string, error) {
		return "take", nil
	})
	assert.Equal(t, cm, result, "SetTakeFunc 应返回自身以支持链式调用")
}

func TestFuncConnectionProvider_SetReturnFunc_Chainable(t *testing.T) {
	cm := NewFuncConnectionProvider[string]()
	result := cm.SetReturnFunc(func(ctx context.Context, connection string) error {
		return nil
	})
	assert.Equal(t, cm, result, "SetReturnFunc 应返回自身以支持链式调用")
}

func TestFuncConnectionProvider_SetShutdownFunc_Chainable(t *testing.T) {
	cm := NewFuncConnectionProvider[string]()
	result := cm.SetShutdownFunc(func(ctx context.Context) error {
		return nil
	})
	assert.Equal(t, cm, result, "SetShutdownFunc 应返回自身以支持链式调用")
}

func TestFuncConnectionProvider_Take_NilFunc_ReturnsError(t *testing.T) {
	cm := NewFuncConnectionProvider[string]()
	_, err := cm.Take(context.Background())
	assert.Error(t, err)
}

func TestFuncConnectionProvider_Take_NilFunc_ReturnsZeroValue(t *testing.T) {
	cm := NewFuncConnectionProvider[string]()
	result, _ := cm.Take(context.Background())
	assert.Empty(t, result)
}

func TestFuncConnectionProvider_Take_WithFunc_ReturnsFuncResult(t *testing.T) {
	cm := NewFuncConnectionProvider[string]().
		SetTakeFunc(func(ctx context.Context) (string, error) {
			return "custom", nil
		})
	result, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "custom", result)
}

func TestFuncConnectionProvider_Take_WithFunc_Error(t *testing.T) {
	expectedErr := errors.New("take failed")
	cm := NewFuncConnectionProvider[string]().
		SetTakeFunc(func(ctx context.Context) (string, error) {
			return "", expectedErr
		})
	result, err := cm.Take(context.Background())
	assert.ErrorIs(t, err, expectedErr)
	assert.Empty(t, result)
}

func TestFuncConnectionProvider_Take_PassesContext(t *testing.T) {
	var receivedCtx context.Context
	cm := NewFuncConnectionProvider[string]().
		SetTakeFunc(func(ctx context.Context) (string, error) {
			receivedCtx = ctx
			return "", nil
		})
	ctx := context.WithValue(context.Background(), "key", "value")
	_, _ = cm.Take(ctx)
	assert.NotNil(t, receivedCtx)
	assert.Equal(t, "value", receivedCtx.Value("key"))
}

func TestFuncConnectionProvider_Return_NilFunc_ReturnsNil(t *testing.T) {
	cm := NewFuncConnectionProvider[string]()
	err := cm.Return(context.Background(), "anything")
	assert.NoError(t, err)
}

func TestFuncConnectionProvider_Return_WithFunc_ReturnsFuncResult(t *testing.T) {
	expectedErr := errors.New("return failed")
	cm := NewFuncConnectionProvider[string]().
		SetReturnFunc(func(ctx context.Context, connection string) error {
			return expectedErr
		})
	err := cm.Return(context.Background(), "connection")
	assert.ErrorIs(t, err, expectedErr)
}

func TestFuncConnectionProvider_Return_WithFunc_NoError(t *testing.T) {
	cm := NewFuncConnectionProvider[string]().
		SetReturnFunc(func(ctx context.Context, connection string) error {
			return nil
		})
	err := cm.Return(context.Background(), "connection")
	assert.NoError(t, err)
}

func TestFuncConnectionProvider_Return_WithFunc_ReceivesConnection(t *testing.T) {
	var receivedConnection string
	cm := NewFuncConnectionProvider[string]().
		SetReturnFunc(func(ctx context.Context, connection string) error {
			receivedConnection = connection
			return nil
		})
	err := cm.Return(context.Background(), "my-connection")
	assert.NoError(t, err)
	assert.Equal(t, "my-connection", receivedConnection)
}

func TestFuncConnectionProvider_Shutdown_NilFunc_ReturnsNil(t *testing.T) {
	cm := NewFuncConnectionProvider[string]()
	err := cm.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestFuncConnectionProvider_Shutdown_WithFunc_ReturnsFuncResult(t *testing.T) {
	expectedErr := errors.New("shutdown failed")
	cm := NewFuncConnectionProvider[string]().
		SetShutdownFunc(func(ctx context.Context) error {
			return expectedErr
		})
	err := cm.Shutdown(context.Background())
	assert.ErrorIs(t, err, expectedErr)
}

func TestFuncConnectionProvider_Shutdown_WithFunc_NoError(t *testing.T) {
	cm := NewFuncConnectionProvider[string]().
		SetShutdownFunc(func(ctx context.Context) error {
			return nil
		})
	err := cm.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestFuncConnectionProvider_Shutdown_WithFunc_ReceivesContext(t *testing.T) {
	var receivedCtx context.Context
	cm := NewFuncConnectionProvider[string]().
		SetShutdownFunc(func(ctx context.Context) error {
			receivedCtx = ctx
			return nil
		})
	ctx := context.WithValue(context.Background(), "shutdown-key", "shutdown-value")
	_ = cm.Shutdown(ctx)
	assert.NotNil(t, receivedCtx)
	assert.Equal(t, "shutdown-value", receivedCtx.Value("shutdown-key"))
}

func TestFuncConnectionProvider_ChainAllSetters(t *testing.T) {
	cm := NewFuncConnectionProvider[string]().
		SetName("chain-test").
		SetTakeFunc(func(ctx context.Context) (string, error) { return "take", nil }).
		SetReturnFunc(func(ctx context.Context, connection string) error { return nil }).
		SetShutdownFunc(func(ctx context.Context) error { return nil })

	assert.Equal(t, "chain-test", cm.Name())

	result, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "take", result)

	err = cm.Return(context.Background(), "conn")
	assert.NoError(t, err)

	err = cm.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestFuncConnectionProvider_GenericTypeInt(t *testing.T) {
	// 验证泛型类型 int 也能正常工作
	cm := NewFuncConnectionProvider[int]().
		SetTakeFunc(func(ctx context.Context) (int, error) {
			return 42, nil
		})
	result, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestFuncConnectionProvider_GenericTypeStruct(t *testing.T) {
	type customStruct struct {
		Value string
	}
	cm := NewFuncConnectionProvider[customStruct]().
		SetTakeFunc(func(ctx context.Context) (customStruct, error) {
			return customStruct{Value: "test"}, nil
		})
	result, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "test", result.Value)
}
