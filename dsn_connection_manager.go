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
	db   *sql.DB
	err  error
	once sync.Once
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
func (x *DsnConnectionManager) Take(ctx context.Context) (*sql.DB, error) {
	x.once.Do(func() {
		db, err := sql.Open(x.driverName, x.DSN)
		if err != nil {
			x.err = err
			return
		}
		x.db = db
	})
	return x.db, x.err
}

func (x *DsnConnectionManager) Return(ctx context.Context, db *sql.DB) error {
	// 归还的时候啥也不用做
	return nil
}

func (x *DsnConnectionManager) Shutdown(ctx context.Context) error {
	// 在连接池被关闭的时候需要把当前持有的连接关闭掉
	if x.err != nil {
		return x.err
	}
	if x.db != nil {
		return x.db.Close()
	}
	return nil
}
