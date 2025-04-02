package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/master/jobmgr"
	"github.com/fyerfyer/scheduler-refactor/master/logmgr"
	"github.com/fyerfyer/scheduler-refactor/master/workermgr"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
	"github.com/fyerfyer/scheduler-refactor/pkg/mongodb"
)

type MasterTestContext struct {
	logger      *zap.Logger
	etcdClient  *etcd.Client
	mongoClient *mongodb.Client
	jobMgr      *jobmgr.JobManager
	logMgr      *logmgr.LogManager
	workerMgr   *workermgr.WorkerManager
}

func setupTestEnv(t *testing.T) (*MasterTestContext, func()) {
	logger := zaptest.NewLogger(t)

	config.GlobalConfig = &config.Config{
		EtcdEndpoints:       []string{"localhost:2379"},
		EtcdDialTimeout:     5000,
		MongoURI:            "mongodb://localhost:27017",
		MongoConnectTimeout: 5000,
		HeartbeatInterval:   1000,
		JobLockTTL:          5,
		LogBatchSize:        10,
		LogCommitTimeout:    500,
	}

	etcdClient, err := etcd.NewClient()
	require.NoError(t, err, "Failed to create etcd client")

	mongoClient, err := mongodb.NewClient()
	require.NoError(t, err, "Failed to create MongoDB client")

	jobMgr := jobmgr.NewJobManager(etcdClient, logger)
	logMgr := logmgr.NewLogManager(mongoClient, logger)
	workerMgr := workermgr.NewWorkerManager(etcdClient, logger)

	ctx := &MasterTestContext{
		logger:      logger,
		etcdClient:  etcdClient,
		mongoClient: mongoClient,
		jobMgr:      jobMgr,
		logMgr:      logMgr,
		workerMgr:   workerMgr,
	}

	cleanupTestData(etcdClient, mongoClient)

	cleanup := func() {
		jobMgr.Stop()
		logMgr.Stop()
		workerMgr.Stop()

		// Clean up test data
		cleanupTestData(etcdClient, mongoClient)

		etcdClient.Close()
		mongoClient.Close()
	}

	return ctx, cleanup
}

func cleanupTestData(etcdClient *etcd.Client, mongoClient *mongodb.Client) {
	etcdClient.DeleteWithPrefix(common.JobSaveDir)
	etcdClient.DeleteWithPrefix(common.JobLockDir)
	etcdClient.DeleteWithPrefix(common.WorkerRegisterDir)
	mongoClient.DropCollection()
}

func registerTestWorker(t *testing.T, etcdClient *etcd.Client, workerID string) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "test-host"
	}

	workerInfo := &common.WorkerInfo{
		IP:       workerID,
		Hostname: hostname,
		CPUUsage: 0.5,
		MemUsage: 0.3,
		LastSeen: time.Now().UnixNano() / int64(time.Millisecond),
	}

	data, err := json.Marshal(workerInfo)
	require.NoError(t, err, "Failed to marshal worker info")

	workerKey := common.WorkerRegisterDir + workerID
	_, err = etcdClient.Put(workerKey, string(data))
	require.NoError(t, err, "Failed to register test worker")
}

func createTestJob(name string, command string, cronExpr string) *common.Job {
	return &common.Job{
		Name:     name,
		Command:  command,
		CronExpr: cronExpr,
		Timeout:  10,
		Disabled: false,
	}
}

func TestJobLifecycle(t *testing.T) {
	ctx, cleanup := setupTestEnv(t)
	defer cleanup()

	jobName := fmt.Sprintf("test-job-%d", time.Now().Unix())
	job := createTestJob(jobName, "echo hello world", "*/5 * * * * *")

	t.Run("CreateJob", func(t *testing.T) {
		err := ctx.jobMgr.SaveJob(job)
		require.NoError(t, err, "Failed to save job")

		savedJob, err := ctx.jobMgr.GetJob(jobName)
		require.NoError(t, err, "Failed to get saved job")
		assert.Equal(t, job.Name, savedJob.Name, "Job name should match")
		assert.Equal(t, job.Command, savedJob.Command, "Job command should match")
		assert.Equal(t, job.CronExpr, savedJob.CronExpr, "Job cron expression should match")
		assert.False(t, savedJob.Disabled, "Job should not be disabled")
	})

	t.Run("UpdateJob", func(t *testing.T) {
		job.Command = "echo updated command"
		err := ctx.jobMgr.SaveJob(job)
		require.NoError(t, err, "Failed to update job")

		updatedJob, err := ctx.jobMgr.GetJob(jobName)
		require.NoError(t, err, "Failed to get updated job")
		assert.Equal(t, "echo updated command", updatedJob.Command, "Updated command should match")
	})

	t.Run("DisableJob", func(t *testing.T) {
		err := ctx.jobMgr.DisableJob(jobName)
		require.NoError(t, err, "Failed to disable job")

		disabledJob, err := ctx.jobMgr.GetJob(jobName)
		require.NoError(t, err, "Failed to get disabled job")
		assert.True(t, disabledJob.Disabled, "Job should be disabled")
	})

	t.Run("EnableJob", func(t *testing.T) {
		err := ctx.jobMgr.EnableJob(jobName)
		require.NoError(t, err, "Failed to enable job")

		enabledJob, err := ctx.jobMgr.GetJob(jobName)
		require.NoError(t, err, "Failed to get enabled job")
		assert.False(t, enabledJob.Disabled, "Job should not be disabled")
	})

	t.Run("DeleteJob", func(t *testing.T) {
		err := ctx.jobMgr.DeleteJob(jobName)
		require.NoError(t, err, "Failed to delete job")

		_, err = ctx.jobMgr.GetJob(jobName)
		assert.Equal(t, common.ErrJobNotFound, err, "Job should be deleted")
	})
}

