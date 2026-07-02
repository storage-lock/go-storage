package storage

import (
	"context"
)

// ConnectionManager 把与Storage的连接的管理抽象为一个组件，属于比较底层的接口，用来适配上层的各种情况
// 比如上层可以是从DSN直接创建数据库连接，也可以是从一个已经存在的连接池中拿出来连接，甚至从已有的ORM、sqlx、sql.DB中复用连接
// 或者任何你想扩展的实现，总之它是一个带泛型的接口，你可以根据你的需求发挥想象力任意创造！
//
// === 职责边界（重要，避免抽象与实现语义不一致的陷阱）===
//
// 1. ConnectionManager 不承担分布式锁的互斥性责任。互斥性的唯一保证是：
//    (a) 存储服务端单语句的原子性（SQL 的 UPDATE...WHERE version=?、Mongo 的 UpdateOne 带 filter、
//        Redis 的 SET NX、DynamoDB 的 ConditionExpression 等）；
//    (b) CapabilityCAS 要求的线性一致性声明。
//    ConnectionManager 只负责"把连接/连接池引用交给 Storage"，不负责连接级独占。
//
// 2. Take/Return 的"借/还"语义对连接池型实现（*sql.DB、*mongo.Client）是 no-op：
//    这些类型本身就是并发安全的连接池，Take 返回池引用、Return 无资源可释放。
//    这是固有边界——*sql.DB 的语义就是池，无法强制单连接独占。Return 的存在是为
//    "真正单连接"实现（如独占 TCP 连接）预留的语义位，内置实现未用到。
//    ⚠️ 不要基于"Return 后连接可被回收/给别人"的假设在 Take/Return 之间做依赖连接级独占的逻辑。
//
// 3. Take 的初始化时机因实现而异：
//    - SQL 系列（DsnConnectionManager/MysqlConnectionManager）：每个 CAS 操作前后成对 Take/Return。
//    - MongoDB（MongoConnectionManager）：仅在 Storage.Init 时 Take 一次并缓存到 Storage 内部，
//      之后 CAS 路径直接用缓存引用，不再 Take/Return。这是 *mongo.Client 作为长生命周期连接池的特性。
//    两种模式都安全——连接池本身并发安全。但意味着 Shutdown ConnectionManager（如 client.Disconnect）
//    会影响 Storage 后续的 CAS，调用方须在确认无并发 Lock/Unlock 时再 Shutdown。
//
// 4. Shutdown 与并发的 Lock/Unlock 不保证安全：Shutdown 会关闭底层连接/连接池，
//    与并发的 CAS 操作竞态可能导致在已关闭连接上操作或 nil 解引用（如 MongoDB Close 置 nil）。
//    调用方须保证：Shutdown 时无任何并发的 Lock/Unlock/看门狗续租在途。
//    推荐顺序：先停所有锁的看门狗/UnLock → 再 StorageLockFactory.Shutdown（先关 Storage 再关 ConnectionManager）。
type ConnectionManager[Connection any] interface {

	// Name 连接提供器的名字，用于区分不同的连接提供器，连接器的名字必须指定不允许为空字符串
	Name() string

	// Take 获取一个往Storage的连接
	// 实现必须是并发安全的。失败时不应缓存错误（漏洞4：失败应可重试，不永久缓存 err）。
	Take(ctx context.Context) (Connection, error)

	// Return 使用完毕，把Storage的连接归还，用于在一些从连接池中拿连接使用完毕必须手动释放否则会资源泄露的场景下及时释放资源
	// 对连接池型实现（*sql.DB、*mongo.Client）此方法为 no-op（见接口文档职责边界第2条）。
	Return(ctx context.Context, connection Connection) error

	// Shutdown 把整个连接管理器关闭掉，彻底不用了，Storage Lock并不会调用这个方法，你应该在你的系统退出的时候调用此方法释放整个连接管理器使用到的资源
	// ⚠️ 不保证与并发的 Lock/Unlock 安全，调用方须确保无并发操作在途（见接口文档职责边界第4条）。
	Shutdown(ctx context.Context) error
}
