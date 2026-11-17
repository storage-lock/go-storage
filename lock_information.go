package storage

import (
	"encoding/json"
	"time"
)

// LockInformation 锁的相关信息，是要持久化保存到相关介质中的
type LockInformation struct {

	// 锁的ID
	LockId string `json:"lock_id"`

	// 当前时谁在持有这个锁，是一个全局唯一的ID
	OwnerId string `json:"owner_id"`

	// 锁的变更版本号，乐观锁避免CAS的ABA问题
	Version Version `json:"version"`

	// 锁被锁定了几次，是为了支持可重入锁，在释放锁的时候会根据加锁的次数来决定是否真正的释放锁还是就减少一次锁定次数
	LockCount int `json:"lock_count"`

	// 这个锁是从啥时候开始被OwnerId所持有的，用于判断持有锁的时间
	LockBeginTime time.Time `json:"lock_begin_time"`

	// 锁的owner持有此锁的租约过期时间，
	LeaseExpireTime time.Time `json:"lease_expire_time"`

	// ReadLock 标记当前锁是否为读锁模式。
	// false 或不设置：写锁/独占锁，语义与旧版本保持一致。
	// true：读锁/共享锁，允许多个不同 owner 同时持有，与写锁互斥。
	ReadLock bool `json:"read_lock,omitempty"`

	// ReadOwners 是共享读锁的每个持有者及其重入深度。
	// ReadLock 为 true 时 LockCount 等于各深度之和；写锁时为空。
	// 省略时按旧记录处理：整个 LockCount 归 OwnerId 一人。
	ReadOwners map[string]int `json:"read_owners,omitempty"`

	// WriteIntentOwner 是第一个排队写者。非空时新的读者不得进入，
	// 用来防止读者持续到来把写者饿死。写者拿到锁后清空。
	WriteIntentOwner string `json:"write_intent_owner,omitempty"`

	// WriteIntentSince 是写意图登记时的存储时间，供等待时长观测。
	WriteIntentSince time.Time `json:"write_intent_since,omitempty"`

	// Waiters 是正在排队的等待者，按登记顺序排列。
	// 开启公平获取时，这个顺序决定谁能拿到锁：排在前面的写者没退出之前，
	// 后面的调用方抢占会失败。登记失败等于没排队，调用方拿不到锁。
	Waiters []LockWaiter `json:"waiters,omitempty"`

	// Extra 保存由调用方附加的锁元数据。该字段是可选的，旧记录省略时视为空。
	Extra map[string]string `json:"extra,omitempty"`
}

// LockWaiter 是一个正在等待这把锁的调用方。
type LockWaiter struct {
	// OwnerId 是等待者的全局唯一标识。
	OwnerId string `json:"owner_id"`
	// Mode 是 "read" 或 "write"。
	Mode string `json:"mode"`
	// Since 是开始等待时的存储时间。
	Since time.Time `json:"since"`
	// Until 是这次排队登记的截止时间，与租约使用同一个时间源。
	// 零值表示旧记录没有截止时间，不能因为缺少该字段就把等待者删掉。
	// 非零且已经不晚于当前时间的登记视为等待者崩溃，其他实例可以移除。
	Until time.Time `json:"until,omitempty"`
}

// ReadDepth 返回 owner 在共享读锁上的重入深度。写锁或未持有时返回 0。
func (x *LockInformation) ReadDepth(owner string) int {
	if x == nil || !x.ReadLock {
		return 0
	}
	if len(x.ReadOwners) > 0 {
		return x.ReadOwners[owner]
	}
	if x.OwnerId == owner {
		return x.LockCount
	}
	return 0
}

// ReaderCount 返回共享读锁的不同持有者数量。
func (x *LockInformation) ReaderCount() int {
	if x == nil || !x.ReadLock || x.LockCount <= 0 {
		return 0
	}
	if len(x.ReadOwners) > 0 {
		return len(x.ReadOwners)
	}
	return 1
}

