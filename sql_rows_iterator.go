package storage

import (
	"database/sql"
	"github.com/golang-infrastructure/go-iterator"
)

// SqlRowsIterator 用来把sql.Row包装为一个迭代器
type SqlRowsIterator struct {
	rows *sql.Rows
	scanErr error
}

var _ iterator.Iterator[*LockInformation] = &SqlRowsIterator{}

func NewSqlRowsIterator(rows *sql.Rows) *SqlRowsIterator {
	return &SqlRowsIterator{
		rows: rows,
	}
}

func (x *SqlRowsIterator) Next() bool {
	hasNext := x.rows.Next()
	if !hasNext {
		// 当遍历完的时候把Rows给关闭掉，防止连接泄露
		_ = x.rows.Close()
	}
	return hasNext
}

func (x *SqlRowsIterator) Value() *LockInformation {
	r, _ := x.ValueE()
	return r
}

// ValueE 返回当前行的锁信息和可能发生的错误
// 与 Value() 不同，此方法会传播 rows.Scan 和 JSON 反序列化错误
func (x *SqlRowsIterator) ValueE() (*LockInformation, error) {
	if x.scanErr != nil {
		return nil, x.scanErr
	}
	var jsonString string
	if err := x.rows.Scan(&jsonString); err != nil {
		x.scanErr = err
		return nil, err
	}
	result, err := LockInformationFromJsonString(jsonString)
	if err != nil {
		x.scanErr = err
		return nil, err
	}
	return result, nil
}

// Err 返回迭代过程中遇到的第一个错误
// 在调用 Next() 和 ValueE() 之后可以调用此方法检查错误
func (x *SqlRowsIterator) Err() error {
	return x.scanErr
}
