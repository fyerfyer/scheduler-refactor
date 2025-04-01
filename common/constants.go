package common

// Etcd相关常量
const (
	// 任务保存目录
	JobSaveDir = "/cron/jobs/"

	// 任务锁目录
	JobLockDir = "/cron/lock/"

	// 服务注册目录
	WorkerRegisterDir = "/cron/workers/"

	// Etcd操作超时时间
	EtcdDialTimeout = 5000 // 毫秒

	// 心跳时间
	WorkerHeartbeatTime = 5000 // 毫秒
)

// 任务事件类型
const (
	JobEventSave   = iota + 1 // 保存任务事件
	JobEventDelete            // 删除任务事件
)

// 任务执行结果状态
const (
	JobStatusSuccess = iota // 执行成功
	JobStatusError          // 执行错误
	JobStatusTimeout        // 执行超时
	JobStatusKilled         // 被强制终止
)

// API响应状态码
const (
	ApiSuccess     = 0    // 成功
	ApiFailure     = 1000 // 一般性错误
	ApiParamError  = 1001 // 参数错误
	ApiJobNotExist = 1002 // 任务不存在
	ApiJobExecFail = 1003 // 任务执行失败
	ApiSystemError = 2000 // 系统错误
	ApiDbError     = 2001 // 数据库错误
	ApiEtcdError   = 2002 // Etcd操作错误
)

// 日志批处理相关
const (
	JobLogBatchSize     = 100  // 日志批量写入的大小
	JobLogCommitTimeout = 1000 // 日志自动提交超时时间(毫秒)
)

// 默认值
const (
	DefaultPage       = 1   // 默认页码
	DefaultPageSize   = 10  // 默认页大小
	MaxPageSize       = 100 // 最大页大小
	DefaultJobTimeout = 60  // 默认任务超时时间(秒)
)

// MongoDB 相关
const (
	LogCollectionName = "job_logs" // 日志集合名
)
