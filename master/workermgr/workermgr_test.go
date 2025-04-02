package workermgr

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap/zaptest"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
)

func setupTestEnv(t *testing.T) (*WorkerManager, *etcd.Client, func()) {
	logger := zaptest.NewLogger(t)

	config.GlobalConfig = &config.Config{
		EtcdEndpoints:     []string{"localhost:2379"},
		EtcdDialTimeout:   5000,
		HeartbeatInterval: 5000, // 使用HeartbeatInterval代替WorkerHeartbeatTime
	}

	etcdClient, err := etcd.NewClient()
	require.NoError(t, err, "Failed to create etcd client")

	cleanup := func() {
		// 清除所有测试用的worker注册信息
		_, err := etcdClient.DeleteWithPrefix(common.WorkerRegisterDir)
		if err != nil {
			t.Logf("Failed to clean up test workers: %v", err)
		}
	}

	// 先清理一次，确保测试环境干净
	cleanup()

	workerMgr := NewWorkerManager(etcdClient, logger)
	require.NotNil(t, workerMgr, "WorkerManager should not be nil")

	return workerMgr, etcdClient, cleanup
}

func registerTestWorker(t *testing.T, etcdClient *etcd.Client, workerID string, online bool) {
	workerInfo := &common.WorkerInfo{
		IP:       workerID,
		Hostname: fmt.Sprintf("host-%s", workerID),
		CPUUsage: 0.5,
		MemUsage: 0.3,
		LastSeen: time.Now().UnixNano() / int64(time.Millisecond),
	}

	// 如果要模拟离线状态，将LastSeen设置为很久以前
	if !online {
		workerInfo.LastSeen = time.Now().Add(-10*time.Minute).UnixNano() / int64(time.Millisecond)
	}

	data, err := json.Marshal(workerInfo)
	require.NoError(t, err, "Failed to marshal worker info")

	workerKey := common.WorkerRegisterDir + workerID
	_, err = etcdClient.Put(workerKey, string(data))
	require.NoError(t, err, "Failed to register test worker")
}

func TestNewWorkerManager(t *testing.T) {
	workerMgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	assert.NotNil(t, workerMgr.etcdClient, "etcdClient should not be nil")
	assert.NotNil(t, workerMgr.logger, "logger should not be nil")
	assert.NotNil(t, workerMgr.workers, "workers map should not be nil")
	assert.NotNil(t, workerMgr.ctx, "context should not be nil")
	assert.NotNil(t, workerMgr.cancelFunc, "cancelFunc should not be nil")
}

func TestLoadWorkers(t *testing.T) {
	workerMgr, etcdClient, cleanup := setupTestEnv(t)
	defer workerMgr.Stop()
	defer cleanup()

	// 注册测试worker
	registerTestWorker(t, etcdClient, "worker1", true)
	registerTestWorker(t, etcdClient, "worker2", true)

	// 手动触发loadWorkers
	workerMgr.loadWorkers()

	// 验证是否加载成功
	assert.Equal(t, 2, len(workerMgr.workers), "Should have loaded 2 workers")
	assert.Contains(t, workerMgr.workers, "worker1", "Should contain worker1")
	assert.Contains(t, workerMgr.workers, "worker2", "Should contain worker2")
}

func TestWatchWorkers(t *testing.T) {
	workerMgr, etcdClient, cleanup := setupTestEnv(t)
	defer cleanup()

	// 等待watchWorkers协程启动
	time.Sleep(100 * time.Millisecond)

	// 注册一个新的worker，触发监听事件
	registerTestWorker(t, etcdClient, "worker3", true)

	// 等待事件被处理
	time.Sleep(300 * time.Millisecond)

	// 验证worker是否被添加
	workerMgr.workerLock.RLock()
	worker, exists := workerMgr.workers["worker3"]
	workerMgr.workerLock.RUnlock()

	assert.True(t, exists, "worker3 should exist after registration")
	assert.Equal(t, "worker3", worker.IP, "Worker IP should match")
	assert.Equal(t, "host-worker3", worker.Hostname, "Worker hostname should match")

	// 测试删除worker
	workerKey := common.WorkerRegisterDir + "worker3"
	_, err := etcdClient.Delete(workerKey)
	require.NoError(t, err, "Failed to delete test worker")

	// 等待删除事件被处理
	time.Sleep(300 * time.Millisecond)

	// 验证worker是否被删除
	workerMgr.workerLock.RLock()
	_, exists = workerMgr.workers["worker3"]
	workerMgr.workerLock.RUnlock()

	assert.False(t, exists, "worker3 should not exist after deletion")
}

