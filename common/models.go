package common

import (
    "time"
)

// Job 任务结构
type Job struct {
    Name      string `json:"name"`      // 任务名称
    Command   string `json:"command"`   // shell命令
    CronExpr  string `json:"cronExpr"`  // cron表达式
    Timeout   int    `json:"timeout"`   // 任务超时时间(秒)，0表示不限制
    Disabled  bool   `json:"disabled"`  // 是否禁用
    CreatedAt int64  `json:"createdAt"` // 创建时间
    UpdatedAt int64  `json:"updatedAt"` // 更新时间
}

// JobEvent 任务变更事件
type JobEvent struct {
    EventType int // 事件类型: 1-保存, 2-删除
    Job       *Job
}

// JobExecuteInfo 任务执行状态信息
type JobExecuteInfo struct {
    Job        *Job               // 任务信息
    PlanTime   time.Time          // 理论调度时间
    RealTime   time.Time          // 实际调度时间
    StartTime  time.Time          // 任务开始执行时间
    EndTime    time.Time          // 任务执行结束时间
    CancelCtx  interface{}        // 任务command的上下文(用于取消任务)
    CancelFunc interface{}        // 用于取消command执行
    Result     *JobExecuteResult  // 任务执行结果
}

// JobExecuteResult 任务执行结果
type JobExecuteResult struct {
    JobName    string    // 任务名称
    Output     string    // 命令输出
    Error      string    // 错误原因
    StartTime  time.Time // 启动时间
    EndTime    time.Time // 结束时间
    ExitCode   int       // 退出码
    IsTimeout  bool      // 是否超时
}

// JobLog 任务执行日志
type JobLog struct {
    JobName      string    `json:"jobName" bson:"jobName"`           // 任务名称
    Command      string    `json:"command" bson:"command"`           // 命令
    Output       string    `json:"output" bson:"output"`             // 命令输出
    Error        string    `json:"error" bson:"error"`               // 错误输出
    PlanTime     int64     `json:"planTime" bson:"planTime"`         // 计划开始时间
    ScheduleTime int64     `json:"scheduleTime" bson:"scheduleTime"` // 实际调度时间
    StartTime    int64     `json:"startTime" bson:"startTime"`       // 任务执行开始时间
    EndTime      int64     `json:"endTime" bson:"endTime"`           // 任务执行结束时间
    ExitCode     int       `json:"exitCode" bson:"exitCode"`         // 退出码
    IsTimeout    bool      `json:"isTimeout" bson:"isTimeout"`       // 是否超时
    WorkerIP     string    `json:"workerIp" bson:"workerIp"`         // 执行机器IP
}

// WorkerInfo 工作节点信息
type WorkerInfo struct {
    IP        string `json:"ip"`        // 节点IP
    Hostname  string `json:"hostname"`  // 主机名
    CPUUsage  float64 `json:"cpuUsage"` // CPU使用率
    MemUsage  float64 `json:"memUsage"` // 内存使用率
    LastSeen  int64   `json:"lastSeen"` // 最后心跳时间
}

// ApiResponse API响应格式
type ApiResponse struct {
    Code    int         `json:"code"`    // 错误码，0-成功，非0-失败
    Message string      `json:"message"` // 错误信息
    Data    interface{} `json:"data"`    // 响应数据
}

// JobListRequest 获取任务列表请求
type JobListRequest struct {
    Page     int    `json:"page"`     // 页码，从1开始
    PageSize int    `json:"pageSize"` // 每页大小
    Keyword  string `json:"keyword"`  // 搜索关键字
}

// JobLogRequest 获取任务日志请求
type JobLogRequest struct {
    JobName  string `json:"jobName"`  // 任务名称
    Page     int    `json:"page"`     // 页码，从1开始
    PageSize int    `json:"pageSize"` // 每页大小
}