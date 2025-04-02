package scheduler

import (
	"context"
	"time"

	"github.com/gorhill/cronexpr"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
	"github.com/fyerfyer/scheduler-refactor/worker/executor"
	"github.com/fyerfyer/scheduler-refactor/worker/joblock"
	"github.com/fyerfyer/scheduler-refactor/worker/jobmgr"
)

// JobSchedulePlan 任务调度计划
type JobSchedulePlan struct {
	Job      *common.Job          // 任务信息
	Expr     *cronexpr.Expression // cron表达式
	NextTime time.Time            // 下次调度时间
}

// Scheduler 任务调度器
type Scheduler struct {
	logger        *zap.Logger                       // 日志对象
	jobManager    *jobmgr.JobManager                // 任务管理器
	etcdClient    *etcd.Client                      // etcd客户端
	jobPlans      map[string]*JobSchedulePlan       // 任务调度计划表
	jobExecuting  map[string]*common.JobExecuteInfo // 正在执行的任务
	jobResultChan <-chan *common.JobExecuteResult   // 任务执行结果通道
	jobEventChan  <-chan *common.JobEvent           // 任务事件通道
	executor      *executor.Executor                // 任务执行器
	planChan      chan *JobSchedulePlan             // 新调度任务通道
	ctx           context.Context                   // 上下文，用于控制退出
	cancelFunc    context.CancelFunc                // 取消函数
}

// NewScheduler 创建调度器
func NewScheduler(
	logger *zap.Logger,
	jobManager *jobmgr.JobManager,
	etcdClient *etcd.Client,
	exec *executor.Executor,
) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	scheduler := &Scheduler{
		logger:        logger,
		jobManager:    jobManager,
		etcdClient:    etcdClient,
		jobPlans:      make(map[string]*JobSchedulePlan),
		jobExecuting:  make(map[string]*common.JobExecuteInfo),
		jobResultChan: exec.GetResultChan(),
		jobEventChan:  jobManager.GetEventChan(),
		executor:      exec,
		planChan:      make(chan *JobSchedulePlan, 100),
		ctx:           ctx,
		cancelFunc:    cancel,
	}

	return scheduler
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.logger.Info("scheduler starting...")

	// 加载所有任务
	s.loadJobs()

	// 启动调度协程
	go s.scheduleLoop()
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cancelFunc()
	s.logger.Info("scheduler stopped")
}

// loadJobs 加载所有任务
func (s *Scheduler) loadJobs() {
	jobs := s.jobManager.ListJobs()
	for _, job := range jobs {
		// 过滤禁用的任务
		if job.Disabled {
			continue
		}

		// 解析cron表达式
		expr, err := cronexpr.Parse(job.CronExpr)
		if err != nil {
			s.logger.Error("failed to parse cron expression",
				zap.String("jobName", job.Name),
				zap.String("cronExpr", job.CronExpr),
				zap.Error(err))
			continue
		}

		// 计算任务下次执行时间
		schedPlan := &JobSchedulePlan{
			Job:      job,
			Expr:     expr,
			NextTime: expr.Next(time.Now()),
		}

		// 添加到调度计划表
		s.jobPlans[job.Name] = schedPlan

		s.logger.Info("job loaded into schedule",
			zap.String("jobName", job.Name),
			zap.String("nextTime", schedPlan.NextTime.Format("2006-01-02 15:04:05")))
	}
}

// handleJobEvent 处理任务事件
func (s *Scheduler) handleJobEvent(event *common.JobEvent) {
	switch event.EventType {
	case common.JobEventSave: // 保存任务事件
		job := event.Job

		// 跳过禁用的任务
		if job.Disabled {
			// 如果任务已在调度计划中，则移除它
			if _, exists := s.jobPlans[job.Name]; exists {
				delete(s.jobPlans, job.Name)
				s.logger.Info("job disabled and removed from schedule",
					zap.String("jobName", job.Name))
			}
			return
		}

		// 解析cron表达式
		expr, err := cronexpr.Parse(job.CronExpr)
		if err != nil {
			s.logger.Error("failed to parse cron expression",
				zap.String("jobName", job.Name),
				zap.String("cronExpr", job.CronExpr),
				zap.Error(err))
			return
		}

		// 构建调度计划
		schedPlan := &JobSchedulePlan{
			Job:      job,
			Expr:     expr,
			NextTime: expr.Next(time.Now()),
		}

		// 更新调度计划
		s.jobPlans[job.Name] = schedPlan

		s.logger.Info("job saved and scheduled",
			zap.String("jobName", job.Name),
			zap.String("nextTime", schedPlan.NextTime.Format("2006-01-02 15:04:05")))

	case common.JobEventDelete: // 删除任务事件
		// 从调度计划表中删除任务
		if _, exists := s.jobPlans[event.Job.Name]; exists {
			delete(s.jobPlans, event.Job.Name)
			s.logger.Info("job removed from schedule", zap.String("jobName", event.Job.Name))
		}
	}
}

