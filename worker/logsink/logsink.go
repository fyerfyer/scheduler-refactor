package logsink

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/mongodb"
)

// LogSink 日志收集器
type LogSink struct {
	client      *mongodb.Client     // MongoDB客户端
	logChan     chan *common.JobLog // 日志通道
	logBatch    []*common.JobLog    // 日志批次暂存
	logger      *zap.Logger         // 日志对象
	batchSize   int                 // 批处理大小
	commitTimer *time.Timer         // 自动提交定时器
}

// NewLogSink 创建日志收集器
func NewLogSink(mongoClient *mongodb.Client, logger *zap.Logger) *LogSink {
	logSink := &LogSink{
		client:    mongoClient,
		logChan:   make(chan *common.JobLog, 1000),
		logBatch:  make([]*common.JobLog, 0, config.GlobalConfig.LogBatchSize),
		logger:    logger,
		batchSize: config.GlobalConfig.LogBatchSize,
	}

	// 启动日志收集协程
	logSink.startWorker()

	return logSink
}

// startWorker 启动日志收集协程
func (l *LogSink) startWorker() {
	go func() {
		// 初始化自动提交定时器
		l.commitTimer = time.NewTimer(time.Duration(config.GlobalConfig.LogCommitTimeout) * time.Millisecond)

		for {
			select {
			case log := <-l.logChan: // 收到一条日志
				// 追加到批次中
				l.logBatch = append(l.logBatch, log)

				// 如果批次已满，立即提交
				if len(l.logBatch) >= l.batchSize {
					l.commitLogs()
					// 重置定时器
					l.commitTimer.Reset(time.Duration(config.GlobalConfig.LogCommitTimeout) * time.Millisecond)
				}

			case <-l.commitTimer.C: // 提交超时
				// 有日志就提交
				if len(l.logBatch) > 0 {
					l.commitLogs()
				}
				// 重置定时器
				l.commitTimer.Reset(time.Duration(config.GlobalConfig.LogCommitTimeout) * time.Millisecond)
			}
		}
	}()
}

// Append 追加日志
func (l *LogSink) Append(jobLog *common.JobLog) {
	select {
	case l.logChan <- jobLog:
		// 投递成功
	default:
		// 通道满了，日志丢弃，记录错误
		l.logger.Error("log channel is full, log discarded",
			zap.String("jobName", jobLog.JobName),
			zap.Int64("startTime", jobLog.StartTime),
			zap.Int64("endTime", jobLog.EndTime))
	}
}

// commitLogs 批量提交日志
func (l *LogSink) commitLogs() {
	// 如果没有日志，直接返回
	if len(l.logBatch) == 0 {
		return
	}

	// 批量插入mongo
	logs := make([]interface{}, len(l.logBatch))
	for i, log := range l.logBatch {
		logs[i] = log
	}

	// 执行批量插入
	_, err := l.client.InsertMany(logs)
	if err != nil {
		l.logger.Error("failed to commit logs",
			zap.Int("count", len(logs)),
			zap.Error(err))
	} else {
		l.logger.Info("committed logs",
			zap.Int("count", len(logs)))
	}

	// 清空批次
	l.logBatch = l.logBatch[:0]
}

// Stop 停止日志收集器
func (l *LogSink) Stop() {
	// 立即提交当前批次的日志
	if len(l.logBatch) > 0 {
		l.commitLogs()
	}
}

// CleanExpiredLogs 清理过期日志
func (l *LogSink) CleanExpiredLogs(retentionDays int) {
	// 默认保留30天的日志
	if retentionDays <= 0 {
		retentionDays = 30
	}

	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// 执行清理
	deletedCount, err := l.client.DeleteOldLogs(cutoffTime)
	if err != nil {
		l.logger.Error("failed to clean expired logs",
			zap.Time("before", cutoffTime),
			zap.Int("retentionDays", retentionDays),
			zap.Error(err))
	} else if deletedCount > 0 {
		l.logger.Info("cleaned expired logs",
			zap.Time("before", cutoffTime),
			zap.Int("retentionDays", retentionDays),
			zap.Int64("deletedCount", deletedCount))
	}
}

// StartLogCleaner 启动定期清理过期日志的协程
func (l *LogSink) StartLogCleaner(ctx context.Context, retentionDays int) {
	go func() {
		// 创建一个每天执行的定时器（凌晨3点执行）
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		// 计算下次执行的时间（今天或明天的凌晨3点）
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		// 第一次执行的定时器
		timer := time.NewTimer(nextRun.Sub(now))
		defer timer.Stop()

		// 先执行一次清理
		l.CleanExpiredLogs(retentionDays)

		for {
			select {
			case <-timer.C:
				// 第一次到时间后，使用ticker
				ticker.Reset(24 * time.Hour)
				l.CleanExpiredLogs(retentionDays)

			case <-ticker.C:
				// 后续每24小时执行一次
				l.CleanExpiredLogs(retentionDays)

			case <-ctx.Done():
				// 上下文取消，退出协程
				l.logger.Info("log cleaner stopped")
				return
			}
		}
	}()

	l.logger.Info("log cleaner started", zap.Int("retentionDays", retentionDays))
}

// GetLogChan 获取日志通道，用于测试
func (l *LogSink) GetLogChan() chan<- *common.JobLog {
	return l.logChan
}
