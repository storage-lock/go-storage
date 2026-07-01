package storage

// HasCapability 检查存储实现是否支持指定的能力
func HasCapability(s Storage, capability StorageCapability) bool {
	for _, c := range s.Capabilities() {
		if c == capability {
			return true
		}
	}
	return false
}

// MustCapabilities 分布式锁正常运行所必须的能力列表（仅从 Storage 自身能力角度）
//
// 注意 CapabilityReliableTime 在这里仍是"必要"，但这只针对"由 Storage 自身提供时间"的情形。
// 在 go-storage-lock 层（NewStorageLockWithOptions）会做更宽松的判断：
// 若 Storage 缺少 CapabilityReliableTime，但用户通过 options.TimeProvider 注入了外部可靠时间源
// （如 NTP），则视为该必要条件已满足——这正是对象存储、HTTP 存储等无服务端时钟介质接入的关键。
//
// CapabilityAtomicDelete 不在此列表中：它是增强能力，缺失时释放路径降级为写墓碑，互斥性不受影响。
var MustCapabilities = []StorageCapability{
	CapabilityCAS,
	CapabilityReliableTime,
}

// CheckCapabilities 检查存储实现是否满足分布式锁的必要条件
// 返回缺失的能力列表，如果返回空切片则表示满足所有必要条件
func CheckCapabilities(s Storage) []StorageCapability {
	capabilities := s.Capabilities()
	missing := make([]StorageCapability, 0)
	for _, must := range MustCapabilities {
		found := false
		for _, c := range capabilities {
			if c == must {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, must)
		}
	}
	return missing
}

// SupportsAtomicDelete 判断存储是否支持原子的条件删除（DeleteWithVersion）
// 不支持时，释放锁需降级为 UpdateWithVersion 写入墓碑标记。互斥性不受影响。
func SupportsAtomicDelete(s Storage) bool {
	return HasCapability(s, CapabilityAtomicDelete)
}
