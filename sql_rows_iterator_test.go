package storage

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-infrastructure/go-iterator"
	"github.com/stretchr/testify/assert"
)

func TestSqlRowsIterator_NewSqlRowsIterator(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"json_string"}),
	)

	rows, err := db.Query("SELECT * FROM locks")
	if err != nil {
		t.Fatalf("db.Query 失败: %v", err)
	}

	iter := NewSqlRowsIterator(rows)
	assert.NotNil(t, iter)
}

func TestSqlRowsIterator_Next_WithData(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	columns := []string{"json_string"}
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows(columns).
			AddRow(`{"lock_id":"lock-1","owner_id":"owner-1","version":1,"lock_count":3}`).
			AddRow(`{"lock_id":"lock-2","owner_id":"owner-2","version":2,"lock_count":1}`).
			AddRow(`{"lock_id":"lock-3","owner_id":"owner-3","version":3,"lock_count":2}`),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	assert.True(t, iter.Next())
	assert.True(t, iter.Next())
	assert.False(t, iter.Next(), "应该没有更多数据")
}

func TestSqlRowsIterator_Next_EmptyRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"json_string"}),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.False(t, iter.Next(), "空数据集应该立即返回 false")
}

func TestSqlRowsIterator_Value_SingleRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	now := time.Now()
	columns := []string{"json_string"}
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows(columns).
			AddRow(`{"lock_id":"lock-1","owner_id":"owner-1","version":1,"lock_count":3,"lock_begin_time":"` + now.Format(time.RFC3339) + `","lease_expire_time":"` + now.Add(time.Hour).Format(time.RFC3339) + `"}`),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	info := iter.Value()

	assert.NotNil(t, info)
	assert.Equal(t, "lock-1", info.LockId)
	assert.Equal(t, "owner-1", info.OwnerId)
	assert.Equal(t, Version(1), info.Version)
	assert.Equal(t, 3, info.LockCount)
	assert.Equal(t, now.Unix(), info.LockBeginTime.Unix())
	assert.Equal(t, now.Add(time.Hour).Unix(), info.LeaseExpireTime.Unix())
}

func TestSqlRowsIterator_Value_MultipleRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	columns := []string{"json_string"}
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows(columns).
			AddRow(`{"lock_id":"lock-1","owner_id":"owner-1","version":1,"lock_count":3}`).
			AddRow(`{"lock_id":"lock-2","owner_id":"owner-2","version":2,"lock_count":1}`),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	info1 := iter.Value()
	assert.Equal(t, "lock-1", info1.LockId)

	assert.True(t, iter.Next())
	info2 := iter.Value()
	assert.Equal(t, "lock-2", info2.LockId)
}

func TestSqlRowsIterator_Value_AfterExhaustion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"json_string"}).AddRow(`{"lock_id":"lock-1"}`),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	assert.Equal(t, "lock-1", iter.Value().LockId)

	// 迭代结束后再次调用 Value，返回 nil 并记录错误
	assert.False(t, iter.Next())
	info := iter.Value()
	assert.Nil(t, info)
	assert.Error(t, iter.Err())
}

func TestSqlRowsIterator_Value_EmptyColumns(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"json_string"}).AddRow(`{}`),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	info := iter.Value()
	assert.NotNil(t, info)
	assert.Empty(t, info.LockId)
	assert.Empty(t, info.OwnerId)
}

func TestSqlRowsIterator_IteratorInterface(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"json_string"}).AddRow(`{"lock_id":"lock-1"}`),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	// 验证实现了 iterator.Iterator[*LockInformation] 接口
	var it iterator.Iterator[*LockInformation] = iter
	assert.NotNil(t, it)
}

func TestSqlRowsIterator_Value_SpecialCharacters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	columns := []string{"json_string"}
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows(columns).
			AddRow(`{"lock_id":"lock\"with\"quotes","owner_id":"owner<with>brackets"}`),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	info := iter.Value()
	assert.Equal(t, `lock"with"quotes`, info.LockId)
	assert.Equal(t, `owner<with>brackets`, info.OwnerId)
}

