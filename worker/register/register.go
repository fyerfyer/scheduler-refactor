package register

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
)

// Register 注册器，负责worker节点的注册和心跳
type Register struct {
	logger      *zap.Logger        // 日志对象
	etcdClient  *etcd.Client       // etcd客户端
	workerInfo  common.WorkerInfo  // 工作节点信息
	registryKey string             // 注册key
	ctx         context.Context    // 上下文，用于控制退出
	cancelFunc  context.CancelFunc // 取消函数
}

// NewRegister 创建注册器
func NewRegister(logger *zap.Logger, etcdClient *etcd.Client) *Register {
	ctx, cancel := context.WithCancel(context.Background())

	// 获取本地主机名
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// 创建工作节点信息
	workerInfo := common.WorkerInfo{
		IP:       config.GlobalConfig.WorkerID,
		Hostname: hostname,
		LastSeen: time.Now().Unix(),
	}

	// 创建注册key
	registryKey := fmt.Sprintf("%s%s", common.WorkerRegisterDir, config.GlobalConfig.WorkerID)

	return &Register{
		logger:      logger,
		etcdClient:  etcdClient,
		workerInfo:  workerInfo,
		registryKey: registryKey,
		ctx:         ctx,
		cancelFunc:  cancel,
	}
}

// Start 开始注册并定期发送心跳
func (r *Register) Start() error {
	r.logger.Info("worker register starting...",
		zap.String("workerID", config.GlobalConfig.WorkerID))

	// 立即执行一次注册
	if err := r.doRegister(); err != nil {
		r.logger.Error("initial registration failed",
			zap.String("workerID", config.GlobalConfig.WorkerID),
			zap.Error(err))
		return err
	}

	// 启动心跳协程
	go r.heartbeatLoop()

	return nil
}

// Stop 停止注册和心跳
func (r *Register) Stop() {
	r.cancelFunc()
	r.logger.Info("worker register stopped")
}

// doRegister 执行注册
func (r *Register) doRegister() error {
	// 更新节点信息
	r.updateWorkerInfo()

	// 序列化为JSON
	data, err := json.Marshal(r.workerInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal worker info: %v", err)
	}

	// 写入etcd，设置租约TTL为心跳间隔的2倍
	heartbeatInterval := config.GlobalConfig.HeartbeatInterval
	ttl := int64(heartbeatInterval * 2 / 1000) // 转换为秒
	if ttl < 5 {
		ttl = 5 // 最小5秒
	}

	// 写入etcd
	err = r.etcdClient.PutWithLease(r.registryKey, string(data), ttl)
	if err != nil {
		return fmt.Errorf("failed to register worker: %v", err)
	}

	r.logger.Info("worker registered successfully",
		zap.String("workerID", config.GlobalConfig.WorkerID),
		zap.Int64("ttl", ttl))

	return nil
}

// updateWorkerInfo 更新工作节点信息
func (r *Register) updateWorkerInfo() {
	// 更新最后心跳时间
	r.workerInfo.LastSeen = time.Now().UnixNano() / int64(time.Millisecond)

	// 这里可以添加更多节点状态收集逻辑，例如CPU和内存使用率
	r.collectSystemStats()
}

// collectSystemStats 收集系统状态信息
func (r *Register) collectSystemStats() {
	// 简单实现，实际生产环境可能需要更复杂的监控
	// 这里只是示例，不做实际的系统指标收集
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 仅作为演示，实际上这个值没有太大意义
	r.workerInfo.MemUsage = float64(memStats.Alloc) / float64(memStats.Sys)

	// CPU使用率需要更复杂的计算，这里简单设置一个示例值
	r.workerInfo.CPUUsage = 0.5 // 示例值，真实实现应当计算实际CPU使用率
}

// heartbeatLoop 心跳循环
func (r *Register) heartbeatLoop() {
	// 心跳间隔
	interval := time.Duration(config.GlobalConfig.HeartbeatInterval) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done(): // 上下文取消
			r.logger.Info("heartbeat loop stopped")
			return
		case <-ticker.C: // 定时器触发
			if err := r.doRegister(); err != nil {
				r.logger.Error("heartbeat failed",
					zap.String("workerID", config.GlobalConfig.WorkerID),
					zap.Error(err))
			}
		}
	}
}

// GetWorkerInfo 获取工作节点信息
func (r *Register) GetWorkerInfo() common.WorkerInfo {
	return r.workerInfo
}