func TestListWorkers(t *testing.T) {
	workerMgr, etcdClient, cleanup := setupTestEnv(t)
	defer cleanup()

	// 注册测试worker
	registerTestWorker(t, etcdClient, "worker1", true)
	registerTestWorker(t, etcdClient, "worker2", true)

	// 手动加载worker
	workerMgr.loadWorkers()

	// 测试获取worker列表
	workers := workerMgr.ListWorkers()
	assert.Equal(t, 2, len(workers), "Should list 2 workers")

	// 确认返回的是副本，不是原始map的引用
	workerMap := make(map[string]struct{})
	for _, worker := range workers {
		workerMap[worker.IP] = struct{}{}
	}

	assert.Contains(t, workerMap, "worker1", "ListWorkers should include worker1")
	assert.Contains(t, workerMap, "worker2", "ListWorkers should include worker2")
}

func TestGetWorker(t *testing.T) {
	workerMgr, etcdClient, cleanup := setupTestEnv(t)
	defer cleanup()

	// 注册测试worker
	registerTestWorker(t, etcdClient, "worker1", true)

	// 手动加载worker
	workerMgr.loadWorkers()

	// 测试获取存在的worker
	worker, exists := workerMgr.GetWorker("worker1")
	assert.True(t, exists, "worker1 should exist")
	assert.Equal(t, "worker1", worker.IP, "Worker IP should match")
	assert.Equal(t, "host-worker1", worker.Hostname, "Worker hostname should match")

	// 测试获取不存在的worker
	worker, exists = workerMgr.GetWorker("nonexistent")
	assert.False(t, exists, "nonexistent worker should not exist")
	assert.Nil(t, worker, "Worker should be nil for nonexistent worker")
}

func TestCheckWorkers(t *testing.T) {
	workerMgr, etcdClient, cleanup := setupTestEnv(t)
	defer cleanup()

	// 注册在线和离线worker
	registerTestWorker(t, etcdClient, "online-worker", true)
	registerTestWorker(t, etcdClient, "offline-worker", false)

	// 手动加载worker
	workerMgr.loadWorkers()

	// 检查worker状态
	statusMap := workerMgr.CheckWorkers()

	assert.Equal(t, "online", statusMap["online-worker"], "online-worker should be online")
	assert.Equal(t, "offline", statusMap["offline-worker"], "offline-worker should be offline")
}

func TestGetWorkerStats(t *testing.T) {
	workerMgr, etcdClient, cleanup := setupTestEnv(t)
	defer cleanup()

	// 注册在线和离线worker
	registerTestWorker(t, etcdClient, "online-worker", true)
	registerTestWorker(t, etcdClient, "offline-worker", false)

	// 手动加载worker
	workerMgr.loadWorkers()

	// 获取统计信息
	stats := workerMgr.GetWorkerStats()

	assert.Equal(t, 2, stats["total"], "Should have 2 total workers")
	assert.Equal(t, 1, stats["online"], "Should have 1 online worker")
	assert.Equal(t, 1, stats["offline"], "Should have 1 offline worker")
	assert.Equal(t, 0.5, stats["avgCpuUsage"], "Average CPU usage should be 0.5")
	assert.Equal(t, 0.3, stats["avgMemUsage"], "Average memory usage should be 0.3")
}

func TestHandleWorkerEvent(t *testing.T) {
	workerMgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// 创建worker信息
	workerInfo := &common.WorkerInfo{
		IP:       "test-worker",
		Hostname: "host-test",
		CPUUsage: 0.7,
		MemUsage: 0.4,
		LastSeen: time.Now().Unix(),
	}

	// 序列化worker信息
	data, err := json.Marshal(workerInfo)
	require.NoError(t, err, "Failed to marshal worker info")

	// 模拟PUT事件
	putEvent := &clientv3.Event{
		Type: clientv3.EventTypePut,
		Kv: &mvccpb.KeyValue{
			Key:   []byte(common.WorkerRegisterDir + "test-worker"),
			Value: data,
		},
	}

	// 处理PUT事件
	workerMgr.handleWorkerEvent(putEvent)

	// 验证worker是否被添加
	workerMgr.workerLock.RLock()
	worker, exists := workerMgr.workers["test-worker"]
	workerMgr.workerLock.RUnlock()

	assert.True(t, exists, "Worker should be added after PUT event")
	assert.Equal(t, "test-worker", worker.IP, "Worker IP should match")
	assert.Equal(t, "host-test", worker.Hostname, "Worker hostname should match")

	// 模拟DELETE事件
	deleteEvent := &clientv3.Event{
		Type: clientv3.EventTypeDelete,
		Kv: &mvccpb.KeyValue{
			Key: []byte(common.WorkerRegisterDir + "test-worker"),
		},
	}

	// 处理DELETE事件
	workerMgr.handleWorkerEvent(deleteEvent)

	// 验证worker是否被删除
	workerMgr.workerLock.RLock()
	_, exists = workerMgr.workers["test-worker"]
	workerMgr.workerLock.RUnlock()

	assert.False(t, exists, "Worker should be removed after DELETE event")
}

