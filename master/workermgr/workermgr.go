package workermgr

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
)

// WorkerManager 工作节点管理器
type WorkerManager struct {
	etcdClient *etcd.Client                  // etcd客户端
	logger     *zap.Logger                   // 日志对象
	workers    map[string]*common.WorkerInfo // 工作节点列表
	workerLock sync.RWMutex                  // 读写锁，保护workers
	ctx        context.Context               // 上下文，用于控制退出
	cancelFunc context.CancelFunc            // 取消函数
}

// NewWorkerManager 创建工作节点管理器
func NewWorkerManager(etcdClient *etcd.Client, logger *zap.Logger) *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background())

	wm := &WorkerManager{
		etcdClient: etcdClient,
		logger:     logger,
		workers:    make(map[string]*common.WorkerInfo),
		ctx:        ctx,
		cancelFunc: cancel,
	}

	// 立即获取当前所有工作节点
	wm.loadWorkers()

	// 启动工作节点监控
	go wm.watchWorkers()

	return wm
}

// loadWorkers 加载所有工作节点信息
func (wm *WorkerManager) loadWorkers() {
	// 从etcd获取所有工作节点
	resp, err := wm.etcdClient.GetWithPrefix(common.WorkerRegisterDir)
	if err != nil {
		wm.logger.Error("failed to load workers",
			zap.Error(err))
		return
	}

	// 解析工作节点信息
	workers := make(map[string]*common.WorkerInfo)
	for _, kv := range resp.Kvs {
		workerID := string(kv.Key[len(common.WorkerRegisterDir):])

		worker := &common.WorkerInfo{}
		if err := json.Unmarshal(kv.Value, worker); err != nil {
			wm.logger.Error("failed to unmarshal worker info",
				zap.String("workerID", workerID),
				zap.Error(err))
			continue
		}

		workers[workerID] = worker
	}

	// 更新工作节点列表
	wm.workerLock.Lock()
	wm.workers = workers
	wm.workerLock.Unlock()

	wm.logger.Info("workers loaded", zap.Int("count", len(workers)))
}

// watchWorkers 监控工作节点变化
func (wm *WorkerManager) watchWorkers() {
	// 监听worker目录变化
	watchChan := wm.etcdClient.WatchWithPrefix(common.WorkerRegisterDir)

	// 处理工作节点变化事件
	for {
		select {
		case <-wm.ctx.Done():
			// 上下文被取消，退出监控
			wm.logger.Info("worker watcher stopped")
			return

		case watchResp := <-watchChan:
			for _, event := range watchResp.Events {
				wm.handleWorkerEvent(event)
			}
		}
	}
}

// handleWorkerEvent 处理工作节点事件
func (wm *WorkerManager) handleWorkerEvent(event *clientv3.Event) {
	workerID := string(event.Kv.Key[len(common.WorkerRegisterDir):])

	switch event.Type {
	case clientv3.EventTypePut: // 工作节点注册或心跳
		// 解析工作节点信息
		worker := &common.WorkerInfo{}
		if err := json.Unmarshal(event.Kv.Value, worker); err != nil {
			wm.logger.Error("failed to unmarshal worker info",
				zap.String("workerID", workerID),
				zap.Error(err))
			return
		}

		// 更新工作节点信息
		wm.workerLock.Lock()
		wm.workers[workerID] = worker
		wm.workerLock.Unlock()

		wm.logger.Debug("worker registered or heartbeat",
			zap.String("workerID", workerID),
			zap.String("hostname", worker.Hostname))

	case clientv3.EventTypeDelete: // 工作节点注销
		// 从节点列表中删除
		wm.workerLock.Lock()
		delete(wm.workers, workerID)
		wm.workerLock.Unlock()

		wm.logger.Info("worker unregistered",
			zap.String("workerID", workerID))
	}
}

// ListWorkers 获取当前所有工作节点列表
func (wm *WorkerManager) ListWorkers() []*common.WorkerInfo {
	wm.workerLock.RLock()
	defer wm.workerLock.RUnlock()

	// 复制一份工作节点列表
	workers := make([]*common.WorkerInfo, 0, len(wm.workers))
	for _, worker := range wm.workers {
		workers = append(workers, worker)
	}

	return workers
}

// GetWorker 获取指定工作节点信息
func (wm *WorkerManager) GetWorker(workerID string) (*common.WorkerInfo, bool) {
	wm.workerLock.RLock()
	defer wm.workerLock.RUnlock()

	worker, exists := wm.workers[workerID]
	return worker, exists
}

// CheckWorkers 检查工作节点健康状态
func (wm *WorkerManager) CheckWorkers() map[string]string {
	wm.workerLock.RLock()
	defer wm.workerLock.RUnlock()

	now := time.Now().Unix()
	result := make(map[string]string)

	// 检查每个节点的心跳时间
	for id, worker := range wm.workers {
		// 计算最后心跳时间与现在的差值（秒）
		lastHeartbeat := now - worker.LastSeen/1000 // 转换为秒

		if lastHeartbeat > int64(common.WorkerHeartbeatTime/1000*3) {
			// 超过3个心跳周期未收到心跳，标记为离线
			result[id] = "offline"
		} else {
			result[id] = "online"
		}
	}

	return result
}

// GetWorkerStats 获取工作节点统计信息
func (wm *WorkerManager) GetWorkerStats() map[string]interface{} {
	wm.workerLock.RLock()
	defer wm.workerLock.RUnlock()

	// 统计在线节点数量
	total := len(wm.workers)
	online := 0

	// 计算CPU和内存平均使用率
	var totalCPU float64
	var totalMem float64

	now := time.Now().Unix()
	for _, worker := range wm.workers {
		lastHeartbeat := now - worker.LastSeen/1000 // 转换为秒

		if lastHeartbeat <= int64(common.WorkerHeartbeatTime/1000*3) {
			// 节点在线
			online++
			totalCPU += worker.CPUUsage
			totalMem += worker.MemUsage
		}
	}

	// 计算平均值
	var avgCPU, avgMem float64
	if online > 0 {
		avgCPU = totalCPU / float64(online)
		avgMem = totalMem / float64(online)
	}

	// 构建统计结果
	stats := map[string]interface{}{
		"total":       total,
		"online":      online,
		"offline":     total - online,
		"avgCpuUsage": avgCPU,
		"avgMemUsage": avgMem,
	}

	return stats
}

// Stop 停止工作节点管理器
func (wm *WorkerManager) Stop() {
	wm.cancelFunc()
	wm.logger.Info("worker manager stopped")
}
