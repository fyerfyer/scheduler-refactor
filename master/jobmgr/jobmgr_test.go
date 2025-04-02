package jobmgr

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
)

func setupTestEnv(t *testing.T) (*JobManager, *etcd.Client, func()) {
	logger := zaptest.NewLogger(t)

	config.GlobalConfig = &config.Config{
		EtcdEndpoints:   []string{"localhost:2379"},
		EtcdDialTimeout: 5000,
	}

	etcdClient, err := etcd.NewClient()
	require.NoError(t, err, "Failed to create etcd client")

	jobMgr := NewJobManager(etcdClient, logger)
	require.NotNil(t, jobMgr, "JobManager should not be nil")

	cleanup := func() {
		jobMgr.Stop()
		etcdClient.Close()
	}

	return jobMgr, etcdClient, cleanup
}

func TestSaveJob(t *testing.T) {
	jobMgr, etcdClient, cleanup := setupTestEnv(t)
	defer cleanup()

	job := &common.Job{
		Name:      "test-job",
		Command:   "echo hello",
		CronExpr:  "*/5 * * * * *",
		Timeout:   10,
		Disabled:  false,
		CreatedAt: 0,
		UpdatedAt: 0,
	}

	err := jobMgr.SaveJob(job)
	require.NoError(t, err, "SaveJob should not return error")

	resp, err := etcdClient.Get(common.JobSaveDir + job.Name)
	require.NoError(t, err, "etcd Get should not return error")
	assert.Equal(t, int64(1), resp.Count, "Job should exist in etcd")

	assert.NotZero(t, job.CreatedAt, "CreatedAt should be set")
	assert.NotZero(t, job.UpdatedAt, "UpdatedAt should be set")
}

func TestGetJob(t *testing.T) {
	jobMgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	job := &common.Job{
		Name:     "test-get-job",
		Command:  "echo hello",
		CronExpr: "*/5 * * * * *",
		Timeout:  10,
		Disabled: false,
	}

	err := jobMgr.SaveJob(job)
	require.NoError(t, err, "SaveJob should not return error")

	fetchedJob, err := jobMgr.GetJob("test-get-job")
	require.NoError(t, err, "GetJob should not return error")
	assert.Equal(t, job.Name, fetchedJob.Name, "Job name should match")
	assert.Equal(t, job.Command, fetchedJob.Command, "Job command should match")
	assert.Equal(t, job.CronExpr, fetchedJob.CronExpr, "Job cron expression should match")

	_, err = jobMgr.GetJob("non-existent-job")
	assert.Equal(t, common.ErrJobNotFound, err, "Getting non-existent job should return ErrJobNotFound")
}

func TestListJobs(t *testing.T) {
	jobMgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	jobs := []*common.Job{
		{
			Name:     "test-list-job1",
			Command:  "echo hello1",
			CronExpr: "*/5 * * * * *",
			Timeout:  10,
		},
		{
			Name:     "test-list-job2",
			Command:  "echo hello2",
			CronExpr: "*/10 * * * * *",
			Timeout:  20,
		},
	}

	for _, job := range jobs {
		err := jobMgr.SaveJob(job)
		require.NoError(t, err, "SaveJob should not return error")
	}

	listedJobs, err := jobMgr.ListJobs()
	require.NoError(t, err, "ListJobs should not return error")

	foundJob1 := false
	foundJob2 := false

	for _, job := range listedJobs {
		if job.Name == "test-list-job1" {
			foundJob1 = true
		} else if job.Name == "test-list-job2" {
			foundJob2 = true
		}
	}

	assert.True(t, foundJob1, "test-list-job1 should be in the list")
	assert.True(t, foundJob2, "test-list-job2 should be in the list")
}

func TestDeleteJob(t *testing.T) {
	jobMgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	job := &common.Job{
		Name:     "test-delete-job",
		Command:  "echo hello",
		CronExpr: "*/5 * * * * *",
		Timeout:  10,
	}

	err := jobMgr.SaveJob(job)
	require.NoError(t, err, "SaveJob should not return error")

	err = jobMgr.DeleteJob("test-delete-job")
	require.NoError(t, err, "DeleteJob should not return error")

	_, err = jobMgr.GetJob("test-delete-job")
	assert.Equal(t, common.ErrJobNotFound, err, "Job should be deleted")

	err = jobMgr.DeleteJob("non-existent-job")
	assert.Equal(t, common.ErrJobNotFound, err, "Deleting non-existent job should return ErrJobNotFound")
}

func TestDisableEnableJob(t *testing.T) {
	jobMgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	job := &common.Job{
		Name:     "test-disable-job",
		Command:  "echo hello",
		CronExpr: "*/5 * * * * *",
		Timeout:  10,
		Disabled: false,
	}

	err := jobMgr.SaveJob(job)
	require.NoError(t, err, "SaveJob should not return error")

	err = jobMgr.DisableJob("test-disable-job")
	require.NoError(t, err, "DisableJob should not return error")

	fetchedJob, err := jobMgr.GetJob("test-disable-job")
	require.NoError(t, err, "GetJob should not return error")
	assert.True(t, fetchedJob.Disabled, "Job should be disabled")

	err = jobMgr.EnableJob("test-disable-job")
	require.NoError(t, err, "EnableJob should not return error")

	fetchedJob, err = jobMgr.GetJob("test-disable-job")
	require.NoError(t, err, "GetJob should not return error")
	assert.False(t, fetchedJob.Disabled, "Job should be enabled")
}

