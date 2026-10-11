package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLockInformationFromJsonString_ValidJson(t *testing.T) {
	now := time.Now()
	info := &LockInformation{
		LockId:          "lock-1",
		OwnerId:         "owner-1",
		Version:         1,
		LockCount:       3,
		LockBeginTime:   now,
		LeaseExpireTime: now.Add(time.Hour),
	}

	jsonStr := info.ToJsonString()
	result, err := LockInformationFromJsonString(jsonStr)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "lock-1", result.LockId)
	assert.Equal(t, "owner-1", result.OwnerId)
	assert.Equal(t, Version(1), result.Version)
	assert.Equal(t, 3, result.LockCount)
	assert.Equal(t, now.Unix(), result.LockBeginTime.Unix())
	assert.Equal(t, now.Add(time.Hour).Unix(), result.LeaseExpireTime.Unix())
}

func TestLockInformationFromJsonString_EmptyString(t *testing.T) {
	result, err := LockInformationFromJsonString("")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestLockInformationFromJsonString_InvalidJson(t *testing.T) {
	result, err := LockInformationFromJsonString("not-valid-json")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestLockInformationFromJsonString_MalformedJson(t *testing.T) {
	result, err := LockInformationFromJsonString(`{"lock_id": "test"`)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestToJsonString_ValidInfo(t *testing.T) {
	now := time.Now()
	info := &LockInformation{
		LockId:          "lock-1",
		OwnerId:         "owner-1",
		Version:         1,
		LockCount:       3,
		LockBeginTime:   now,
		LeaseExpireTime: now.Add(time.Hour),
	}

	jsonStr := info.ToJsonString()
	assert.NotEmpty(t, jsonStr)

	// 验证 JSON 包含关键字段
	assert.Contains(t, jsonStr, "lock-1")
	assert.Contains(t, jsonStr, "owner-1")
	assert.Contains(t, jsonStr, `"version":1`)
	assert.Contains(t, jsonStr, `"lock_count":3`)
}

func TestToJsonString_EmptyInfo(t *testing.T) {
	info := &LockInformation{}
	jsonStr := info.ToJsonString()
	assert.NotEmpty(t, jsonStr)
	assert.Contains(t, jsonStr, `""`)
}

func TestToJsonString_SpecialCharacters(t *testing.T) {
	info := &LockInformation{
		LockId:    `lock"with"quotes`,
		OwnerId:   `owner<with>brackets`,
		Version:   1,
		LockCount: 1,
	}

	jsonStr := info.ToJsonString()
	assert.NotEmpty(t, jsonStr)

	result, err := LockInformationFromJsonString(jsonStr)
	assert.NoError(t, err)
	assert.Equal(t, `lock"with"quotes`, result.LockId)
	assert.Equal(t, `owner<with>brackets`, result.OwnerId)
}

func TestToJsonString_ZeroTime(t *testing.T) {
	info := &LockInformation{
		LockId:          "lock-1",
		OwnerId:         "owner-1",
		Version:         1,
		LockCount:       1,
		LockBeginTime:   time.Time{},
		LeaseExpireTime: time.Time{},
	}

	jsonStr := info.ToJsonString()
	assert.NotEmpty(t, jsonStr)

	result, err := LockInformationFromJsonString(jsonStr)
	assert.NoError(t, err)
	assert.True(t, result.LockBeginTime.IsZero())
	assert.True(t, result.LeaseExpireTime.IsZero())
}

func TestToJsonString_UnicodeCharacters(t *testing.T) {
	info := &LockInformation{
		LockId:    "锁-中文",
		OwnerId:   "所有者-日本語",
		Version:   1,
		LockCount: 1,
	}

	jsonStr := info.ToJsonString()
	assert.NotEmpty(t, jsonStr)
	assert.Contains(t, jsonStr, "锁-中文")
	assert.Contains(t, jsonStr, "所有者-日本語")

	result, err := LockInformationFromJsonString(jsonStr)
	assert.NoError(t, err)
	assert.Equal(t, "锁-中文", result.LockId)
	assert.Equal(t, "所有者-日本語", result.OwnerId)
}

func TestToJsonString_RoundTrip(t *testing.T) {
	now := time.Now()
	original := &LockInformation{
		LockId:          "round-trip-lock",
		OwnerId:         "round-trip-owner",
		Version:         42,
		LockCount:       5,
		LockBeginTime:   now,
		LeaseExpireTime: now.Add(30 * time.Minute),
	}

	jsonStr := original.ToJsonString()
	restored, err := LockInformationFromJsonString(jsonStr)
	assert.NoError(t, err)
	assert.NotNil(t, restored)
	assert.Equal(t, original.LockId, restored.LockId)
	assert.Equal(t, original.OwnerId, restored.OwnerId)
	assert.Equal(t, original.Version, restored.Version)
	assert.Equal(t, original.LockCount, restored.LockCount)
	assert.Equal(t, original.LockBeginTime.Unix(), restored.LockBeginTime.Unix())
	assert.Equal(t, original.LeaseExpireTime.Unix(), restored.LeaseExpireTime.Unix())
}

func TestToJsonStringE_ValidInfo(t *testing.T) {
	now := time.Now()
	info := &LockInformation{
		LockId:          "lock-1",
		OwnerId:         "owner-1",
		Version:         1,
		LockCount:       3,
		LockBeginTime:   now,
		LeaseExpireTime: now.Add(time.Hour),
	}

	jsonStr, err := info.ToJsonStringE()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonStr)
	assert.Equal(t, info.ToJsonString(), jsonStr)
	assert.Contains(t, jsonStr, "lock-1")
}

func TestToJsonStringE_EmptyInfo(t *testing.T) {
	info := &LockInformation{}
	jsonStr, err := info.ToJsonStringE()
	assert.NoError(t, err)
	assert.Equal(t, info.ToJsonString(), jsonStr)
}

func TestMustLockInformationFromJsonString_ValidJson(t *testing.T) {
	info := &LockInformation{
		LockId:  "must-lock",
		OwnerId: "must-owner",
		Version: 1,
	}

	jsonStr := info.ToJsonString()
	result := MustLockInformationFromJsonString(jsonStr)
	assert.NotNil(t, result)
	assert.Equal(t, "must-lock", result.LockId)
}

func TestMustLockInformationFromJsonString_InvalidJson(t *testing.T) {
	assert.Panics(t, func() {
		MustLockInformationFromJsonString("not-valid-json")
	})
}

func TestLockInformation_IsExpired(t *testing.T) {
	now := time.Now()
	info := &LockInformation{
		LockId:          "lock-1",
		OwnerId:         "owner-1",
		Version:         1,
		LeaseExpireTime: now.Add(time.Hour),
	}

	assert.False(t, info.IsExpired(now))
	assert.False(t, info.IsExpired(now.Add(30*time.Minute)))
	assert.True(t, info.IsExpired(info.LeaseExpireTime))
	assert.True(t, info.IsExpired(now.Add(2*time.Hour)))
}

func TestLockInformation_IsOwnedBy(t *testing.T) {
	info := &LockInformation{
		LockId:  "lock-1",
		OwnerId: "owner-1",
		Version: 1,
	}

	assert.True(t, info.IsOwnedBy("owner-1"))
	assert.False(t, info.IsOwnedBy("owner-2"))
	assert.False(t, info.IsOwnedBy(""))
}

func TestLockInformation_IsValid(t *testing.T) {
	assert.True(t, (&LockInformation{
		LockId:  "lock-1",
		OwnerId: "owner-1",
		Version: 1,
	}).IsValid())

	assert.False(t, (&LockInformation{
		LockId:  "",
		OwnerId: "owner-1",
		Version: 1,
	}).IsValid())

	assert.False(t, (&LockInformation{
		LockId:  "lock-1",
		OwnerId: "",
		Version: 1,
	}).IsValid())

	assert.False(t, (&LockInformation{
		LockId:  "lock-1",
		OwnerId: "owner-1",
		Version: 0,
	}).IsValid())

	assert.False(t, (&LockInformation{}).IsValid())
}

func TestLockInformation_RemainingLease(t *testing.T) {
	now := time.Now()
	info := &LockInformation{
		LockId:          "lock-1",
		OwnerId:         "owner-1",
		Version:         1,
		LeaseExpireTime: now.Add(2 * time.Hour),
	}

	remaining := info.RemainingLease(now)
	assert.Equal(t, 2*time.Hour, remaining)

	remaining = info.RemainingLease(now.Add(30 * time.Minute))
	assert.Equal(t, 90*time.Minute, remaining)

	remaining = info.RemainingLease(now.Add(3 * time.Hour))
	assert.True(t, remaining < 0)
	assert.Equal(t, -time.Hour, remaining)
}