func TestStop(t *testing.T) {
	workerMgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	initialCtx := workerMgr.ctx
	workerMgr.Stop()

	select {
	case <-initialCtx.Done():
		assert.True(t, true, "Context should be canceled after Stop")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be canceled after Stop")
	}
}

func TestWorkerManagerWithRealEtcdEvents(t *testing.T) {
	workerMgr, etcdClient, cleanup := setupTestEnv(t)
	defer cleanup()

	// 等待watchWorkers协程启动
	time.Sleep(100 * time.Millisecond)

	// 测试添加多个worker
	registerTestWorker(t, etcdClient, "worker1", true)
	registerTestWorker(t, etcdClient, "worker2", true)

	// 等待事件处理
	time.Sleep(300 * time.Millisecond)

	workers := workerMgr.ListWorkers()
	assert.Equal(t, 2, len(workers), "Should have 2 workers")

	// 测试更新worker信息
	updatedInfo := &common.WorkerInfo{
		IP:       "worker1",
		Hostname: "updated-host",
		CPUUsage: 0.8,
		MemUsage: 0.6,
		LastSeen: time.Now().Unix(),
	}

	// 更新worker1
	data, err := json.Marshal(updatedInfo)
	require.NoError(t, err, "Failed to marshal worker info")

	workerKey := common.WorkerRegisterDir + "worker1"
	_, err = etcdClient.Put(workerKey, string(data))
	require.NoError(t, err, "Failed to update test worker")

	// 等待事件处理
	time.Sleep(300 * time.Millisecond)

	// 验证worker是否被更新
	worker, exists := workerMgr.GetWorker("worker1")
	assert.True(t, exists, "worker1 should exist")
	assert.Equal(t, "updated-host", worker.Hostname, "Worker hostname should be updated")
	assert.Equal(t, 0.8, worker.CPUUsage, "Worker CPU usage should be updated")
}

func TestWorkerManagerConcurrency(t *testing.T) {
	workerMgr, etcdClient, cleanup := setupTestEnv(t)
	defer cleanup()

	// 并发注册多个worker
	for i := 0; i < 5; i++ {
		workerID := fmt.Sprintf("worker%d", i)
		go registerTestWorker(t, etcdClient, workerID, true)
	}

	// 等待所有注册完成
	time.Sleep(500 * time.Millisecond)

	// 检查是否所有worker都被正确加载
	workers := workerMgr.ListWorkers()
	assert.LessOrEqual(t, 5, len(workers), "Should have at least 5 workers")

	// 并发获取worker信息
	var found int32 = 0
	resultChan := make(chan struct{}, 5)

	for i := 0; i < 5; i++ {
		workerID := fmt.Sprintf("worker%d", i)
		go func(id string) {
			_, exists := workerMgr.GetWorker(id)
			if exists {
				resultChan <- struct{}{}
			}
		}(workerID)
	}

	// 等待所有goroutine完成
	for i := 0; i < 5; i++ {
		select {
		case <-resultChan:
			found++
		case <-time.After(100 * time.Millisecond):
			// 超时
		}
	}

	assert.Equal(t, int32(5), found, "All 5 workers should be found")
}

func TestDeleteWithPrefix(t *testing.T) {
	_, etcdClient, cleanup := setupTestEnv(t)
	defer cleanup()

	registerTestWorker(t, etcdClient, "test1", true)
	registerTestWorker(t, etcdClient, "test2", true)

	resp, err := etcdClient.GetWithPrefix(common.WorkerRegisterDir)
	require.NoError(t, err, "Should get keys from etcd")
	assert.Equal(t, 2, len(resp.Kvs), "Should have 2 worker keys")

	_, err = etcdClient.DeleteWithPrefix(common.WorkerRegisterDir)
	require.NoError(t, err, "Should delete keys with prefix")

	resp, err = etcdClient.GetWithPrefix(common.WorkerRegisterDir)
	require.NoError(t, err, "Should get keys from etcd after deletion")
	assert.Equal(t, 0, len(resp.Kvs), "All worker keys should be deleted")
}
