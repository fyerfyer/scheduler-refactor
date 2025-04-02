package logmgr

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/pkg/mongodb"
)

// LogManager 日志管理器，负责任务日志的查询和管理
type LogManager struct {
	mongoClient *mongodb.Client // MongoDB客户端
	logger      *zap.Logger     // 日志对象
	ctx         context.Context // 上下文，用于控制退出
	cancelFunc  context.CancelFunc
}

// NewLogManager 创建日志管理器
func NewLogManager(mongoClient *mongodb.Client, logger *zap.Logger) *LogManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &LogManager{
		mongoClient: mongoClient,
		logger:      logger,
		ctx:         ctx,
		cancelFunc:  cancel,
	}
}

// ListLogs 获取任务日志列表
func (lm *LogManager) ListLogs(jobName string, page, pageSize int) ([]*common.JobLog, int64, error) {
	// 参数校验
	if page <= 0 {
		page = common.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = common.DefaultPageSize
	}
	if pageSize > common.MaxPageSize {
		pageSize = common.MaxPageSize
	}

	// 计算分页
	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	// 查询日志
	logs, err := lm.mongoClient.FindJobLogs(jobName, skip, limit)
	if err != nil {
		lm.logger.Error("failed to fetch job logs",
			zap.String("jobName", jobName),
			zap.Int("page", page),
			zap.Int("pageSize", pageSize),
			zap.Error(err))
		return nil, 0, err
	}

	// 获取总数
	total, err := lm.mongoClient.CountJobLogs(jobName)
	if err != nil {
		lm.logger.Error("failed to count job logs",
			zap.String("jobName", jobName),
			zap.Error(err))
		return logs, 0, err
	}

	return logs, total, nil
}

// GetJobLog 获取指定任务的最近一条日志
func (lm *LogManager) GetJobLog(jobName string) (*common.JobLog, error) {
	// 查询最近一条日志
	logs, err := lm.mongoClient.FindJobLogs(jobName, 0, 1)
	if err != nil {
		lm.logger.Error("failed to fetch latest job log",
			zap.String("jobName", jobName),
			zap.Error(err))
		return nil, err
	}

	// 检查是否有日志
	if len(logs) == 0 {
		return nil, common.ErrJobNotFound
	}

	return logs[0], nil
}

// CleanExpiredLogs 清理过期日志
func (lm *LogManager) CleanExpiredLogs(retentionDays int) error {
	// 默认保留30天的日志
	if retentionDays <= 0 {
		retentionDays = 30
	}

	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// 执行清理
	deletedCount, err := lm.mongoClient.DeleteOldLogs(cutoffTime)
	if err != nil {
		lm.logger.Error("failed to clean expired logs",
			zap.Time("before", cutoffTime),
			zap.Int("retentionDays", retentionDays),
			zap.Error(err))
		return err
	}

	lm.logger.Info("cleaned expired logs",
		zap.Time("before", cutoffTime),
		zap.Int("retentionDays", retentionDays),
		zap.Int64("deletedCount", deletedCount))

	return nil
}

// GetLogStatistics 获取任务日志统计信息
func (lm *LogManager) GetLogStatistics(jobName string, days int) (map[string]interface{}, error) {
	// 默认统计最近7天
	if days <= 0 {
		days = 7
	}

	// 计算起始时间
	startTime := time.Now().AddDate(0, 0, -days).Unix()

	// 获取日志
	logs, err := lm.getLogsSince(jobName, startTime)
	if err != nil {
		return nil, err
	}

	// 统计成功、失败数量
	successCount := 0
	failCount := 0
	timeoutCount := 0
	totalDuration := int64(0)

	for _, log := range logs {
		if log.ExitCode == 0 {
			successCount++
		} else {
			failCount++
		}

		if log.IsTimeout {
			timeoutCount++
		}

		// 计算执行时长
		duration := log.EndTime - log.StartTime
		totalDuration += duration
	}

	// 计算平均执行时长
	var avgDuration float64
	if len(logs) > 0 {
		avgDuration = float64(totalDuration) / float64(len(logs))
	}

	// 构建统计结果
	stats := map[string]interface{}{
		"totalCount":   len(logs),
		"successCount": successCount,
		"failCount":    failCount,
		"timeoutCount": timeoutCount,
		"avgDuration":  avgDuration, // 单位：秒
		"period":       days,
	}

	return stats, nil
}

// getLogsSince 获取指定时间之后的日志
func (lm *LogManager) getLogsSince(jobName string, timestamp int64) ([]*common.JobLog, error) {
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 从MongoDB中获取日志数据
	// 注意：这里需要扩展mongodb.Client以支持这种查询
	logs, err := lm.mongoClient.FindJobLogsSince(jobName, timestamp)
	if err != nil {
		lm.logger.Error("failed to get logs since timestamp",
			zap.String("jobName", jobName),
			zap.Int64("since", timestamp),
			zap.Error(err))
		return nil, err
	}

	return logs, nil
}

// Stop 停止日志管理器
func (lm *LogManager) Stop() {
	lm.cancelFunc()
	lm.logger.Info("log manager stopped")
}

// StartLogCleaner 启动日志清理器
func (lm *LogManager) StartLogCleaner(retentionDays int) {
	go func() {
		// 定期清理日志，每天运行一次
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-lm.ctx.Done():
				// 上下文被取消，退出清理
				return
			case <-ticker.C:
				// 运行日志清理
				if err := lm.CleanExpiredLogs(retentionDays); err != nil {
					lm.logger.Error("periodic log cleaning failed", zap.Error(err))
				}
			}
		}
	}()
}
