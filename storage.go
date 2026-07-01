package storage

import (
	"context"
	"github.com/golang-infrastructure/go-iterator"
)

// Version 表示一个锁的版本号，锁在每次被更改状态，比如每次被持有释放的时候都会增加版本号
type Version uint64

// StorageCapability 表示存储介质支持的能力
type StorageCapability string

const (
	// CapabilityCAS 表示存储介质支持 CAS（Compare-And-Set）操作
	// 这是分布式锁的必要条件之一，要求：
	//   - CreateWithVersion：如果 lockId 已存在则必须失败（唯一约束）
	//   - UpdateWithVersion：如果版本号不匹配则必须失败（条件更新）
	//   - DeleteWithVersion：如果版本号不匹配则必须失败（条件删除）
	// 如果存储不支持此能力，则无法保证锁的互斥性，不能用于生产环境
	CapabilityCAS StorageCapability = "cas"

	// CapabilityReliableTime 表示存储介质能够提供可靠的时间源
	// 这是分布式锁的必要条件之一，要求：
	//   - GetTime 返回的时间必须单调递增（不能出现时钟回拨）
	//   - 不同节点的时钟偏差必须远小于锁的 LeaseExpireAfter
	// 如果存储不支持此能力，则在时钟漂移较大的场景下可能破坏锁的互斥性
	// ⚠️ 注意：此能力可由"外部注入的 TimeProvider"替代满足，而非必须由 Storage 自身提供。
	// 例如对象存储（S3/OSS）、纯 HTTP 存储没有服务端时钟，无法自身满足此能力，
	// 但可通过在创建锁时注入外部 NTP 时间源（go-ntp-time-provider）来满足必要条件。
	// 详见 NewStorageLockWithOptions 的能力校验逻辑。
	CapabilityReliableTime StorageCapability = "reliable-time"

	// CapabilityAtomicDelete 表示存储介质支持原子的"条件删除"（DeleteWithVersion）
	// 这是一个"增强能力"而非"硬性必要条件"：
	//   - 支持此能力时，释放锁走 DeleteWithVersion 真正删除记录，语义最清晰
	//   - 不支持此能力时（如对象存储只有"条件 PUT" If-Match ETag，没有"条件 DELETE"），
	//     释放锁会降级为用 UpdateWithVersion 写入一个"墓碑"标记（LockCount=0、已释放），
	//     下次获取锁时走 lockExists → lockReleased 分支识别并抢占，互斥性依然由 CAS 保证
	// 因此，不支持此能力的存储介质仍可接入分布式锁，只是释放路径不同。
	// 真正不可降级的是 CapabilityCAS——墓碑写入本身就是一次 UpdateWithVersion 的原子 CAS。
	CapabilityAtomicDelete StorageCapability = "atomic-delete"
)