func TestKillJob(t *testing.T) {
	jobMgr, etcdClient, cleanup := setupTestEnv(t)
	defer cleanup()

	job := &common.Job{
		Name:     "test-kill-job",
		Command:  "echo hello",
		CronExpr: "*/5 * * * * *",
		Timeout:  10,
	}

	err := jobMgr.SaveJob(job)
	require.NoError(t, err, "SaveJob should not return error")

	err = jobMgr.KillJob("test-kill-job")
	require.NoError(t, err, "KillJob should not return error")

	resp, err := etcdClient.Get(common.JobLockDir + "test-kill-job")
	require.NoError(t, err, "etcd Get should not return error")
	assert.Equal(t, int64(1), resp.Count, "Kill marker should exist in etcd")

	time.Sleep(6 * time.Second)

	resp, err = etcdClient.Get(common.JobLockDir + "test-kill-job")
	require.NoError(t, err, "etcd Get should not return error")
	assert.Equal(t, int64(0), resp.Count, "Kill marker should be expired after TTL")
}

func TestSearchJobs(t *testing.T) {
	jobMgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	jobs := []*common.Job{
		{
			Name:     "apple-job",
			Command:  "echo apple",
			CronExpr: "*/5 * * * * *",
			Timeout:  10,
		},
		{
			Name:     "banana-task",
			Command:  "echo banana",
			CronExpr: "*/10 * * * * *",
			Timeout:  20,
		},
		{
			Name:     "cherry-service",
			Command:  "grep apple file.txt",
			CronExpr: "*/15 * * * * *",
			Timeout:  30,
		},
	}

	for _, job := range jobs {
		err := jobMgr.SaveJob(job)
		require.NoError(t, err, "SaveJob should not return error")
	}

	t.Run("SearchByName", func(t *testing.T) {
		results, err := jobMgr.SearchJobs("apple")
		require.NoError(t, err, "SearchJobs should not return error")
		assert.Equal(t, 2, len(results), "Should find 2 jobs with 'apple'")
	})

	t.Run("SearchByCommand", func(t *testing.T) {
		results, err := jobMgr.SearchJobs("banana")
		require.NoError(t, err, "SearchJobs should not return error")
		assert.Equal(t, 1, len(results), "Should find 1 job with 'banana'")
	})

	t.Run("EmptyKeyword", func(t *testing.T) {
		results, err := jobMgr.SearchJobs("")
		require.NoError(t, err, "SearchJobs should not return error")
		assert.GreaterOrEqual(t, len(results), 3, "Should return all jobs")
	})

	t.Run("NoMatch", func(t *testing.T) {
		results, err := jobMgr.SearchJobs("nonexistent")
		require.NoError(t, err, "SearchJobs should not return error")
		assert.Equal(t, 0, len(results), "Should find no jobs")
	})
}

func TestStop(t *testing.T) {
	jobMgr, _, cleanup := setupTestEnv(t)
	defer cleanup()

	initialCtx := jobMgr.ctx
	jobMgr.Stop()

	select {
	case <-initialCtx.Done():
		assert.True(t, true, "Context should be canceled")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be canceled after Stop")
	}
}

func TestContainsString(t *testing.T) {
	testCases := []struct {
		source   string
		substr   string
		expected bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "universe", false},
		{"", "test", false},
		{"test", "", true}, // Empty substring should match
		{"abc", "abcd", false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s contains %s", tc.source, tc.substr), func(t *testing.T) {
			result := containsString(tc.source, tc.substr)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJobManagerWithContext(t *testing.T) {
	logger := zaptest.NewLogger(t)

	config.GlobalConfig = &config.Config{
		EtcdEndpoints:   []string{"localhost:2379"},
		EtcdDialTimeout: 5000,
	}

	etcdClient, err := etcd.NewClient()
	require.NoError(t, err, "Failed to create etcd client")
	defer etcdClient.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobMgr := &JobManager{
		etcdClient: etcdClient,
		logger:     logger,
		ctx:        ctx,
		cancelFunc: cancel,
	}

	job := &common.Job{
		Name:     "context-test-job",
		Command:  "echo hello",
		CronExpr: "*/5 * * * * *",
	}

	err = jobMgr.SaveJob(job)
	require.NoError(t, err, "SaveJob should not return error")

	cancel() // Cancel the context

	err = jobMgr.SaveJob(job)
	require.NoError(t, err, "SaveJob should still work after context cancel")
}

func TestToLower(t *testing.T) {
	testCases := []struct {
		input    byte
		expected byte
	}{
		{'A', 'a'},
		{'Z', 'z'},
		{'a', 'a'},
		{'z', 'z'},
		{'0', '0'},
		{'.', '.'},
		{' ', ' '},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("toLower(%c)", tc.input), func(t *testing.T) {
			result := toLower(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
