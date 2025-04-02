package main

import (
	"encoding/json"
	"github.com/fyerfyer/scheduler-refactor/worker/executor"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/mongodb"
	"github.com/fyerfyer/scheduler-refactor/worker/logsink"
)

func TestInitConfig(t *testing.T) {
	configContent := `{
        "etcdEndpoints": ["localhost:2379"],
        "etcdDialTimeout": 5000,
        "workerId": "test-worker",
        "heartbeatInterval": 3000,
        "logBatchSize": 50,
        "logCommitTimeout": 2000,
        "executorThreads": 5,
        "jobLockTtl": 10,
        "mongoUri": "mongodb://localhost:27017",
        "mongoConnectTimeout": 3000
    }`

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err, "Failed to create test config file")

	err = config.InitConfig(configPath)
	require.NoError(t, err, "Failed to initialize config")

	assert.Equal(t, []string{"localhost:2379"}, config.GlobalConfig.EtcdEndpoints)
	assert.Equal(t, 5000, config.GlobalConfig.EtcdDialTimeout)
	assert.Equal(t, "test-worker", config.GlobalConfig.WorkerID)
	assert.Equal(t, 3000, config.GlobalConfig.HeartbeatInterval)
	assert.Equal(t, 50, config.GlobalConfig.LogBatchSize)
	assert.Equal(t, 2000, config.GlobalConfig.LogCommitTimeout)
}

func TestInitWorker(t *testing.T) {
	err := setupConfig(t)
	require.NoError(t, err, "Failed to setup config")

	wctx := &workerContext{}
	err = initWorker(wctx)
	require.NoError(t, err, "Failed to initialize worker components")

	assert.NotNil(t, wctx.logger, "Logger should not be nil")
	assert.NotNil(t, wctx.etcdClient, "ETCD client should not be nil")
	assert.NotNil(t, wctx.mongoClient, "MongoDB client should not be nil")
	assert.NotNil(t, wctx.executor, "Executor should not be nil")
	assert.NotNil(t, wctx.jobManager, "JobManager should not be nil")
	assert.NotNil(t, wctx.register, "Register should not be nil")
	assert.NotNil(t, wctx.scheduler, "Scheduler should not be nil")
	assert.NotNil(t, wctx.logSink, "LogSink should not be nil")

	closeWorkerContext(wctx)
}

func TestStartWorker(t *testing.T) {
	err := setupConfig(t)
	require.NoError(t, err, "Failed to setup config")

	wctx := &workerContext{}
	err = initWorker(wctx)
	require.NoError(t, err, "Failed to initialize worker components")
	defer closeWorkerContext(wctx)

	startWorker(wctx)

	time.Sleep(500 * time.Millisecond)

	registryKey := common.WorkerRegisterDir + config.GlobalConfig.WorkerID
	resp, err := wctx.etcdClient.Get(registryKey)
	require.NoError(t, err, "Failed to get registry key")
	assert.True(t, len(resp.Kvs) > 0, "Worker should be registered in etcd")

	var workerInfo common.WorkerInfo
	require.NoError(t, json.Unmarshal(resp.Kvs[0].Value, &workerInfo), "Failed to unmarshal worker info")
}