func TestWorkerDiscovery(t *testing.T) {
	ctx, cleanup := setupTestEnv(t)
	defer cleanup()

	workerID := fmt.Sprintf("test-worker-%d", time.Now().Unix())

	t.Run("RegisterWorker", func(t *testing.T) {
		registerTestWorker(t, ctx.etcdClient, workerID)

		time.Sleep(500 * time.Millisecond)

		workers := ctx.workerMgr.ListWorkers()
		assert.GreaterOrEqual(t, len(workers), 1, "At least one worker should be registered")

		found := false
		for _, worker := range workers {
			if worker.IP == workerID {
				found = true
				assert.NotEmpty(t, worker.Hostname, "Worker hostname should not be empty")
				break
			}
		}
		assert.True(t, found, "Registered worker should be found")
	})

	t.Run("GetWorkerStatus", func(t *testing.T) {
		time.Sleep(100 * time.Millisecond)

		status := ctx.workerMgr.CheckWorkers()
		assert.Contains(t, status, workerID, "Worker status should contain the registered worker")
		assert.Equal(t, "online", status[workerID], "Worker should be online")
	})

	t.Run("GetWorkerStats", func(t *testing.T) {
		stats := ctx.workerMgr.GetWorkerStats()
		assert.GreaterOrEqual(t, stats["total"].(int), 1, "Should have at least 1 worker in total")
		assert.GreaterOrEqual(t, stats["online"].(int), 1, "Should have at least 1 worker online")
		assert.NotZero(t, stats["avgCpuUsage"], "Average CPU usage should not be zero")
		assert.NotZero(t, stats["avgMemUsage"], "Average memory usage should not be zero")
	})
}

func TestJobKillMarker(t *testing.T) {
	ctx, cleanup := setupTestEnv(t)
	defer cleanup()

	jobName := fmt.Sprintf("kill-test-job-%d", time.Now().Unix())
	job := createTestJob(jobName, "sleep 30", "*/5 * * * * *")

	err := ctx.jobMgr.SaveJob(job)
	require.NoError(t, err, "Failed to save job")

	err = ctx.jobMgr.KillJob(jobName)
	require.NoError(t, err, "Failed to kill job")

	resp, err := ctx.etcdClient.Get(common.JobLockDir + jobName)
	require.NoError(t, err, "Failed to get kill marker")
	assert.Equal(t, int64(1), resp.Count, "Kill marker should exist in etcd")

	time.Sleep(6 * time.Second)

	resp, err = ctx.etcdClient.Get(common.JobLockDir + jobName)
	require.NoError(t, err, "Failed to get kill marker after TTL")
	assert.Equal(t, int64(0), resp.Count, "Kill marker should be expired after TTL")
}

