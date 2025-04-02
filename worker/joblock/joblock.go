package joblock

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
)

// JobLock 任务锁结构
type JobLock struct {
	etcdClient *etcd.Client       // etcd客户端
	jobName    string             // 任务名称
	lockKey    string             // 锁路径
	leaseID    clientv3.LeaseID   // 租约ID
	isLocked   bool               // 是否已上锁
	cancelFunc context.CancelFunc // 用于取消自动续租
}

// NewJobLock 创建任务锁
func NewJobLock(etcdClient *etcd.Client, jobName string) *JobLock {
	return &JobLock{
		etcdClient: etcdClient,
		jobName:    jobName,
		lockKey:    fmt.Sprintf("%s%s", common.JobLockDir, jobName), // 锁在etcd中的key
		leaseID:    0,
		isLocked:   false,
	}
}

// TryLock 尝试获取任务锁
func (jl *JobLock) TryLock() error {
	// 获取配置的锁超时时间
	ttl := int64(config.GlobalConfig.JobLockTTL)

	// 尝试获取锁
	leaseID, err := jl.etcdClient.TryAcquireLock(jl.lockKey, ttl)
	if err != nil {
		return err
	}

	// 获取锁成功，记录租约ID
	jl.leaseID = leaseID
	jl.isLocked = true

	// 自动续租
	ctx, cancel := context.WithCancel(context.Background())
	jl.cancelFunc = cancel

	// 启动一个goroutine处理续租响应
	go jl.keepAlive(ctx)

	return nil
}

// Unlock 释放锁
func (jl *JobLock) Unlock() {
	// 如果已经上锁
	if jl.isLocked {
		// 取消自动续租
		if jl.cancelFunc != nil {
			jl.cancelFunc()
		}

		// 释放锁
		jl.etcdClient.ReleaseLock(jl.lockKey, jl.leaseID)

		// 重置状态
		jl.leaseID = 0
		jl.isLocked = false
	}
}

// keepAlive 保持锁有效
func (jl *JobLock) keepAlive(ctx context.Context) {
	// 启动自动续租
	keepAliveChan, err := jl.etcdClient.KeepAlive(jl.leaseID)
	if err != nil {
		// 续租失败，锁已失效
		jl.isLocked = false
		return
	}

	for {
		select {
		case <-ctx.Done(): // 上下文取消
			return
		case _, ok := <-keepAliveChan: // 续租应答
			if !ok {
				// 续租失败，锁已失效
				jl.isLocked = false
				return
			}
		}
	}
}

// IsLocked 判断是否已上锁
func (jl *JobLock) IsLocked() bool {
	return jl.isLocked
}

// JobName 获取任务名
func (jl *JobLock) JobName() string {
	return jl.jobName
}

// LockWithTimeout 带超时的获取锁
func (jl *JobLock) LockWithTimeout(timeout time.Duration) error {
	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 尝试获取锁的通道
	done := make(chan error, 1)
	go func() {
		done <- jl.TryLock()
	}()

	// 等待锁或超时
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("acquire lock for job %s timeout after %v", jl.jobName, timeout)
	}
}