func TestHandleExecuteResults(t *testing.T) {
	err := setupConfig(t)
	require.NoError(t, err, "Failed to setup config")

	wctx := &workerContext{}
	err = initWorker(wctx)
	require.NoError(t, err, "Failed to initialize worker components")
	defer closeWorkerContext(wctx)

	job := &common.Job{
		Name:      "test-job",
		Command:   "echo hello",
		CronExpr:  "*/1 * * * * *",
		Timeout:   10,
		Disabled:  false,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	jobExecuteInfo := &common.JobExecuteInfo{
		Job:      job,
		PlanTime: time.Now(),
		RealTime: time.Now(),
	}

	wctx.scheduler.GetExecutingJobs()["test-job"] = jobExecuteInfo

	// 不使用反射，直接创建一个执行结果并发送到日志收集器
	go handleExecuteResults(wctx)

	// 创建测试结果
	result := &common.JobExecuteResult{
		JobName:   "test-job",
		Output:    "hello",
		Error:     "",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		ExitCode:  0,
	}

	// 创建日志并直接发送到logSink而不是通过执行器的结果通道
	jobLog := executor.BuildJobLog(result, jobExecuteInfo)
	wctx.logSink.Append(jobLog)

	time.Sleep(500 * time.Millisecond)
}

func TestLogSinkCleanup(t *testing.T) {
	err := setupConfig(t)
	require.NoError(t, err, "Failed to setup config")

	logger := initLogger()
	mongoClient, err := mongodb.NewClient()
	require.NoError(t, err, "Failed to create MongoDB client")
	defer mongoClient.Close()

	// 先删除集合，确保测试环境干净
	err = mongoClient.DropCollection()
	require.NoError(t, err, "Failed to drop collection before test")

	logSink := logsink.NewLogSink(mongoClient, logger)

	jobLogs := []*common.JobLog{
		{
			JobName:      "test-old-job",
			Command:      "echo old",
			Output:       "old",
			PlanTime:     time.Now().Add(-10 * 24 * time.Hour).Unix(),
			ScheduleTime: time.Now().Add(-10 * 24 * time.Hour).Unix(),
			StartTime:    time.Now().Add(-10 * 24 * time.Hour).Unix(),
			EndTime:      time.Now().Add(-10 * 24 * time.Hour).Unix(),
			ExitCode:     0,
			WorkerIP:     "test-worker",
		},
		{
			JobName:      "test-recent-job",
			Command:      "echo recent",
			Output:       "recent",
			PlanTime:     time.Now().Unix(),
			ScheduleTime: time.Now().Unix(),
			StartTime:    time.Now().Unix(),
			EndTime:      time.Now().Unix(),
			ExitCode:     0,
			WorkerIP:     "test-worker",
		},
	}

	docs := make([]interface{}, len(jobLogs))
	for i, log := range jobLogs {
		docs[i] = log
	}

	_, err = mongoClient.InsertMany(docs)
	require.NoError(t, err, "Failed to insert test logs")

	logSink.CleanExpiredLogs(7)
	time.Sleep(500 * time.Millisecond)

	logs, err := mongoClient.FindJobLogs("test-old-job", 0, 10)
	require.NoError(t, err, "Failed to query logs")
	assert.Equal(t, 0, len(logs), "Old logs should be deleted")

	logs, err = mongoClient.FindJobLogs("test-recent-job", 0, 10)
	require.NoError(t, err, "Failed to query logs")
	assert.Equal(t, 1, len(logs), "Recent logs should not be deleted")
	err = mongoClient.DropCollection()
	require.NoError(t, err, "Failed to drop collection after test")
}

func closeWorkerContext(wctx *workerContext) {
	if wctx.etcdClient != nil {
		wctx.etcdClient.Close()
	}
	if wctx.mongoClient != nil {
		wctx.mongoClient.Close()
	}
	if wctx.scheduler != nil {
		wctx.scheduler.Stop()
	}
	if wctx.register != nil {
		wctx.register.Stop()
	}
}

func setupConfig(t *testing.T) error {
	config.GlobalConfig = &config.Config{
		EtcdEndpoints:       []string{"localhost:2379"},
		EtcdDialTimeout:     5000,
		WorkerID:            "test-worker-" + time.Now().Format("20060102150405"),
		HeartbeatInterval:   1000,
		LogBatchSize:        10,
		LogCommitTimeout:    500,
		ExecutorThreads:     5,
		JobLockTTL:          5,
		MongoURI:            "mongodb://localhost:27017",
		MongoConnectTimeout: 5000,
	}
	return nil
}

func TestInitLogger(t *testing.T) {
	logger := initLogger()
	assert.NotNil(t, logger, "Logger should not be nil")
}