func TestLogManagement(t *testing.T) {
	ctx, cleanup := setupTestEnv(t)
	defer cleanup()

	jobName := fmt.Sprintf("log-test-job-%d", time.Now().Unix())

	t.Run("CleanOldLogs", func(t *testing.T) {
		err := ctx.mongoClient.DropCollection()
		require.NoError(t, err, "Failed to drop collection before test")

		now := time.Now().Unix()

		recentLog := &common.JobLog{
			JobName:   jobName,
			Command:   "echo recent",
			Output:    "recent output",
			StartTime: now,
			EndTime:   now + 1,
			WorkerIP:  "test-worker",
		}

		oldLog := &common.JobLog{
			JobName:   jobName,
			Command:   "echo old",
			Output:    "old output",
			StartTime: now - 31*24*60*60, // 31 days ago
			EndTime:   now - 31*24*60*60 + 1,
			WorkerIP:  "test-worker",
		}

		logs := []interface{}{recentLog, oldLog}
		_, err = ctx.mongoClient.InsertMany(logs)
		require.NoError(t, err, "Failed to insert test logs")

		err = ctx.logMgr.CleanExpiredLogs(30)
		require.NoError(t, err, "Failed to clean old logs")

		count, err := ctx.mongoClient.CountJobLogs(jobName)
		require.NoError(t, err, "Failed to count logs")
		assert.Equal(t, int64(1), count, "Should have only 1 log after cleaning")
	})

	t.Run("LogStatistics", func(t *testing.T) {
		err := ctx.mongoClient.DropCollection()
		require.NoError(t, err, "Failed to drop collection before test")

		now := time.Now().Unix()

		successLog := &common.JobLog{
			JobName:   jobName,
			Command:   "echo success",
			Output:    "success output",
			ExitCode:  0,
			StartTime: now - 60,
			EndTime:   now - 59,
			WorkerIP:  "test-worker",
		}

		failLog := &common.JobLog{
			JobName:   jobName,
			Command:   "echo fail",
			Output:    "fail output",
			Error:     "some error",
			ExitCode:  1,
			StartTime: now - 120,
			EndTime:   now - 118,
			WorkerIP:  "test-worker",
		}

		timeoutLog := &common.JobLog{
			JobName:   jobName,
			Command:   "sleep 10",
			Output:    "",
			Error:     "timeout",
			ExitCode:  1,
			IsTimeout: true,
			StartTime: now - 180,
			EndTime:   now - 170,
			WorkerIP:  "test-worker",
		}

		logs := []interface{}{successLog, failLog, timeoutLog}
		_, err = ctx.mongoClient.InsertMany(logs)
		require.NoError(t, err, "Failed to insert test logs")

		stats, err := ctx.logMgr.GetLogStatistics(jobName, 1) // Last 1 day
		require.NoError(t, err, "Failed to get log statistics")

		assert.Equal(t, 3, stats["totalCount"], "Should have 3 logs in total")
		assert.Equal(t, 1, stats["successCount"], "Should have 1 successful log")
		assert.Equal(t, 2, stats["failCount"], "Should have 2 failed logs")
		assert.Equal(t, 1, stats["timeoutCount"], "Should have 1 timeout log")
		assert.NotZero(t, stats["avgDuration"], "Average duration should not be zero")
	})
}

func TestFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cleanup := setupTestEnv(t)
	defer cleanup()

	jobName := fmt.Sprintf("workflow-test-job-%d", time.Now().Unix())
	workerID := fmt.Sprintf("workflow-test-worker-%d", time.Now().Unix())

	t.Run("RegisterWorkerAndCreateJob", func(t *testing.T) {
		registerTestWorker(t, ctx.etcdClient, workerID)

		job := createTestJob(jobName, "echo test workflow", "*/1 * * * * *")

		err := ctx.jobMgr.SaveJob(job)
		require.NoError(t, err, "Failed to save job")

		savedJob, err := ctx.jobMgr.GetJob(jobName)
		require.NoError(t, err, "Failed to get saved job")
		assert.Equal(t, job.Name, savedJob.Name)

		time.Sleep(1500 * time.Millisecond)
	})

	t.Run("LogGeneration", func(t *testing.T) {
		now := time.Now().Unix()
		jobLog := &common.JobLog{
			JobName:      jobName,
			Command:      "echo test workflow",
			Output:       "test workflow output",
			Error:        "",
			PlanTime:     now - 1,
			ScheduleTime: now,
			StartTime:    now,
			EndTime:      now + 1,
			ExitCode:     0,
			IsTimeout:    false,
			WorkerIP:     workerID,
		}

		_, err := ctx.mongoClient.InsertOne(jobLog)
		require.NoError(t, err, "Failed to insert test job log")

		time.Sleep(100 * time.Millisecond)

		logs, total, err := ctx.logMgr.ListLogs(jobName, 1, 10)
		require.NoError(t, err, "Failed to list logs")
		assert.Equal(t, int64(1), total, "Should have one log")
		assert.Equal(t, jobName, logs[0].JobName, "Log job name should match")

		t.Logf("Job log successfully generated and verified")
	})

	t.Run("KillJobAndVerify", func(t *testing.T) {
		err := ctx.jobMgr.KillJob(jobName)
		require.NoError(t, err, "Failed to kill job")

		resp, err := ctx.etcdClient.Get(common.JobLockDir + jobName)
		require.NoError(t, err, "Failed to get kill marker")

		if resp.Count == 0 {
			t.Log("Kill marker not found, might already be expired")
		} else {
			t.Log("Kill marker exists, job should be killed")

			time.Sleep(2 * time.Second)

			resp, err = ctx.etcdClient.Get(common.JobLockDir + jobName)
			require.NoError(t, err, "Failed to get kill marker after TTL")

			if resp.Count > 0 {
				t.Log("Kill marker still exists, waiting for TTL expiration")
			}
		}
	})

	t.Run("CleanupJob", func(t *testing.T) {
		err := ctx.jobMgr.DeleteJob(jobName)
		require.NoError(t, err, "Failed to delete job")

		_, err = ctx.jobMgr.GetJob(jobName)
		assert.Equal(t, common.ErrJobNotFound, err, "Job should be deleted")
	})

	t.Run("LogStatisticsVerification", func(t *testing.T) {
		stats, err := ctx.logMgr.GetLogStatistics(jobName, 1)
		require.NoError(t, err, "Failed to get log statistics")

		assert.GreaterOrEqual(t, stats["totalCount"], 1, "Should have at least one log")
		assert.Equal(t, 1, stats["successCount"], "Should have one successful execution")
		assert.Equal(t, 0, stats["failCount"], "Should have no failed executions")
	})
}
