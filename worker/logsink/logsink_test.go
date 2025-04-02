package logsink

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/mongodb"
)

func setupTest(t *testing.T) (*mongodb.Client, *zap.Logger) {
	// 确保全局配置已初始化
	if config.GlobalConfig == nil {
		config.GlobalConfig = &config.Config{
			MongoURI:            "mongodb://localhost:27017",
			MongoConnectTimeout: 5000,
			LogBatchSize:        10,
			LogCommitTimeout:    1000,
			WorkerID:            "test-worker",
		}
	}

	// 创建MongoDB客户端
	client, err := mongodb.NewClient()
	require.NoError(t, err, "Failed to create MongoDB client")

	// 创建日志对象
	logger, _ := zap.NewDevelopment()

	return client, logger
}

func createTestJobLog() *common.JobLog {
	now := time.Now().Unix()
	return &common.JobLog{
		JobName:      "test_job",
		Command:      "echo hello",
		Output:       "hello\n",
		Error:        "",
		PlanTime:     now - 10,
		ScheduleTime: now - 8,
		StartTime:    now - 7,
		EndTime:      now,
		ExitCode:     0,
		IsTimeout:    false,
		WorkerIP:     "test-worker",
	}
}

func TestNewLogSink(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	logSink := NewLogSink(client, logger)
	assert.NotNil(t, logSink, "LogSink should not be nil")
	assert.Equal(t, config.GlobalConfig.LogBatchSize, logSink.batchSize, "Batch size should match config")
	assert.NotNil(t, logSink.logChan, "Log channel should be initialized")
	assert.NotNil(t, logSink.logBatch, "Log batch should be initialized")

	// 清理资源
	logSink.Stop()
}

func TestLogSink_Append(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	logSink := NewLogSink(client, logger)
	defer logSink.Stop()

	// 测试添加日志
	jobLog := createTestJobLog()
	logSink.Append(jobLog)

	// 检查日志是否被添加到通道
	assert.Equal(t, 1, len(logSink.logChan), "Log should be appended to channel")
}

func TestLogSink_CommitLogs(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	// 配置较小的批次大小以便测试
	config.GlobalConfig.LogBatchSize = 3

	logSink := NewLogSink(client, logger)
	defer logSink.Stop()

	// 添加足够多的日志以触发自动提交
	for i := 0; i < config.GlobalConfig.LogBatchSize; i++ {
		jobLog := createTestJobLog()
		jobLog.JobName = jobLog.JobName + "_" + time.Now().String()
		logSink.Append(jobLog)
	}

	// 等待日志被批量提交
	time.Sleep(2 * time.Second)

	// 通过检查logBatch是否为空来验证日志已提交
	assert.Equal(t, 0, len(logSink.logBatch), "Log batch should be empty after commit")
}

func TestLogSink_CommitTimeout(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	// 配置短的提交超时以便测试
	config.GlobalConfig.LogCommitTimeout = 100 // 100ms

	logSink := NewLogSink(client, logger)
	defer logSink.Stop()

	// 添加一条日志（不足以触发批量提交）
	jobLog := createTestJobLog()
	logSink.Append(jobLog)

	// 等待超时提交
	time.Sleep(200 * time.Millisecond)

	// 通过检查logBatch是否为空来验证日志已提交
	assert.Equal(t, 0, len(logSink.logBatch), "Log batch should be empty after timeout commit")
}

func TestLogSink_Stop(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	logSink := NewLogSink(client, logger)

	// 添加一些日志但不足以触发批量提交
	jobLog := createTestJobLog()
	logSink.logBatch = append(logSink.logBatch, jobLog)

	// 停止应该触发剩余日志的提交
	logSink.Stop()

	// 通过检查logBatch是否为空来验证日志已提交
	assert.Equal(t, 0, len(logSink.logBatch), "Log batch should be empty after stop")
}

func TestLogSink_CleanExpiredLogs(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	logSink := NewLogSink(client, logger)
	defer logSink.Stop()

	// 创建一些测试日志
	oldJobLog := createTestJobLog()
	oldJobLog.EndTime = time.Now().AddDate(0, 0, -40).Unix() // 40天前的日志

	newJobLog := createTestJobLog()
	newJobLog.EndTime = time.Now().Unix() // 今天的日志

	// 将日志直接添加到MongoDB（跳过批处理机制以便测试）
	_, _ = client.InsertOne(oldJobLog)
	_, _ = client.InsertOne(newJobLog)

	// 清理30天前的日志
	logSink.CleanExpiredLogs(30)

	// 等待清理完成
	time.Sleep(500 * time.Millisecond)

	// 查询旧日志，应该已被删除
	logs, err := client.FindJobLogs(oldJobLog.JobName, 0, 10)
	require.NoError(t, err, "Query should not fail")

	// 检查是否还能找到旧日志
	found := false
	for _, log := range logs {
		if log.EndTime == oldJobLog.EndTime {
			found = true
			break
		}
	}
	assert.False(t, found, "Old log should be cleaned")
}

func TestLogSink_StartLogCleaner(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	logSink := NewLogSink(client, logger)

	// 创建一个上下文，可以手动取消
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动日志清理器，使用小的留存期以方便测试
	logSink.StartLogCleaner(ctx, 1)

	// 测试取消上下文能否正确停止清理器
	cancel()
	time.Sleep(100 * time.Millisecond)

	// 由于我们已经取消了上下文，清理器应该已经停止
	// 这个测试主要确保StartLogCleaner不会出错或崩溃

	// 清理资源
	logSink.Stop()
}

func TestLogSink_ChannelOverflow(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	// 创建一个很小的通道容量
	smallCapacity := 5

	// 手动创建LogSink以使用小容量通道
	logSink := &LogSink{
		client:    client,
		logChan:   make(chan *common.JobLog, smallCapacity),
		logBatch:  make([]*common.JobLog, 0, config.GlobalConfig.LogBatchSize),
		logger:    logger,
		batchSize: config.GlobalConfig.LogBatchSize,
	}

	// 不启动worker，以测试通道溢出

	// 添加足够多的日志以填满通道
	for i := 0; i < smallCapacity; i++ {
		jobLog := createTestJobLog()
		logSink.Append(jobLog)
	}

	// 再添加一条，这条应该会被丢弃
	extraLog := createTestJobLog()
	logSink.Append(extraLog)

	// 验证通道大小等于其容量
	assert.Equal(t, smallCapacity, len(logSink.logChan), "Channel should be full")
}