// Storage 表示一个存储介质的实现，要实现四个增删改查的方法和一个初始化的方法，以及能够提供Storage的日期，
// 因为在分布式系统中日期很重要，必须保证参与分布式运算的各个节点使用相同的时间
//
// # 必要条件
//
// 要正确实现分布式锁，Storage 实现必须满足以下两个必要条件：
//
// 1. CAS（Compare-And-Set）原子性
//
// 所有写操作（CreateWithVersion、UpdateWithVersion、DeleteWithVersion）必须是原子的：
//   - 检查条件（版本号/唯一性）和执行操作必须在同一次存储访问中完成
//   - 不允许"先读取检查再操作"的 Check-Then-Act 模式，因为两步之间存在竞态条件
//   - 不支持此能力的存储（如文件系统、对象存储）无法保证锁的互斥性
//
// 2. 可靠的时间源
//
// 时间必须满足：
//   - 单调递增：不能出现时钟回拨，否则可能导致未过期的锁被判为过期
//   - 节点间一致：所有参与分布式锁的节点看到的时间偏差应远小于 LeaseExpireAfter
//   - 推荐：使用存储服务器自身的时间（单实例数据库），或 NTP 同步良好的时间源
//
// 时间源不必由 Storage 自身提供。若 Storage 没有服务端时钟（对象存储、HTTP 存储），
// 可在创建锁时通过 options.TimeProvider 注入外部时间源（如 go-ntp-time-provider）。
// 只要"Storage 声明 CapabilityReliableTime"与"外部注入了 TimeProvider"满足其一即可。
//
// # 不满足必要条件的后果
//
// | 条件不满足 | 后果 |
// |-----------|------|
// | 不支持 CAS | 互斥性被破坏：多个节点可能同时持有同一把锁 |
// | 时间源不可靠（时钟回拨） | 互斥性被破坏：锁可能被提前释放，多个节点同时抢占 |
// | 时间源不可靠（时钟漂移） | 活性受影响：锁可能延迟释放，但互斥性不受影响 |
// | 不支持 CapabilityAtomicDelete | 无后果，仅释放路径降级为写墓碑标记，互斥性不受影响 |
type Storage interface {

	// GetName Storage的名称，用于区分不同的Storage的实现
	// Returns:
	//     string: Storage的名字，应该返回一个有辨识度并且简单易懂的名字，名字不能为空，否则认为是不合法的Storage实现
	GetName() string

	// Capabilities 声明此存储介质支持的能力
	// 存储实现必须返回自己支持的能力列表，至少应该包含 CapabilityCAS 和 CapabilityReliableTime
	// 如果缺少必要能力，上层代码应该给出警告或拒绝使用
	// Returns:
	//     []StorageCapability: 支持的能力列表
	Capabilities() []StorageCapability

	// Init 初始化操作，比如创建存储锁的表，需要支持多次调用，每次创建Storage的时候会调用此方法初始化
	// Params:
	//     ctx:
	// Returns:
	//    error: 初始化发生错误时返回对应的错误
	Init(ctx context.Context) error

	// UpdateWithVersion 如果存储的是指定版本的话，则将其更新
	// ⚠️ 必要条件：此操作必须是原子的 CAS 操作，检查版本号和更新必须在同一次存储访问中完成
	// 不允许先 Get 再 Update 的 Check-Then-Act 模式
	// Params:
	//     lockId 表示锁的ID
	//     exceptedValue 仅当老的值为这个时才进行更新
	//     newValue 更新为的新的值
	// Returns:
	//    error: 如果是版本不匹配，则返回错误 ErrVersionMiss，如果是其它类型的错误，依据情况自行返回
	UpdateWithVersion(ctx context.Context, lockId string, exceptedVersion, newVersion Version, lockInformation *LockInformation) error

	// CreateWithVersion 尝试将锁的信息插入到存储介质中，返回是否插入成功，底层存储的时候应该将锁的ID作为唯一ID，不能重复存储
	// 也就是说这个方法仅在锁不存在的时候才能执行成功，其它情况应该插入失败返回对应的错误
	// ⚠️ 必要条件：lockId 的唯一性检查必须是原子的，不能先查询是否存在再插入
	// Params:
	//     ctx:
	//     lockId:
	//     version:
	//     lockInformation:
	// Returns:
	//     error:
	CreateWithVersion(ctx context.Context, lockId string, version Version, lockInformation *LockInformation) error

	// DeleteWithVersion 如果锁的当前版本是期望的版本，则将其删除
	// ⚠️ 必要条件：此操作必须是原子的 CAS 操作，检查版本号和删除必须在同一次存储访问中完成
	// 如果是版本不匹配，则返回错误 ErrVersionMiss，如果是其它类型的错误，依据情况自行返回
	//
	DeleteWithVersion(ctx context.Context, lockId string, exceptedVersion Version, lockInformation *LockInformation) error

	// Get 获取锁之前存储的值，如果没有的话则返回空字符串，如果发生了错误则返回对应的错误信息，如果正常返回则是LockInformation的JSON字符串
	// Params:
	//     ctx: 用来做超时控制之类的
	//     lockId: 要查询的锁的ID
	// Returns:
	//    string:
	//    error:
	Get(ctx context.Context, lockId string) (string, error)

	// TimeProvider 分布式锁的话时间必须使用统一的时间，这个时间推荐是以Storage的时间为准，Storage要能够提供时间查询的功能
	// 这是因为分布式锁的算法需要根据时间来协调推进，而当时间不准确的时候算法可能会失效从而导致锁失效
	// ⚠️ 必要条件：返回的时间必须单调递增，不能出现时钟回拨
	// Params:
	//     ctx: 用来做超时控制之类的
	// Returns:
	//     time.Time: 返回Storage的当前时间
	//     error: 获取时间失败时则返回对应的错误
	TimeProvider

	// Close 关闭此存储介质，一般在系统退出释放资源的时候调用一下
	// Params:
	//     ctx: 用来做超时控制之类的
	// Returns:
	//     error: 如果关闭失败，则返回对应的错误
	Close(ctx context.Context) error

	// List 列出当前的Storage所持有的所有的锁的信息，因为数量可能会比较多，所以这里使用了一个迭代器模式
	// 虽然实际上可能用channel会更Golang一些，但是迭代器会比较易于实现并能够绑定一些内置的方法便于操作
	// Params:
	//     ctx: 用来做超时控制之类的
	// Returns:
	//     iterator.Iterator[*LockInformation]: 迭代器用来承载当前所有的锁
	//     error: 如果列出失败，则返回对应类型的错误
	List(ctx context.Context) (iterator.Iterator[*LockInformation], error)
}
