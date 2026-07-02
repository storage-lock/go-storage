package storage

import (
	"context"
	"database/sql"
	"sync"
)

// DsnConnectionManager 从DSN维持数据库连接，用于只有一个DSN的情况下创建连接管理器
type DsnConnectionManager struct {

	// TODO 2023-8-4 01:39:41 这几个字段单独抽取为一个ConnectionManager
	//// 主机的名字
	//Host string
	//
	//// 主机的端口
	//Port uint
	//
	//// 用户名
	//User string
	//
	//// 密码
	//Passwd string

	driverName string

	// DSN
	// "root:123456@tcp(127.0.0.1:4000)/test?charset=utf8mb4"
	DSN string

	// 初始化好的数据库实例
	db *sql.DB

	// initMu 保护下面的初始化状态。原实现用 sync.Once，但 Once 一旦执行（哪怕失败）就不再重试，
	// 导致 sql.Open 首次失败后整个 ConnectionManager 永久不可用（漏洞4/liveness）。
	// 改为 mutex + initialized 标志：成功后缓存 db 不再重复 Open；失败不缓存 err，下次 Take 可重试。
	initMu      sync.Mutex
	initialized bool
}

var _ ConnectionManager[*sql.DB] = &DsnConnectionManager{}

func NewDsnConnectionManager(driverName, dsn string) *DsnConnectionManager {
	return &DsnConnectionManager{
		driverName: driverName,
		DSN:        dsn,
	}
}

//// NewSQLStorageConnectionGetter 从服务器属性创建数据库连接
//func NewSQLStorageConnectionGetter(host string, port uint, user, passwd string) *DsnConnectionManager {
//	return &DsnConnectionManager{
//		Host:   host,
//		Port:   port,
//		User:   user,
//		Passwd: passwd,
//	}
//}

const DSNConnectionManagerName = "dsn-connection-manager"

func (x *DsnConnectionManager) Name() string {
	return DSNConnectionManagerName
}

// Take 获取到数据库的连接
//
// 漏洞4 修复：原用 sync.Once，sql.Open 首次失败后 err 永久缓存，后续 Take 永远返回同一 err，
// 无法重试（对 DNS 抖动/网络瞬断等临时性故障不友好，liveness 问题，不破互斥性——拿不到 db
// 则 CAS 根本无法发起）。改为 mutex + initialized 标志：成功后缓存 db 不再重复 Open；
// 失败不缓存 err，下次 Take 可重试。
func (x *DsnConnectionManager) Take(ctx context.Context) (*sql.DB, error) {
	x.initMu.Lock()
	defer x.initMu.Unlock()
	if x.initialized {
		return x.db, nil
	}
	db, err := sql.Open(x.driverName, x.DSN)
	if err != nil {
		// 失败不缓存，下次 Take 可重试
		return nil, err
	}
	x.db = db
	x.initialized = true
	return x.db, nil
}

func (x *DsnConnectionManager) Return(ctx context.Context, db *sql.DB) error {
	// 归还的时候啥也不用做
	return nil
}

func (x *DsnConnectionManager) Shutdown(ctx context.Context) error {
	// 在连接池被关闭的时候需要把当前持有的连接关闭掉
	x.initMu.Lock()
	db := x.db
	x.initMu.Unlock()
	if db != nil {
		return db.Close()
	}
	return nil
}
