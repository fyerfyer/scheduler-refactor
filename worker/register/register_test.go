package register

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
)

func TestNewRegister(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	etcdClient, err := setupTestEtcd()
	require.NoError(t, err, "Failed to setup test ETCD")
	defer etcdClient.Close()

	reg := NewRegister(logger, etcdClient)

	assert.NotNil(t, reg, "Register should not be nil")
	assert.Equal(t, config.GlobalConfig.WorkerID, reg.workerInfo.IP)
	assert.NotEmpty(t, reg.workerInfo.Hostname, "Hostname should not be empty")
	assert.NotZero(t, reg.workerInfo.LastSeen, "LastSeen should not be zero")
	assert.Contains(t, reg.registryKey, common.WorkerRegisterDir, "Registry key should contain worker register directory")
}

func TestRegisterLifecycle(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	etcdClient, err := setupTestEtcd()
	require.NoError(t, err, "Failed to setup test ETCD")
	defer etcdClient.Close()
	config.GlobalConfig.HeartbeatInterval = 500

	reg := NewRegister(logger, etcdClient)

	// 启动注册
	err = reg.Start()
	assert.NoError(t, err, "Should start register successfully")

	// 等待足够的时间以确保注册完成
	time.Sleep(100 * time.Millisecond)

	// 检查是否成功注册到etcd
	resp, err := etcdClient.Get(reg.registryKey)
	assert.NoError(t, err, "Should get key from etcd")
	assert.True(t, len(resp.Kvs) > 0, "Key should exist in etcd")

	// 解析注册信息
	var workerInfo common.WorkerInfo
	err = json.Unmarshal(resp.Kvs[0].Value, &workerInfo)
	assert.NoError(t, err, "Should unmarshal worker info")
	assert.Equal(t, config.GlobalConfig.WorkerID, workerInfo.IP)

	// 等待心跳更新
	time.Sleep(600 * time.Millisecond)

	// 检查心跳是否更新了时间
	resp, err = etcdClient.Get(reg.registryKey)
	assert.NoError(t, err, "Should get key from etcd after heartbeat")
	assert.True(t, len(resp.Kvs) > 0, "Key should still exist in etcd")

	var updatedWorkerInfo common.WorkerInfo
	err = json.Unmarshal(resp.Kvs[0].Value, &updatedWorkerInfo)
	assert.NoError(t, err, "Should unmarshal updated worker info")
	assert.True(t, updatedWorkerInfo.LastSeen >= workerInfo.LastSeen, "LastSeen should be updated")

	reg.Stop()

	// 等待context被取消
	time.Sleep(100 * time.Millisecond)

	// 验证context已被取消
	select {
	case <-reg.ctx.Done():
		assert.True(t, true, "Context should be canceled")
	default:
		assert.Fail(t, "Context should be canceled after register is stopped")
	}
}

func TestRegisterWithInvalidEtcd(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config.GlobalConfig = &config.Config{
		EtcdEndpoints:     []string{"localhost:1234"},
		EtcdDialTimeout:   100,
		WorkerID:          "test-worker",
		HeartbeatInterval: 1000,
		JobLockTTL:        5,
	}

	etcdClient, err := etcd.NewClient()
	if err != nil {
		// 如果创建客户端就失败了，这是预期的行为
		return
	}

	// 如果能创建客户端，但连接应该是有问题的
	reg := NewRegister(logger, etcdClient)

	// 尝试启动注册器，应该会失败
	err = reg.Start()
	assert.Error(t, err, "Should fail to start register with invalid etcd")
}

