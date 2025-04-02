package joblock

import (
	clientv3 "go.etcd.io/etcd/client/v3"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
)

func setupEtcdClient(t *testing.T) *etcd.Client {
	// 确保全局配置已初始化
	if config.GlobalConfig == nil {
		config.GlobalConfig = &config.Config{
			EtcdEndpoints:   []string{"localhost:2379"},
			EtcdDialTimeout: 5000,
			JobLockTTL:      5,
		}
	}

	client, err := etcd.NewClient()
	require.NoError(t, err, "Failed to create etcd client")
	return client
}

func cleanupLock(t *testing.T, client *etcd.Client, jobName string) {
	lockKey := common.JobLockDir + jobName
	_, err := client.Delete(lockKey)
	if err != nil {
		t.Logf("Warning: cleanup lock failed: %v", err)
	}
}

func TestNewJobLock(t *testing.T) {
	client := setupEtcdClient(t)
	defer client.Close()

	jobName := "test_job_lock"
	jobLock := NewJobLock(client, jobName)

	assert.Equal(t, client, jobLock.etcdClient)
	assert.Equal(t, jobName, jobLock.jobName)
	assert.Equal(t, common.JobLockDir+jobName, jobLock.lockKey)
	assert.Equal(t, false, jobLock.isLocked)
}

func TestJobLock_TryLock(t *testing.T) {
	client := setupEtcdClient(t)
	defer client.Close()

	jobName := "test_try_lock"
	jobLock := NewJobLock(client, jobName)

	// 清理之前可能存在的锁
	cleanupLock(t, client, jobName)

	// 尝试获取锁
	err := jobLock.TryLock()
	assert.NoError(t, err, "Should acquire lock without error")
	assert.True(t, jobLock.IsLocked(), "Lock should be acquired")
	assert.NotEqual(t, 0, jobLock.leaseID, "Lease ID should be set")

	// 释放锁
	jobLock.Unlock()
	assert.False(t, jobLock.IsLocked(), "Lock should be released")
}

func TestJobLock_LockCompetition(t *testing.T) {
	client := setupEtcdClient(t)
	defer client.Close()

	jobName := "test_lock_competition"

	// 清理之前可能存在的锁
	cleanupLock(t, client, jobName)

	// 第一个锁
	lock1 := NewJobLock(client, jobName)
	err := lock1.TryLock()
	assert.NoError(t, err, "First lock should be acquired")
	assert.True(t, lock1.IsLocked(), "First lock should be acquired")

	// 第二个锁，应该获取失败
	lock2 := NewJobLock(client, jobName)
	err = lock2.TryLock()
	assert.Error(t, err, "Second lock should fail")
	assert.False(t, lock2.IsLocked(), "Second lock should not be acquired")
	assert.Equal(t, common.ErrLockAlreadyAcquired, err, "Error should be ErrLockAlreadyAcquired")

	// 释放第一个锁
	lock1.Unlock()

	// 第二个锁现在应该能够获取了
	err = lock2.TryLock()
	assert.NoError(t, err, "Second lock should succeed after first is released")
	assert.True(t, lock2.IsLocked(), "Second lock should be acquired")

	// 清理
	lock2.Unlock()
}

func TestJobLock_Unlock(t *testing.T) {
	client := setupEtcdClient(t)
	defer client.Close()

	jobName := "test_unlock"
	jobLock := NewJobLock(client, jobName)

	// 清理之前可能存在的锁
	cleanupLock(t, client, jobName)

	// 获取锁
	err := jobLock.TryLock()
	require.NoError(t, err, "Should acquire lock without error")

	// 释放锁
	jobLock.Unlock()
	assert.False(t, jobLock.IsLocked(), "Lock should be released")
	assert.Equal(t, clientv3.LeaseID(0), jobLock.leaseID, "Lease ID should be reset")

	// 确认锁确实被释放了，另一把锁应该能够获取
	anotherLock := NewJobLock(client, jobName)
	err = anotherLock.TryLock()
	assert.NoError(t, err, "Another lock should be acquired after unlock")

	// 清理
	anotherLock.Unlock()
}

func TestJobLock_LockWithTimeout(t *testing.T) {
	client := setupEtcdClient(t)
	defer client.Close()

	jobName := "test_lock_timeout"

	cleanupLock(t, client, jobName)

	lock := NewJobLock(client, jobName)
	err := lock.LockWithTimeout(2 * time.Second)
	assert.NoError(t, err, "Should acquire lock with timeout")
	assert.True(t, lock.IsLocked(), "Lock should be acquired")

	lock.Unlock()

	lock1 := NewJobLock(client, jobName)
	err = lock1.TryLock()
	require.NoError(t, err, "First lock should be acquired")

	lock2 := NewJobLock(client, jobName)
	err = lock2.LockWithTimeout(500 * time.Millisecond)

	assert.Error(t, err, "Should fail to acquire lock")
	assert.Equal(t, common.ErrLockAlreadyAcquired, err, "Error should be ErrLockAlreadyAcquired")
	assert.False(t, lock2.IsLocked(), "Second lock should not be acquired")

	lock1.Unlock()
}

func TestJobLock_Concurrency(t *testing.T) {
	client := setupEtcdClient(t)
	defer client.Close()

	jobName := "test_concurrency"

	// 清理之前可能存在的锁
	cleanupLock(t, client, jobName)

	// 并发获取锁的数量
	concurrency := 10
	var wg sync.WaitGroup
	wg.Add(concurrency)

	successCount := 0
	var mu sync.Mutex

	// 并发获取锁
	for i := 0; i < concurrency; i++ {
		go func(index int) {
			defer wg.Done()

			lock := NewJobLock(client, jobName)
			err := lock.TryLock()

			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()

				// 持有锁一小段时间
				time.Sleep(100 * time.Millisecond)

				// 释放锁
				lock.Unlock()
			}
		}(i)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 只有一个goroutine应该成功获取锁
	assert.Equal(t, 1, successCount, "Only one lock should be acquired")
}

func TestJobLock_IsLocked(t *testing.T) {
	client := setupEtcdClient(t)
	defer client.Close()

	jobName := "test_is_locked"
	jobLock := NewJobLock(client, jobName)

	// 清理之前可能存在的锁
	cleanupLock(t, client, jobName)

	// 初始状态应该是未上锁
	assert.False(t, jobLock.IsLocked(), "Initially should not be locked")

	// 获取锁后状态应该是已上锁
	err := jobLock.TryLock()
	require.NoError(t, err, "Should acquire lock without error")
	assert.True(t, jobLock.IsLocked(), "Should be locked after TryLock")

	// 释放锁后状态应该是未上锁
	jobLock.Unlock()
	assert.False(t, jobLock.IsLocked(), "Should not be locked after Unlock")
}

func TestJobLock_JobName(t *testing.T) {
	client := setupEtcdClient(t)
	defer client.Close()

	jobName := "test_job_name"
	jobLock := NewJobLock(client, jobName)

	assert.Equal(t, jobName, jobLock.JobName(), "JobName should return the correct job name")
}
