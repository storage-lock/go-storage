package storage

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixedSqlDBConnectionManager_Name_ReturnsConstant(t *testing.T) {
	cm := NewFixedSqlDBConnectionManager(nil)
	assert.Equal(t, FixedSqlDBConnectionManagerName, cm.Name())
}

func TestFixedSqlDBConnectionManager_Name_IsFixedString(t *testing.T) {
	cm := NewFixedSqlDBConnectionManager(nil)
	assert.Equal(t, "fixed-sql-db-connection-manager", cm.Name())
}

func TestFixedSqlDBConnectionManager_Take_ReturnsNilWhenStoredNil(t *testing.T) {
	cm := NewFixedSqlDBConnectionManager(nil)
	db, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, db)
}

func TestFixedSqlDBConnectionManager_Take_ReturnsSameInstance(t *testing.T) {
	var storedDB *sql.DB
	cm := NewFixedSqlDBConnectionManager(storedDB)
	db1, _ := cm.Take(context.Background())
	db2, _ := cm.Take(context.Background())
	assert.Equal(t, db1, db2)
	assert.Equal(t, storedDB, db1)
}

func TestFixedSqlDBConnectionManager_Return_NoError(t *testing.T) {
	cm := NewFixedSqlDBConnectionManager(nil)
	err := cm.Return(context.Background(), nil)
	assert.NoError(t, err)
}

func TestFixedSqlDBConnectionManager_Return_AlwaysSucceeds(t *testing.T) {
	var storedDB *sql.DB
	cm := NewFixedSqlDBConnectionManager(storedDB)
	err := cm.Return(context.Background(), nil)
	assert.NoError(t, err)
}

func TestFixedSqlDBConnectionManager_Shutdown_NilDb(t *testing.T) {
	cm := NewFixedSqlDBConnectionManager(nil)
	err := cm.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestFixedSqlDBConnectionManager_Shutdown_WithDb(t *testing.T) {
	// 使用 sql.Open 创建一个真实的 *sql.DB 实例进行测试
	// 这里使用 ":memory:" 作为 DSN，创建一个内存数据库连接
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Skipf("跳过测试：无法打开 sqlite3 连接: %v", err)
	}
	defer db.Close()

	cm := NewFixedSqlDBConnectionManager(db)
	// Shutdown 应该能够关闭底层的 db
	err = cm.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestFixedSqlDBConnectionManager_Shutdown_AfterShutdownCannotUseDb(t *testing.T) {
	// 使用 sql.Open 创建一个真实的 *sql.DB 实例进行测试
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Skipf("跳过测试：无法打开 sqlite3 连接: %v", err)
	}

	cm := NewFixedSqlDBConnectionManager(db)
	// Shutdown 应该能够关闭底层的 db
	err = cm.Shutdown(context.Background())
	assert.NoError(t, err)

	// 验证 db 已被关闭，再次操作应该报错
	_, err = db.Query("SELECT 1")
	assert.Error(t, err)
}
