package storage

import (
	"context"
	"fmt"
)

// FuncConnectionProvider 通过一个函数获取连接，这样就不必再单独写一个接口了，不是一个具体的实现，仅仅是为了外部实现简单一些
// TODO 2023-8-4 01:38:10 单元测试
//
// 并发安全约定（漏洞7）：Set 系列方法是 builder 语义，非并发安全。
// 调用方须在构造期完成所有 Set（链式调用），构造完成、开始并发 Take/Return/Shutdown 后不得再调 Set。
// 即 Set 与 Take 之间必须存在 happens-before 关系（如构造完成后通过 channel/sync 把实例发布给消费者）。
type FuncConnectionProvider[Connection any] struct {
	name         string
	takeFunc     func(ctx context.Context) (Connection, error)
	returnFunc   func(ctx context.Context, connection Connection) error
	shutdownFunc func(ctx context.Context) error
}

var _ ConnectionManager[any] = &FuncConnectionProvider[any]{}

func NewFuncConnectionProvider[Connection any]() *FuncConnectionProvider[Connection] {
	return &FuncConnectionProvider[Connection]{}
}

func (x *FuncConnectionProvider[Connection]) SetName(name string) *FuncConnectionProvider[Connection] {
	x.name = name
	return x
}

func (x *FuncConnectionProvider[Connection]) SetTakeFunc(takeFunc func(ctx context.Context) (Connection, error)) *FuncConnectionProvider[Connection] {
	x.takeFunc = takeFunc
	return x
}

func (x *FuncConnectionProvider[Connection]) SetReturnFunc(returnFunc func(ctx context.Context, connection Connection) error) *FuncConnectionProvider[Connection] {
	x.returnFunc = returnFunc
	return x
}

func (x *FuncConnectionProvider[Connection]) SetShutdownFunc(shutdownFunc func(ctx context.Context) error) *FuncConnectionProvider[Connection] {
	x.shutdownFunc = shutdownFunc
	return x
}

func (x *FuncConnectionProvider[Connection]) Name() string {
	return x.name
}

func (x *FuncConnectionProvider[Connection]) Take(ctx context.Context) (Connection, error) {
	if x.takeFunc == nil {
		var zero Connection
		return zero, fmt.Errorf("take func nil")
	} else {
		return x.takeFunc(ctx)
	}
}

func (x *FuncConnectionProvider[Connection]) Return(ctx context.Context, connection Connection) error {
	if x.returnFunc != nil {
		return x.returnFunc(ctx, connection)
	} else {
		return nil
	}
}

func (x *FuncConnectionProvider[Connection]) Shutdown(ctx context.Context) error {
	if x.shutdownFunc != nil {
		return x.shutdownFunc(ctx)
	} else {
		return nil
	}
}