func TestDoRegister(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	etcdClient, err := setupTestEtcd()
	require.NoError(t, err, "Failed to setup test ETCD")
	defer etcdClient.Close()

	reg := NewRegister(logger, etcdClient)

	// 测试初次注册
	err = reg.doRegister()
	assert.NoError(t, err, "Should register successfully")

	// 验证数据已写入etcd
	resp, err := etcdClient.Get(reg.registryKey)
	assert.NoError(t, err, "Should get key from etcd")
	assert.True(t, len(resp.Kvs) > 0, "Key should exist in etcd")

	// 验证写入的数据是否正确
	var workerInfo common.WorkerInfo
	err = json.Unmarshal(resp.Kvs[0].Value, &workerInfo)
	assert.NoError(t, err, "Should unmarshal worker info")
	assert.Equal(t, config.GlobalConfig.WorkerID, workerInfo.IP)
	assert.NotEmpty(t, workerInfo.Hostname)
	assert.NotZero(t, workerInfo.LastSeen)
}

func TestUpdateWorkerInfo(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	etcdClient, err := setupTestEtcd()
	require.NoError(t, err, "Failed to setup test ETCD")
	defer etcdClient.Close()

	reg := NewRegister(logger, etcdClient)

	// 记录初始时间
	initialTime := reg.workerInfo.LastSeen

	// 等待一些时间
	time.Sleep(10 * time.Millisecond)

	// 更新工作节点信息
	reg.updateWorkerInfo()

	// 验证时间是否已更新
	assert.True(t, reg.workerInfo.LastSeen > initialTime, "LastSeen should be updated")

	// 验证是否收集了系统状态
	assert.NotZero(t, reg.workerInfo.MemUsage, "MemUsage should be collected")
	assert.NotZero(t, reg.workerInfo.CPUUsage, "CPUUsage should be collected")
}

func TestGetWorkerInfo(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	etcdClient, err := setupTestEtcd()
	require.NoError(t, err, "Failed to setup test ETCD")
	defer etcdClient.Close()

	reg := NewRegister(logger, etcdClient)

	// 测试获取工作节点信息
	info := reg.GetWorkerInfo()

	assert.Equal(t, reg.workerInfo.IP, info.IP)
	assert.Equal(t, reg.workerInfo.Hostname, info.Hostname)
	assert.Equal(t, reg.workerInfo.LastSeen, info.LastSeen)
	assert.Equal(t, reg.workerInfo.CPUUsage, info.CPUUsage)
	assert.Equal(t, reg.workerInfo.MemUsage, info.MemUsage)
}

func TestHeartbeatLoop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping heartbeat test in short mode")
	}

	logger, _ := zap.NewDevelopment()
	etcdClient, err := setupTestEtcd()
	require.NoError(t, err, "Failed to setup test ETCD")
	defer etcdClient.Close()

	// 设置一个较短的心跳间隔
	config.GlobalConfig.HeartbeatInterval = 100 // 100毫秒

	reg := NewRegister(logger, etcdClient)

	// 启动心跳
	ctx, cancel := context.WithCancel(context.Background())
	reg.ctx = ctx
	reg.cancelFunc = cancel

	// 先注册一次
	err = reg.doRegister()
	assert.NoError(t, err, "Initial register should succeed")

	// 启动心跳
	go reg.heartbeatLoop()

	// 检查初始注册状态
	resp, err := etcdClient.Get(reg.registryKey)
	require.NoError(t, err, "Should get key from etcd")
	initialValue := string(resp.Kvs[0].Value)

	// 等待几次心跳周期
	time.Sleep(350 * time.Millisecond)

	// 检查是否有更新
	resp, err = etcdClient.Get(reg.registryKey)
	require.NoError(t, err, "Should get key from etcd")
	updatedValue := string(resp.Kvs[0].Value)

	// 由于心跳会更新LastSeen，值应该不同
	assert.NotEqual(t, initialValue, updatedValue,
		"Worker info should be updated by heartbeat")

	// 取消上下文，心跳应该停止
	cancel()
	time.Sleep(150 * time.Millisecond)
}

func setupTestEtcd() (*etcd.Client, error) {
	config.GlobalConfig = &config.Config{
		EtcdEndpoints:     []string{"localhost:2379"},
		EtcdDialTimeout:   5000,
		WorkerID:          "test-worker",
		HeartbeatInterval: 1000,
		JobLockTTL:        5,
	}

	return etcd.NewClient()
}
