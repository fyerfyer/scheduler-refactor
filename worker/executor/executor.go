package executor

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"runtime"
	"time"

	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
)

// Executor 任务执行器
type Executor struct {
	logger     *zap.Logger                   // 日志对象
	jobResults chan *common.JobExecuteResult // 任务执行结果通道
}

// NewExecutor 创建执行器
func NewExecutor(logger *zap.Logger) *Executor {
	return &Executor{
		logger:     logger,
		jobResults: make(chan *common.JobExecuteResult, 1000), // 结果缓冲区
	}
}

// ExecuteJob 执行一个任务
func (e *Executor) ExecuteJob(info *common.JobExecuteInfo) {
	go func() {
		// 记录任务开始执行时间
		startTime := time.Now()

		// 结果对象
		result := &common.JobExecuteResult{
			JobName:   info.Job.Name,
			StartTime: startTime,
		}

		// 创建上下文（用于任务超时控制）
		var ctx context.Context
		var cancel context.CancelFunc

		// 设置超时
		if info.Job.Timeout > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), time.Duration(info.Job.Timeout)*time.Second)
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}
		defer cancel()

		// 保存上下文到执行信息中，方便外部取消任务
		info.CancelCtx = ctx
		info.CancelFunc = cancel

		// 执行命令并捕获输出
		var cmd *exec.Cmd
		var output bytes.Buffer
		var errOutput bytes.Buffer

		// 根据不同系统执行命令
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "cmd", "/C", info.Job.Command)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", info.Job.Command)
		}

		// 捕获输出
		cmd.Stdout = &output
		cmd.Stderr = &errOutput

		// 执行命令
		err := cmd.Run()

		// 记录结束时间
		endTime := time.Now()

		// 设置结果信息
		result.EndTime = endTime
		result.Output = output.String()

		// 处理执行结果
		if err != nil {
			// 检查是否因为超时被取消
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				result.Error = "job execution timed out"
				result.IsTimeout = true
				result.ExitCode = -1
			} else {
				result.Error = err.Error()
				if exitErr, ok := err.(*exec.ExitError); ok {
					result.ExitCode = exitErr.ExitCode()
				} else {
					result.ExitCode = -1
				}
			}

			e.logger.Warn("job execution failed",
				zap.String("jobName", info.Job.Name),
				zap.String("error", result.Error),
				zap.Int("exitCode", result.ExitCode))
		} else {
			result.ExitCode = 0
			e.logger.Info("job executed successfully",
				zap.String("jobName", info.Job.Name),
				zap.Duration("duration", endTime.Sub(startTime)))
		}

		// 将结果投递到结果通道
		e.jobResults <- result
	}()
}

// KillJob 强制终止任务
func (e *Executor) KillJob(jobName string, info *common.JobExecuteInfo) {
	if info != nil && info.CancelFunc != nil {
		// 调用取消函数
		cancelFunc, ok := info.CancelFunc.(context.CancelFunc)
		if ok {
			cancelFunc()
			e.logger.Info("job killed by user request",
				zap.String("jobName", jobName))
		}
	}
}

// GetResultChan 获取任务结果通道
func (e *Executor) GetResultChan() <-chan *common.JobExecuteResult {
	return e.jobResults
}

// BuildJobLog 构建任务执行日志
func BuildJobLog(result *common.JobExecuteResult, info *common.JobExecuteInfo) *common.JobLog {
	jobLog := &common.JobLog{
		JobName:      result.JobName,
		Command:      info.Job.Command,
		Output:       result.Output,
		Error:        result.Error,
		PlanTime:     info.PlanTime.Unix(),
		ScheduleTime: info.RealTime.Unix(),
		StartTime:    result.StartTime.Unix(),
		EndTime:      result.EndTime.Unix(),
		ExitCode:     result.ExitCode,
		IsTimeout:    result.IsTimeout,
		WorkerIP:     config.GlobalConfig.WorkerID, // 使用WorkerID作为标识
	}

	return jobLog
}
