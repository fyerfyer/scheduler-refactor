package jobmgr

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
)

func setupTest(t *testing.T) (*etcd.Client, *zap.Logger) {
	if config.GlobalConfig == nil {
		config.GlobalConfig = &config.Config{
			EtcdEndpoints:   []string{"localhost:2379"},
			EtcdDialTimeout: 5000,
		}
	}

	client, err := etcd.NewClient()
	require.NoError(t, err, "Failed to create etcd client")
	logger, _ := zap.NewDevelopment()

	return client, logger
}

func cleanupJob(t *testing.T, client *etcd.Client, jobName string) {
	jobKey := common.JobSaveDir + jobName
	_, err := client.Delete(jobKey)
	if err != nil {
		t.Logf("Warning: cleanup job failed: %v", err)
	}
}

func createTestJob(t *testing.T, client *etcd.Client, job *common.Job) {
	jobKey := common.JobSaveDir + job.Name
	jobData, err := json.Marshal(job)
	require.NoError(t, err, "Failed to marshal job")

	_, err = client.Put(jobKey, string(jobData))
	require.NoError(t, err, "Failed to put job in etcd")
}

func TestJobManager_LoadJobs(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	cleanupJob(t, client, "test_job1")
	cleanupJob(t, client, "test_job2")

	job1 := &common.Job{
		Name:      "test_job1",
		Command:   "echo hello",
		CronExpr:  "*/5 * * * * *",
		Timeout:   10,
		Disabled:  false,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	job2 := &common.Job{
		Name:      "test_job2",
		Command:   "echo world",
		CronExpr:  "*/10 * * * * *",
		Timeout:   20,
		Disabled:  true,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	createTestJob(t, client, job1)
	createTestJob(t, client, job2)

	jobMgr := NewJobManager(client, logger)
	defer jobMgr.Stop()

	loadedJob1, exists1 := jobMgr.GetJob("test_job1")
	assert.True(t, exists1, "Job1 should exist in cache")
	assert.Equal(t, job1.Command, loadedJob1.Command, "Job1 command should match")

	loadedJob2, exists2 := jobMgr.GetJob("test_job2")
	assert.True(t, exists2, "Job2 should exist in cache")
	assert.Equal(t, job2.Command, loadedJob2.Command, "Job2 command should match")
}

func TestJobManager_WatchJobs(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	jobName := "test_watch_job"
	cleanupJob(t, client, jobName)

	jobMgr := NewJobManager(client, logger)
	defer jobMgr.Stop()

	eventChan := jobMgr.GetEventChan()

	job := &common.Job{
		Name:      jobName,
		Command:   "echo hello",
		CronExpr:  "*/5 * * * * *",
		Timeout:   10,
		Disabled:  false,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	createTestJob(t, client, job)

	var saveEvent *common.JobEvent
	select {
	case event := <-eventChan:
		saveEvent = event
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for save event")
	}

	assert.Equal(t, common.JobEventSave, saveEvent.EventType, "Event type should be JobEventSave")
	assert.Equal(t, jobName, saveEvent.Job.Name, "Job name should match")

	cleanupJob(t, client, jobName)

	var deleteEvent *common.JobEvent
	select {
	case event := <-eventChan:
		deleteEvent = event
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for delete event")
	}

	assert.Equal(t, common.JobEventDelete, deleteEvent.EventType, "Event type should be JobEventDelete")
	assert.Equal(t, jobName, deleteEvent.Job.Name, "Job name should match")
}

func TestJobManager_ListJobs(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	cleanupJob(t, client, "test_list_job1")
	cleanupJob(t, client, "test_list_job2")

	job1 := &common.Job{
		Name:      "test_list_job1",
		Command:   "echo hello",
		CronExpr:  "*/5 * * * * *",
		Timeout:   10,
		Disabled:  false,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	job2 := &common.Job{
		Name:      "test_list_job2",
		Command:   "echo world",
		CronExpr:  "*/10 * * * * *",
		Timeout:   20,
		Disabled:  true,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	createTestJob(t, client, job1)
	createTestJob(t, client, job2)

	jobMgr := NewJobManager(client, logger)
	defer jobMgr.Stop()

	jobs := jobMgr.ListJobs()

	assert.GreaterOrEqual(t, len(jobs), 2, "Should have at least 2 jobs")

	var foundJob1, foundJob2 bool
	for _, job := range jobs {
		if job.Name == "test_list_job1" {
			foundJob1 = true
		}
		if job.Name == "test_list_job2" {
			foundJob2 = true
		}
	}

	assert.True(t, foundJob1, "test_list_job1 should be in the list")
	assert.True(t, foundJob2, "test_list_job2 should be in the list")
}

func TestJobManager_GetJob(t *testing.T) {
	client, logger := setupTest(t)
	defer client.Close()

	jobName := "test_get_job"
	cleanupJob(t, client, jobName)

	job := &common.Job{
		Name:      jobName,
		Command:   "echo hello",
		CronExpr:  "*/5 * * * * *",
		Timeout:   10,
		Disabled:  false,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	createTestJob(t, client, job)

	jobMgr := NewJobManager(client, logger)
	defer jobMgr.Stop()

	loadedJob, exists := jobMgr.GetJob(jobName)
	assert.True(t, exists, "Job should exist")
	assert.NotNil(t, loadedJob, "Loaded job should not be nil")
	assert.Equal(t, job.Command, loadedJob.Command, "Job command should match")

	nonExistJob, exists := jobMgr.GetJob("non_exist_job")
	assert.False(t, exists, "Non-existent job should not exist")
	assert.Nil(t, nonExistJob, "Non-existent job should be nil")
}
