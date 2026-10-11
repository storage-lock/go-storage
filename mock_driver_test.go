package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockDriver 实现 database/sql/driver.Driver 接口
type mockDriver struct{}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

func (d *mockDriver) OpenConnector(name string) (driver.Connector, error) {
	return &mockConnector{driver: &mockDriver{}}, nil
}

// mockConnector 实现 driver.Connector 接口
type mockConnector struct {
	driver driver.Driver
}

func (c *mockConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return c.driver.Open("")
}

func (c *mockConnector) Driver() driver.Driver {
	return c.driver
}

// mockConn 实现 driver.Conn 接口
type mockConn struct {
	mu     sync.Mutex
	closed bool
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return &mockStmt{}, nil
}

func (c *mockConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *mockConn) Begin() (driver.Tx, error) {
	return &mockTx{}, nil
}

func (c *mockConn) Ping(ctx context.Context) error {
	return nil
}

func (c *mockConn) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// mockStmt 实现 driver.Stmt 接口
type mockStmt struct{}

func (s *mockStmt) Close() error {
	return nil
}

func (s *mockStmt) NumInput() int {
	return 0
}

func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &mockResult{rowsAffected: 1}, nil
}

func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockRows{}, nil
}

// mockResult 实现 driver.Result 接口
type mockResult struct {
	rowsAffected int64
}

func (r *mockResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (r *mockResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// mockRows 实现 driver.Rows 接口
type mockRows struct {
	index int
	data  [][]string
}

func (r *mockRows) Columns() []string {
	return []string{"json_string"}
}

func (r *mockRows) Close() error {
	return nil
}

func (r *mockRows) Next(dest []driver.Value) error {
	r.index++
	if r.index >= len(r.data) {
		return io.EOF
	}
	for i, v := range r.data[r.index] {
		dest[i] = v
	}
	return nil
}

// mockTx 实现 driver.Tx 接口
type mockTx struct{}

func (tx *mockTx) Commit() error {
	return nil
}

func (tx *mockTx) Rollback() error {
	return nil
}

// init 注册 mock 驱动
func init() {
	sql.Register("mock-driver", &mockDriver{})
}

func TestDsnConnectionManager_Take_SuccessWithValidDriver(t *testing.T) {
	cm := NewDsnConnectionManager("mock-driver", "mock-dsn")

	db, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// 验证 db 被缓存，第二次 Take 返回同一个实例
	db2, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, db, db2)
	assert.True(t, cm.initialized)
}

func TestDsnConnectionManager_Shutdown_ClosesNonNilDb(t *testing.T) {
	cm := NewDsnConnectionManager("mock-driver", "mock-dsn")

	db, err := cm.Take(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, db)

	err = cm.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestFixedSqlDBConnectionManager_Shutdown_ClosesNonNilDb(t *testing.T) {
	db, err := sql.Open("mock-driver", "mock-dsn")
	if err != nil {
		t.Fatalf("打开 mock 数据库失败: %v", err)
	}

	cm := NewFixedSqlDBConnectionManager(db)
	err = cm.Shutdown(context.Background())
	assert.NoError(t, err)
}

// mockTimeDriver 用于测试 GetTime 方法
type mockTimeDriver struct{}

func (d *mockTimeDriver) Open(name string) (driver.Conn, error) {
	return &mockTimeConn{}, nil
}

func (d *mockTimeDriver) OpenConnector(name string) (driver.Connector, error) {
	return &mockTimeConnector{driver: &mockTimeDriver{}}, nil
}

type mockTimeConnector struct {
	driver driver.Driver
}

func (c *mockTimeConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return c.driver.Open("")
}

func (c *mockTimeConnector) Driver() driver.Driver {
	return c.driver
}

type mockTimeConn struct{}

func (c *mockTimeConn) Prepare(query string) (driver.Stmt, error) {
	return &mockTimeStmt{}, nil
}

func (c *mockTimeConn) Close() error {
	return nil
}

func (c *mockTimeConn) Begin() (driver.Tx, error) {
	return &mockTx{}, nil
}

func (c *mockTimeConn) Ping(ctx context.Context) error {
	return nil
}

type mockTimeStmt struct{}

func (s *mockTimeStmt) Close() error {
	return nil
}

func (s *mockTimeStmt) NumInput() int {
	return 0
}

func (s *mockTimeStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &mockResult{rowsAffected: 1}, nil
}

func (s *mockTimeStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockTimeRows{}, nil
}

type mockTimeRows struct {
	index int
}

func (r *mockTimeRows) Columns() []string {
	return []string{"timestamp"}
}

func (r *mockTimeRows) Close() error {
	return nil
}

func (r *mockTimeRows) Next(dest []driver.Value) error {
	r.index++
	if r.index > 0 {
		return io.EOF
	}
	dest[0] = time.Now().Unix()
	return nil
}

// mockTimeProvider 用于测试 TimeProvider
type mockTimeProvider struct {
	time time.Time
	err  error
}

func (m *mockTimeProvider) GetTime(ctx context.Context) (time.Time, error) {
	return m.time, m.err
}

func TestSqlBasedStorage_GetTime_UsesTimeProvider(t *testing.T) {
	// 这个测试验证 GetTime 方法能够正确使用 TimeProvider
	provider := &mockTimeProvider{
		time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	result, err := provider.GetTime(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), result)
}

func TestSqlBasedStorage_GetTime_HandlesProviderError(t *testing.T) {
	provider := &mockTimeProvider{
		err: fmt.Errorf("time provider error"),
	}

	_, err := provider.GetTime(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "time provider error")
}