// AddReadOwner 为共享读锁增加 owner 的一层持有。
func (x *LockInformation) AddReadOwner(owner string) {
	if x.ReadOwners == nil {
		x.ReadOwners = map[string]int{}
		if x.OwnerId != "" && x.LockCount > 0 {
			x.ReadOwners[x.OwnerId] = x.LockCount
		}
	}
	x.ReadOwners[owner]++
	x.LockCount++
	x.OwnerId = owner
	x.ReadLock = true
}

// RemoveReadOwner 释放 owner 的一层读锁持有。owner 不持有时返回 false。
func (x *LockInformation) RemoveReadOwner(owner string) bool {
	if !x.ReadLock {
		return false
	}
	if len(x.ReadOwners) == 0 {
		if x.OwnerId != owner || x.LockCount <= 0 {
			return false
		}
		x.LockCount--
		return true
	}
	depth := x.ReadOwners[owner]
	if depth <= 0 {
		return false
	}
	if depth == 1 {
		delete(x.ReadOwners, owner)
	} else {
		x.ReadOwners[owner] = depth - 1
	}
	x.LockCount--
	return true
}

// HasWaiter 报告 owner 是否已以 mode 登记等待。
func (x *LockInformation) HasWaiter(owner, mode string) bool {
	for _, waiter := range x.Waiters {
		if waiter.OwnerId == owner && waiter.Mode == mode {
			return true
		}
	}
	return false
}

// LockInformationFromJsonString 从JSON字符串反序列化锁的信息
func LockInformationFromJsonString(jsonString string) (*LockInformation, error) {
	r := &LockInformation{}
	err := json.Unmarshal([]byte(jsonString), r)
	if err != nil {
		return nil, err
	}
	// JSON null 会成功反序列化成零值指针；调用方把 nil 当作记录不存在，必须显式返回 nil。
	if string(jsonString) == "null" {
		return nil, nil
	}
	return r, nil
}

// ToJsonString 把当前锁的信息序列化为JSON字符串以便存储到数据库
// 注意：此方法忽略序列化错误，如果 LockInformation 包含无法序列化的字段，将返回空字符串。
// 如需错误处理，请使用 ToJsonStringE。
func (x *LockInformation) ToJsonString() string {
	marshal, _ := json.Marshal(x)
	return string(marshal)
}

// ToJsonStringE 把当前锁的信息序列化为JSON字符串，返回 error
func (x *LockInformation) ToJsonStringE() (string, error) {
	marshal, err := json.Marshal(x)
	if err != nil {
		return "", err
	}
	return string(marshal), nil
}

// MustLockInformationFromJsonString 从JSON字符串反序列化锁的信息，失败时 panic
// 适用于确定 JSON 一定有效的场景，减少调用方的错误处理样板代码
func MustLockInformationFromJsonString(jsonString string) *LockInformation {
	r, err := LockInformationFromJsonString(jsonString)
	if err != nil {
		panic("failed to unmarshal LockInformation from JSON: " + err.Error())
	}
	return r
}

// IsExpired 判断锁是否已过期。
// 租约在时间恰好到达 LeaseExpireTime 时即失效，
// 与 StorageLock 获取/检查路径的 !now.Before(expire) 语义保持一致。
func (x *LockInformation) IsExpired(now time.Time) bool {
	return !now.Before(x.LeaseExpireTime)
}

// IsOwnedBy 判断锁是否由指定持有者持有
func (x *LockInformation) IsOwnedBy(ownerId string) bool {
	return x.OwnerId == ownerId
}

// IsValid 检查锁信息是否包含必要的有效字段
func (x *LockInformation) IsValid() bool {
	return x.LockId != "" && x.OwnerId != "" && x.Version > 0
}

// RemainingLease 计算锁的剩余租约时间
// 如果已经过期，返回负值
func (x *LockInformation) RemainingLease(now time.Time) time.Duration {
	return x.LeaseExpireTime.Sub(now)
}