// handleJobResult 处理任务执行结果
func (s *Scheduler) handleJobResult(result *common.JobExecuteResult) {
	// 从执行任务表中删除
	delete(s.jobExecuting, result.JobName)

	s.logger.Info("job execution finished",
		zap.String("jobName", result.JobName),
		zap.String("startTime", result.StartTime.Format("2006-01-02 15:04:05")),
		zap.String("endTime", result.EndTime.Format("2006-01-02 15:04:05")),
		zap.String("output", result.Output),
		zap.String("error", result.Error))
}

// scheduleLoop 调度循环
func (s *Scheduler) scheduleLoop() {
	// 使用ticker进行时间推进，每100ms检查一次
	scheduleTicker := time.NewTicker(100 * time.Millisecond)
	defer scheduleTicker.Stop()

	// 调度循环
	for {
		select {
		case <-s.ctx.Done(): // 上下文被取消，退出调度
			return
		case event := <-s.jobEventChan: // 处理任务事件
			s.handleJobEvent(event)
		case result := <-s.jobResultChan: // 处理任务结果
			s.handleJobResult(result)
		case <-scheduleTicker.C: // 定时调度检查
			s.trySchedule()
		}
	}
}

// trySchedule 尝试执行调度
func (s *Scheduler) trySchedule() {
	// 当前时间
	now := time.Now()

	// 有任务需要执行时的最近时间点
	var nearTime *time.Time

	// 遍历所有调度计划
	for _, plan := range s.jobPlans {
		// 如果任务的调度时间已到
		if plan.NextTime.Before(now) || plan.NextTime.Equal(now) {
			// 尝试执行任务
			s.tryStartJob(plan)

			// 计算任务下次执行时间
			plan.NextTime = plan.Expr.Next(now)
		}

		// 更新最近要执行的任务时间
		if nearTime == nil || plan.NextTime.Before(*nearTime) {
			nt := plan.NextTime
			nearTime = &nt
		}
	}
}

// tryStartJob 尝试启动任务
func (s *Scheduler) tryStartJob(plan *JobSchedulePlan) {
	// 如果任务正在执行，跳过本次调度
	if _, executing := s.jobExecuting[plan.Job.Name]; executing {
		s.logger.Info("job is already executing, skipping schedule",
			zap.String("jobName", plan.Job.Name))
		return
	}

	// 执行任务前，先获取分布式锁
	jobLock := joblock.NewJobLock(s.etcdClient, plan.Job.Name)

	// 尝试获取锁
	err := jobLock.TryLock()
	if err != nil {
		// 获取锁失败，跳过本次调度
		s.logger.Debug("failed to acquire job lock, skipping execution",
			zap.String("jobName", plan.Job.Name),
			zap.Error(err))
		return
	}

	// 构建执行状态信息
	jobExecuteInfo := &common.JobExecuteInfo{
		Job:      plan.Job,
		PlanTime: plan.NextTime,
		RealTime: time.Now(),
	}

	// 保存执行状态
	s.jobExecuting[plan.Job.Name] = jobExecuteInfo

	// 执行任务
	s.executor.ExecuteJob(jobExecuteInfo)

	s.logger.Info("job scheduled for execution",
		zap.String("jobName", plan.Job.Name),
		zap.String("planTime", plan.NextTime.Format("2006-01-02 15:04:05")),
		zap.String("realTime", jobExecuteInfo.RealTime.Format("2006-01-02 15:04:05")))

	// 任务启动后释放锁
	// 注意: 这里我们在任务开始后立即释放锁，允许其他节点在下一次调度时获取锁
	// 真实场景可能需要根据任务特性决定是否在任务结束后释放锁
	jobLock.Unlock()
}

// GetExecutingJobs 获取正在执行的任务
func (s *Scheduler) GetExecutingJobs() map[string]*common.JobExecuteInfo {
	return s.jobExecuting
}

// KillJob 强制终止任务
func (s *Scheduler) KillJob(jobName string) error {
	// 查找是否有该任务正在执行
	if jobInfo, exists := s.jobExecuting[jobName]; exists {
		// 调用执行器的KillJob方法终止任务
		s.executor.KillJob(jobName, jobInfo)
		return nil
	}

	return common.NewJobError(jobName, common.ErrJobNotFound)
}
