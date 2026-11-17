package storage

import "context"

// ErrAtomicBatchUnsupported 表示这次存储不能把多个 key 放进同一次线性一致提交。
// 上层遇到它必须拒绝跨 key 全有全无锁，不能退回逐 key 加锁再宣称原子。
var ErrAtomicBatchUnsupported = errString("atomic batch is not supported by this storage")

type errString string

func (e errString) Error() string { return string(e) }

// CompareAndSwapBatch 是给嵌入 Storage 的实现用的默认拒绝。
// 声明了 CapabilityAtomicBatch 的存储必须用自己的实现覆盖它，不能留这个默认。
func CompareAndSwapBatchUnsupported(context.Context, []CASOp) error {
	return ErrAtomicBatchUnsupported
}
