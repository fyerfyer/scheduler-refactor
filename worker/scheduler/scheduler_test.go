package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
	"github.com/fyerfyer/scheduler-refactor/worker/executor"
	"github.com/fyerfyer/scheduler-refactor/worker/jobmgr"
)

func createTestJob(name string, command string, cronExpr string, disabled bool) *common.Job {
	return &common.Job{
		Name:      name,
		Command:   command,
		CronExpr:  cronExpr,
		Timeout:   10,
		Disabled:  disabled,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
}

func TestNewScheduler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	etcdClient, err := setupTestEtcd()
	require.NoError(t, err, "Failed to setup test ETCD")
	defer etcdClient.Close()

	exec := executor.NewExecutor(logger)
	jobMan := jobmgr.NewJobManager(etcdClient, logger)

	scheduler := NewScheduler(logger, jobMan, etcdClient, exec)

	assert.NotNil(t, scheduler, "Scheduler should not be nil")
	assert.NotNil(t, scheduler.jobPlans, "JobPlans map should not be nil")
	assert.NotNil(t, scheduler.jobExecuting, "JobExecuting map should not be nil")
}

func TestParseCronExpr(t *testing.T) {
	validExpr := "*/5 * * * * *"
	invalidExpr := "invalid cron"

	validResult, err := cronexpr.Parse(validExpr)
	assert.NoError(t, err, "Should parse valid cron expression")
	assert.NotNil(t, validResult, "Valid cron expression should return non-nil result")

	_, err = cronexpr.Parse(invalidExpr)
	assert.Error(t, err, "Should fail to parse invalid cron expression")
}

func TestHandleJobEvent(t *testing.T) {
	scheduler := setupTestScheduler(t)
	defer scheduler.Stop()

	// 测试添加任务事件
	job1 := createTestJob("testjob1", "echo hello", "*/5 * * * * *", false)
	saveEvent := &common.JobEvent{
		EventType: common.JobEventSave,
		Job:       job1,
	}

	scheduler.handleJobEvent(saveEvent)

	_, exists := scheduler.jobPlans["testjob1"]
	assert.True(t, exists, "Job should be added to job plans")

	// 测试禁用任务事件
	job1.Disabled = true
	disableEvent := &common.JobEvent{
		EventType: common.JobEventSave,
		Job:       job1,
	}

	scheduler.handleJobEvent(disableEvent)

	_, exists = scheduler.jobPlans["testjob1"]
	assert.False(t, exists, "Disabled job should be removed from job plans")

	// 测试删除任务事件
	job2 := createTestJob("testjob2", "echo world", "*/10 * * * * *", false)
	saveEvent2 := &common.JobEvent{
		EventType: common.JobEventSave,
		Job:       job2,
	}

	scheduler.handleJobEvent(saveEvent2)

	_, exists = scheduler.jobPlans["testjob2"]
	assert.True(t, exists, "Job should be added to job plans")

	deleteEvent := &common.JobEvent{
		EventType: common.JobEventDelete,
		Job:       job2,
	}

	scheduler.handleJobEvent(deleteEvent)

	_, exists = scheduler.jobPlans["testjob2"]
	assert.False(t, exists, "Deleted job should be removed from job plans")
}

func TestTrySchedule(t *testing.T) {
	scheduler := setupTestScheduler(t)
	defer scheduler.Stop()

	// 创建一个过去时间的计划任务
	pastTime := time.Now().Add(-1 * time.Minute)
	job := createTestJob("testjob", "echo test", "*/1 * * * * *", false)
	expr, _ := cronexpr.Parse(job.CronExpr)

	plan := &JobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: pastTime,
	}

	scheduler.jobPlans["testjob"] = plan
	scheduler.trySchedule()

	// 验证NextTime已经被更新到未来的时间
	assert.True(t, scheduler.jobPlans["testjob"].NextTime.After(time.Now()),
		"NextTime should be updated to a future time")
}

func TestStartAndStop(t *testing.T) {
	scheduler := setupTestScheduler(t)
	scheduler.Start()
	time.Sleep(1 * time.Second)
	scheduler.Stop()

	// 验证停止后上下文是否已取消
	select {
	case <-scheduler.ctx.Done():
		assert.True(t, true, "Context should be canceled after scheduler is stopped")
	default:
		t.Fatal("Context should be canceled after scheduler is stopped")
	}
}

func TestGetExecutingJobs(t *testing.T) {
	scheduler := setupTestScheduler(t)
	defer scheduler.Stop()

	job := createTestJob("testjob", "echo test", "*/1 * * * * *", false)

	jobInfo := &common.JobExecuteInfo{
		Job:      job,
		PlanTime: time.Now(),
		RealTime: time.Now(),
	}

	scheduler.jobExecuting["testjob"] = jobInfo

	executingJobs := scheduler.GetExecutingJobs()
	assert.Equal(t, 1, len(executingJobs), "Should have 1 executing job")
	assert.Equal(t, "testjob", executingJobs["testjob"].Job.Name, "Executing job name should match")
}

func TestKillJob(t *testing.T) {
	scheduler := setupTestScheduler(t)
	defer scheduler.Stop()

	// 设置一个不存在的任务
	err := scheduler.KillJob("nonexistentjob")
	assert.Error(t, err, "Should return error when killing non-existent job")

	// 设置一个正在执行的任务
	job := createTestJob("testjob", "sleep 10", "*/1 * * * * *", false)
	ctx, cancel := context.WithCancel(context.Background())

	jobInfo := &common.JobExecuteInfo{
		Job:        job,
		PlanTime:   time.Now(),
		RealTime:   time.Now(),
		CancelCtx:  ctx,
		CancelFunc: cancel,
	}

	scheduler.jobExecuting["testjob"] = jobInfo

	// 测试Kill
	err = scheduler.KillJob("testjob")
	assert.NoError(t, err, "Should not return error when killing existing job")
}

func setupTestEtcd() (*etcd.Client, error) {
	config.GlobalConfig = &config.Config{
		EtcdEndpoints:   []string{"localhost:2379"},
		EtcdDialTimeout: 5000,
		JobLockTTL:      5,
	}

	return etcd.NewClient()
}

func setupTestScheduler(t *testing.T) *Scheduler {
	logger, _ := zap.NewDevelopment()

	etcdClient, err := setupTestEtcd()
	require.NoError(t, err, "Failed to setup test ETCD")

	exec := executor.NewExecutor(logger)
	jobMan := jobmgr.NewJobManager(etcdClient, logger)

	return NewScheduler(logger, jobMan, etcdClient, exec)
}
