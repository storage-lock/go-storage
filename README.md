# Storage的接口定义

# 一、这是什么

Storage Lock中的Lock是要存储在Storage上的，这个仓库就是定义了Storage的相关实现规范。

# 二、安装依赖

```bash
go get -u github.com/storage-lock/go-storage
```

# 三、组件简介

## LockInformation

LockInformation用于表示锁相关的信息，包括锁当前被谁持有，如果是重入锁的话当前的深度是多少，锁被获取的事件是啥时候，锁过期的时间是啥时候，锁每次被修改的时候版本号都会增加，那么锁的当前的版本号也会记录在LockInformation中。

```go
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
}
```

## Storage

定义了Storage的相关规范与功能，Storage的作用是用于把上面定义的锁的信息持久化存储。

```go
package storage

import (
	"context"
	"github.com/golang-infrastructure/go-iterator"
)

// Version 表示一个锁的版本号，锁在每次被更改状态，比如每次被持有释放的时候都会增加版本号
type Version uint64

// Storage 表示一个存储介质的实现，要实现四个增删改查的方法和一个初始化的方法，以及能够提供Storage的日期，
// 因为在分布式系统中日期很重要，必须保证参与分布式运算的各个节点使用相同的时间
type Storage interface {

	// GetName Storage的名称，用于区分不同的Storage的实现
	// Returns:
	//     string: Storage的名字，应该返回一个有辨识度并且简单易懂的名字，名字不能为空，否则认为是不合法的Storage实现
	GetName() string

	// Init 初始化操作，比如创建存储锁的表，需要支持多次调用，每次创建Storage的时候会调用此方法初始化
	// Params:
	//     ctx:
	// Returns:
	//    error: 初始化发生错误时返回对应的错误
	Init(ctx context.Context) error

	// UpdateWithVersion 如果存储的是指定版本的话，则将其更新
	// Params:
	//     lockId 表示锁的ID
	//     exceptedValue 仅当老的值为这个时才进行更新
	//     newValue 更新为的新的值
	// Returns:
	//    error: 如果是版本不匹配，则返回错误 ErrVersionMiss，如果是其它类型的错误，依据情况自行返回
	UpdateWithVersion(ctx context.Context, lockId string, exceptedVersion, newVersion Version, lockInformation *LockInformation) error

	// InsertWithVersion 尝试将锁的信息插入到存储介质中，返回是否插入成功，底层存储的时候应该将锁的ID作为唯一ID，不能重复存储
	// 也就是说这个方法仅在锁不存在的时候才能执行成功，其它情况应该插入失败返回对应的错误
	// Params:
	//     ctx:
	//     lockId:
	//     version:
	//     lockInformation:
	// Returns:
	//     error:
	InsertWithVersion(ctx context.Context, lockId string, version Version, lockInformation *LockInformation) error

	// DeleteWithVersion 如果锁的当前版本是期望的版本，则将其删除
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
	// TODO 2023-5-15 01:48:19 基于实例的时间在分布式数据库中可能会失效，单实例没问题
	// TODO 2023-8-3 21:53:35 在文档中用实际例子演示分布式情况下可能会存在的问题
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
```

## TimeProvider

分布式情况下涉及到不同节点的协同，它们之间的时间一致性很重要，Storage要实现TimeProvider接口，在分布式算法中作为时间源提供时间。

```go
// TimeProvider 能够提供时间的时间源，可以从数据库读取，也可以从NTP服务器上读取，只要能够返回一个准确的时间即可（其实不准确也可以，只要是统一的，一直往前的，不会出现时钟回拨的）
type TimeProvider interface {

	// GetTime 获取当前时间
	GetTime(ctx context.Context) (time.Time, error)
}
```

## ConnectionManager

ConnectionManager用来管理与Storage的连接，包括连接的申请与回收，以及在系统退出时的资源销毁等等。

```go
// ConnectionManager 把与Storage的连接的管理抽象为一个组件，属于比较底层的接口，用来适配上层的各种情况
// 比如上层可以是从DSN直接创建数据库连接，也可以是从一个已经存在的连接池中拿出来连接，甚至从已有的ORM、sqlx、sql.DB中复用连接
// 或者任何你想扩展的实现，总之它是一个带泛型的接口，你可以根据你的需求发挥想象力任意创造！
type ConnectionManager[Connection any] interface {

	// Name 连接提供器的名字，用于区分不同的连接提供器，连接器的名字必须指定不允许为空字符串
	Name() string

	// Take 获取一个往Storage的连接
	Take(ctx context.Context) (Connection, error)

	// Return 使用完毕，把Storage的连接归还，用于在一些从连接池中拿连接使用完毕必须手动释放否则会资源泄露的场景下及时释放资源
	Return(ctx context.Context, connection Connection) error

	// Shutdown 把整个连接管理器关闭掉，彻底不用了，Storage Lock并不会调用这个方法，你应该在你的系统退出的时候调用此方法释放整个连接管理器使用到的资源
	Shutdown(ctx context.Context) error
}
```

对于ConnectionManager，内置了三个实现：

- DsnConnectionManager：指定driverName和DSN创建一个连接，之后就一直使用这个连接

- FixedSqlDBConnectionManager：每次都返回一个固定的sql.DB

- FuncConnectionProvider：把上面的接口的几个方法使用函数式的方式包裹了一下，用于不方便声明struct的情况下用几个func组合出ConnectionManager接口

# 四、实现示例

下面是一些实现了Storage的例子：

- https://github.com/storage-lock/go-mysql-storage
- https://github.com/storage-lock/go-postgresql-storage



