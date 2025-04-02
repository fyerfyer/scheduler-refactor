package executor

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
)

func setupTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestExecutor_ExecuteJob_Success(t *testing.T) {
	logger := setupTestLogger()
	executor := NewExecutor(logger)

	job := &common.Job{
		Name:      "test_success_job",
		Command:   "echo hello world",
		CronExpr:  "*/5 * * * * *",
		Timeout:   10,
		Disabled:  false,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	jobInfo := &common.JobExecuteInfo{
		Job:      job,
		PlanTime: time.Now(),
		RealTime: time.Now(),
	}

	executor.ExecuteJob(jobInfo)

	select {
	case result := <-executor.GetResultChan():
		assert.Equal(t, job.Name, result.JobName)
		assert.Equal(t, "hello world\r\n", result.Output)
		assert.Equal(t, "", result.Error)
		assert.Equal(t, 0, result.ExitCode)
		assert.False(t, result.IsTimeout)
	case <-time.After(3 * time.Second):
		t.Fatal("execution timeout")
	}
}

func TestExecutor_ExecuteJob_Error(t *testing.T) {
	logger := setupTestLogger()
	executor := NewExecutor(logger)

	job := &common.Job{
		Name:      "test_error_job",
		Command:   "non_existent_command",
		CronExpr:  "*/5 * * * * *",
		Timeout:   10,
		Disabled:  false,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	jobInfo := &common.JobExecuteInfo{
		Job:      job,
		PlanTime: time.Now(),
		RealTime: time.Now(),
	}

	executor.ExecuteJob(jobInfo)

	select {
	case result := <-executor.GetResultChan():
		assert.Equal(t, job.Name, result.JobName)
		assert.NotEmpty(t, result.Error)
		assert.NotEqual(t, 0, result.ExitCode)
	case <-time.After(3 * time.Second):
		t.Fatal("execution timeout")
	}
}

func TestExecutor_ExecuteJob_Timeout(t *testing.T) {
	if os.Getenv("SKIP_SLOW_TESTS") == "1" {
		t.Skip("Skipping slow test")
	}

	logger := setupTestLogger()
	executor := NewExecutor(logger)

	job := &common.Job{
		Name:      "test_timeout_job",
		Command:   "sleep 5",
		CronExpr:  "*/5 * * * * *",
		Timeout:   1, // 1秒超时
		Disabled:  false,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	jobInfo := &common.JobExecuteInfo{
		Job:      job,
		PlanTime: time.Now(),
		RealTime: time.Now(),
	}

	executor.ExecuteJob(jobInfo)

	select {
	case result := <-executor.GetResultChan():
		assert.Equal(t, job.Name, result.JobName)
		assert.True(t, result.IsTimeout)
		assert.Contains(t, result.Error, "timed out")
	case <-time.After(3 * time.Second):
		t.Fatal("execution timeout")
	}
}

func TestExecutor_KillJob(t *testing.T) {
	logger := setupTestLogger()
	executor := NewExecutor(logger)

	job := &common.Job{
		Name:      "test_kill_job",
		Command:   "sleep 10",
		CronExpr:  "*/5 * * * * *",
		Timeout:   30,
		Disabled:  false,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	jobInfo := &common.JobExecuteInfo{
		Job:        job,
		PlanTime:   time.Now(),
		RealTime:   time.Now(),
		CancelCtx:  ctx,
		CancelFunc: cancel,
	}

	executor.ExecuteJob(jobInfo)

	time.Sleep(500 * time.Millisecond)
	executor.KillJob(job.Name, jobInfo)

	select {
	case result := <-executor.GetResultChan():
		assert.Equal(t, job.Name, result.JobName)
		assert.NotEqual(t, 0, result.ExitCode)
	case <-time.After(3 * time.Second):
		t.Fatal("failed to kill job")
	}
}

func TestBuildJobLog(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-5 * time.Second)

	job := &common.Job{
		Name:      "test_job_log",
		Command:   "echo hello",
		CronExpr:  "*/5 * * * * *",
		Timeout:   10,
		Disabled:  false,
		CreatedAt: now.Add(-1 * time.Hour).Unix(),
		UpdatedAt: now.Add(-30 * time.Minute).Unix(),
	}

	planTime := now.Add(-10 * time.Second)
	realTime := now.Add(-8 * time.Second)

	jobInfo := &common.JobExecuteInfo{
		Job:      job,
		PlanTime: planTime,
		RealTime: realTime,
	}

	result := &common.JobExecuteResult{
		JobName:   job.Name,
		Output:    "hello\n",
		Error:     "",
		StartTime: startTime,
		EndTime:   now,
		ExitCode:  0,
		IsTimeout: false,
	}

	jobLog := BuildJobLog(result, jobInfo)

	assert.Equal(t, job.Name, jobLog.JobName)
	assert.Equal(t, job.Command, jobLog.Command)
	assert.Equal(t, result.Output, jobLog.Output)
	assert.Equal(t, result.Error, jobLog.Error)
	assert.Equal(t, planTime.Unix(), jobLog.PlanTime)
	assert.Equal(t, realTime.Unix(), jobLog.ScheduleTime)
	assert.Equal(t, startTime.Unix(), jobLog.StartTime)
	assert.Equal(t, now.Unix(), jobLog.EndTime)
	assert.Equal(t, 0, jobLog.ExitCode)
	assert.False(t, jobLog.IsTimeout)
}
