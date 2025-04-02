package logmgr

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/mongodb"
)

func setupTestEnv(t *testing.T) (*LogManager, *mongodb.Client, func()) {
	logger := zaptest.NewLogger(t)

	config.GlobalConfig = &config.Config{
		MongoURI:            "mongodb://localhost:27017",
		MongoConnectTimeout: 5000,
	}

	mongoClient, err := mongodb.NewClient()
	require.NoError(t, err, "Failed to create MongoDB client")

	logMgr := NewLogManager(mongoClient, logger)
	require.NotNil(t, logMgr, "LogManager should not be nil")

	cleanup := func() {
		logMgr.Stop()
		mongoClient.DropCollection()
		mongoClient.Close()
	}

	return logMgr, mongoClient, cleanup
}

func insertTestLogs(t *testing.T, client *mongodb.Client, count int, jobName string) {
	logs := make([]interface{}, 0, count)
	now := time.Now().Unix()

	for i := 0; i < count; i++ {
		log := &common.JobLog{
			JobName:      jobName,
			Command:      "echo test",
			Output:       "test output",
			PlanTime:     now - int64(i*60),
			ScheduleTime: now - int64(i*60) + 1,
			StartTime:    now - int64(i*60) + 2,
			EndTime:      now - int64(i*60) + 5,
			ExitCode:     i % 2, // 偶数索引的记录成功，奇数索引的记录失败
			IsTimeout:    i%5 == 0,
			WorkerIP:     "test-worker",
		}
		logs = append(logs, log)
	}

	_, err := client.InsertMany(logs)
	require.NoError(t, err, "Failed to insert test logs")
}

func TestListLogs(t *testing.T) {
	logMgr, mongoClient, cleanup := setupTestEnv(t)
	defer cleanup()

	jobName := "test-job"
	insertTestLogs(t, mongoClient, 25, jobName)

	t.Run("DefaultPagination", func(t *testing.T) {
		logs, total, err := logMgr.ListLogs(jobName, 0, 0)
		require.NoError(t, err, "ListLogs should not return error with default pagination")
		assert.Equal(t, int64(25), total, "Total count should match inserted logs count")
		assert.Equal(t, common.DefaultPageSize, len(logs), "Should return DefaultPageSize logs")
	})

	t.Run("CustomPagination", func(t *testing.T) {
		logs, total, err := logMgr.ListLogs(jobName, 2, 5)
		require.NoError(t, err, "ListLogs should not return error with custom pagination")
		assert.Equal(t, int64(25), total, "Total count should match inserted logs count")
		assert.Equal(t, 5, len(logs), "Should return specified page size")
	})

	t.Run("LimitMaxPageSize", func(t *testing.T) {
		logs, _, err := logMgr.ListLogs(jobName, 1, 200)
		require.NoError(t, err, "ListLogs should not return error when exceeding MaxPageSize")
		assert.Equal(t, common.MaxPageSize, len(logs), "Should limit page size to MaxPageSize")
	})

	t.Run("EmptyJobName", func(t *testing.T) {
		logs, total, err := logMgr.ListLogs("", 1, 10)
		require.NoError(t, err, "ListLogs should not return error with empty job name")
		assert.Equal(t, int64(25), total, "Total count should match all logs")
		assert.Equal(t, 10, len(logs), "Should return logs for all jobs")
	})

	t.Run("NonExistentJob", func(t *testing.T) {
		logs, total, err := logMgr.ListLogs("non-existent-job", 1, 10)
		require.NoError(t, err, "ListLogs should not return error for non-existent job")
		assert.Equal(t, int64(0), total, "Total count should be 0 for non-existent job")
		assert.Equal(t, 0, len(logs), "Should return empty logs array")
	})
}

func TestGetJobLog(t *testing.T) {
	logMgr, mongoClient, cleanup := setupTestEnv(t)
	defer cleanup()

	jobName := "test-job"
	insertTestLogs(t, mongoClient, 5, jobName)

	t.Run("ExistingJob", func(t *testing.T) {
		log, err := logMgr.GetJobLog(jobName)
		require.NoError(t, err, "GetJobLog should not return error for existing job")
		assert.Equal(t, jobName, log.JobName, "Job name should match")
		assert.NotEmpty(t, log.Command, "Command should not be empty")
	})

	t.Run("NonExistentJob", func(t *testing.T) {
		_, err := logMgr.GetJobLog("non-existent-job")
		assert.Equal(t, common.ErrJobNotFound, err, "GetJobLog should return ErrJobNotFound")
	})
}

