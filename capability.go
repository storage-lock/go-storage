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

// MustCapabilities 分布式锁正常运行所必须的能力列表
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