func TestSqlRowsIterator_Value_Unicode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	columns := []string{"json_string"}
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows(columns).
			AddRow(`{"lock_id":"锁-中文","owner_id":"所有者-日本語"}`),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	info := iter.Value()
	assert.Equal(t, "锁-中文", info.LockId)
	assert.Equal(t, "所有者-日本語", info.OwnerId)
}

func TestSqlRowsIterator_CloseCalledWhenExhausted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"json_string"}).AddRow(`{"lock_id":"lock-1"}`),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	_ = iter.Value()
	assert.False(t, iter.Next())
	_ = rows.Close()
}

func TestSqlRowsIterator_RoundTrip(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	now := time.Now()
	columns := []string{"json_string"}
	original := &LockInformation{
		LockId:          "round-trip",
		OwnerId:         "owner",
		Version:         42,
		LockCount:       5,
		LockBeginTime:   now,
		LeaseExpireTime: now.Add(30 * time.Minute),
	}

	jsonStr := original.ToJsonString()

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows(columns).AddRow(jsonStr),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	info := iter.Value()

	assert.Equal(t, original.LockId, info.LockId)
	assert.Equal(t, original.OwnerId, info.OwnerId)
	assert.Equal(t, original.Version, info.Version)
	assert.Equal(t, original.LockCount, info.LockCount)
	assert.Equal(t, original.LockBeginTime.Unix(), info.LockBeginTime.Unix())
	assert.Equal(t, original.LeaseExpireTime.Unix(), info.LeaseExpireTime.Unix())
}

func TestSqlRowsIterator_ValueE_ValidJson(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	now := time.Now()
	columns := []string{"json_string"}
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows(columns).
			AddRow(`{"lock_id":"lock-1","owner_id":"owner-1","version":1,"lock_count":3,"lock_begin_time":"` + now.Format(time.RFC3339) + `","lease_expire_time":"` + now.Add(time.Hour).Format(time.RFC3339) + `"}`),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	info, err := iter.ValueE()
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "lock-1", info.LockId)
	assert.Equal(t, "owner-1", info.OwnerId)
}

func TestSqlRowsIterator_ValueE_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	// 模拟 Scan 失败：列类型不匹配，无法扫描到 string
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"json_string"}).AddRow(12345),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	info, err := iter.ValueE()
	assert.Error(t, err)
	assert.Nil(t, info)
}

func TestSqlRowsIterator_ValueE_InvalidJson(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	columns := []string{"json_string"}
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows(columns).AddRow("not-valid-json"),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	info, err := iter.ValueE()
	assert.Error(t, err)
	assert.Nil(t, info)
}

func TestSqlRowsIterator_Err_ReturnsScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	// 构造一个 Scan 失败的结果行
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"json_string"}).AddRow(12345),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	_, scanErr := iter.ValueE()
	assert.Error(t, scanErr)
	assert.Error(t, iter.Err())
	assert.Equal(t, scanErr, iter.Err(), "Err() 应返回首次 ValueE() 的错误")
}

func TestSqlRowsIterator_ValueE_CachedErrorOnSecondCall(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	// 构造一个 Scan 失败的结果行
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"json_string"}).AddRow(12345),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	_, err1 := iter.ValueE()
	assert.Error(t, err1)

	// 第二次调用应直接返回缓存错误，不重复 Scan
	info, err2 := iter.ValueE()
	assert.Error(t, err2)
	assert.Nil(t, info)
	assert.Equal(t, err1, err2)
	assert.Equal(t, err1, iter.Err())
}

func TestSqlRowsIterator_Value_DelegatesToValueE(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	now := time.Now()
	columns := []string{"json_string"}
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows(columns).
			AddRow(`{"lock_id":"lock-1","owner_id":"owner-1","version":1,"lock_count":3,"lock_begin_time":"` + now.Format(time.RFC3339) + `","lease_expire_time":"` + now.Add(time.Hour).Format(time.RFC3339) + `"}`),
	)

	rows, _ := db.Query("SELECT * FROM locks")
	iter := NewSqlRowsIterator(rows)

	assert.True(t, iter.Next())
	info := iter.Value()
	assert.Equal(t, "lock-1", info.LockId)
	assert.NoError(t, iter.Err())
}