func TestCleanExpiredLogs(t *testing.T) {
	logMgr, mongoClient, cleanup := setupTestEnv(t)
	defer cleanup()

	// 插入一些测试日志，日期跨越一个月
	now := time.Now()

	// 创建今天的日志
	recentLogs := []interface{}{
		&common.JobLog{
			JobName:   "recent-job",
			Command:   "echo recent",
			StartTime: now.Unix(),
			EndTime:   now.Unix() + 5,
		},
	}

	// 创建31天前的日志
	oldTime := now.AddDate(0, 0, -31)
	oldLogs := []interface{}{
		&common.JobLog{
			JobName:   "old-job",
			Command:   "echo old",
			StartTime: oldTime.Unix(),
			EndTime:   oldTime.Unix() + 5,
		},
	}

	// 插入日志
	_, err := mongoClient.InsertMany(recentLogs)
	require.NoError(t, err, "Failed to insert recent logs")
	_, err = mongoClient.InsertMany(oldLogs)
	require.NoError(t, err, "Failed to insert old logs")

	// 运行清理，保留30天内的日志
	err = logMgr.CleanExpiredLogs(30)
	require.NoError(t, err, "CleanExpiredLogs should not return error")

	// 验证只有最近的日志还存在
	count, err := mongoClient.CountJobLogs("")
	require.NoError(t, err, "CountJobLogs should not return error")
	assert.Equal(t, int64(1), count, "Only recent logs should remain")

	// 验证存在的是最近的日志
	logs, err := mongoClient.FindJobLogs("recent-job", 0, 10)
	require.NoError(t, err, "FindJobLogs should not return error")
	assert.Equal(t, 1, len(logs), "Should find the recent log")
	assert.Equal(t, "recent-job", logs[0].JobName, "Recent job should still exist")

	// 验证旧日志已被删除
	logs, err = mongoClient.FindJobLogs("old-job", 0, 10)
	require.NoError(t, err, "FindJobLogs should not return error")
	assert.Equal(t, 0, len(logs), "Old logs should be deleted")
}

func TestGetLogStatistics(t *testing.T) {
	logMgr, mongoClient, cleanup := setupTestEnv(t)
	defer cleanup()

	jobName := "test-job"
	insertTestLogs(t, mongoClient, 20, jobName)

	t.Run("DefaultPeriod", func(t *testing.T) {
		stats, err := logMgr.GetLogStatistics(jobName, 0)
		require.NoError(t, err, "GetLogStatistics should not return error with default period")

		assert.Contains(t, stats, "totalCount", "Stats should contain totalCount")
		assert.Contains(t, stats, "successCount", "Stats should contain successCount")
		assert.Contains(t, stats, "failCount", "Stats should contain failCount")
		assert.Contains(t, stats, "timeoutCount", "Stats should contain timeoutCount")
		assert.Contains(t, stats, "avgDuration", "Stats should contain avgDuration")
		assert.Contains(t, stats, "period", "Stats should contain period")

		assert.Equal(t, 7, stats["period"], "Default period should be 7 days")
	})

	t.Run("CustomPeriod", func(t *testing.T) {
		stats, err := logMgr.GetLogStatistics(jobName, 14)
		require.NoError(t, err, "GetLogStatistics should not return error with custom period")
		assert.Equal(t, 14, stats["period"], "Period should match specified value")
	})

	t.Run("NonExistentJob", func(t *testing.T) {
		stats, err := logMgr.GetLogStatistics("non-existent-job", 7)
		require.NoError(t, err, "GetLogStatistics should not return error for non-existent job")
		assert.Equal(t, 0, stats["totalCount"], "Total count should be 0")
		assert.Equal(t, 0, stats["successCount"], "Success count should be 0")
	})

	t.Run("SuccessAndFailureCounts", func(t *testing.T) {
		stats, err := logMgr.GetLogStatistics(jobName, 7)
		require.NoError(t, err, "GetLogStatistics should not return error")

		// 因为我们在insertTestLogs中设置了偶数索引成功，奇数索引失败
		successCount := stats["successCount"].(int)
		failCount := stats["failCount"].(int)

		// 验证成功和失败的数量总和等于总数
		assert.Equal(t, stats["totalCount"], successCount+failCount,
			"Success and failure counts should sum up to total count")
	})
}

func TestStartLogCleaner(t *testing.T) {
	logMgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// 创建具有较短清理间隔的测试用上下文
	testCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 启动日志清理器，设置为保留1天日志
	logMgr.StartLogCleaner(1)

	// 等待短暂时间让清理任务开始
	time.Sleep(50 * time.Millisecond)

	// 停止日志管理器应该不会导致panic
	logMgr.Stop()

	// 等待上下文取消
	<-testCtx.Done()
}

func TestLogManagerStop(t *testing.T) {
	logMgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	initialCtx := logMgr.ctx
	logMgr.Stop()

	select {
	case <-initialCtx.Done():
		assert.True(t, true, "Context should be canceled after Stop")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context was not canceled after Stop")
	}
}
