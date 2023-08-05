package storage

import (
	"context"
	"time"
)

// TimeProvider 能够提供时间的时间源，可以从数据库读取，也可以从NTP服务器上读取，只要能够返回一个准确的时间即可（其实不准确也可以，只要是统一的，一直往前的，不会出现时钟回拨的）
type TimeProvider interface {

	// GetTime 获取当前时间
	GetTime(ctx context.Context) (time.Time, error)
}
