package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/master/api"
	"github.com/fyerfyer/scheduler-refactor/master/jobmgr"
	"github.com/fyerfyer/scheduler-refactor/master/logmgr"
	"github.com/fyerfyer/scheduler-refactor/master/workermgr"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
	"github.com/fyerfyer/scheduler-refactor/pkg/mongodb"
)

var baseURL string

func setupIntegrationTest(t *testing.T) (*http.Server, func()) {
	apiPort := 18080
	config.GlobalConfig = &config.Config{
		EtcdEndpoints:       []string{"localhost:2379"},
		EtcdDialTimeout:     5000,
		ApiPort:             apiPort,
		MongoURI:            "mongodb://localhost:27017",
		MongoConnectTimeout: 5000,
	}

	baseURL = fmt.Sprintf("http://localhost:%d/api/v1", apiPort)
	logger, _ := zap.NewDevelopment()
	etcdClient, err := etcd.NewClient()
	require.NoError(t, err, "Failed to connect to etcd")
	mongoClient, err := mongodb.NewClient()
	require.NoError(t, err, "Failed to connect to MongoDB")

	jobManager := jobmgr.NewJobManager(etcdClient, logger)
	logManager := logmgr.NewLogManager(mongoClient, logger)
	workerManager := workermgr.NewWorkerManager(etcdClient, logger)

	apiServer := api.NewServer(logger, jobManager, logManager, workerManager)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", apiPort),
		Handler: apiServer.GetHTTPEngine(),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Failed to start API server: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	cleanup := func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        
        if err := server.Shutdown(ctx); err != nil {
            t.Logf("Failed to shutdown server: %v", err)
        }

        // 清除etcd中的数据
        if _, err := etcdClient.DeleteWithPrefix("/cron/jobs/"); err != nil {
            t.Logf("Failed to clean up etcd data: %v", err)
        }

        // 清除MongoDB中的数据
        collection, err := mongoClient.GetCollection(common.LogCollectionName)
        if err == nil {
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            defer cancel()
            if err := collection.Drop(ctx); err != nil {
                t.Logf("Failed to drop log collection: %v", err)
            }
        }

        if err := etcdClient.Close(); err != nil {
            t.Logf("Failed to close etcd client: %v", err)
        }
        
        if err := mongoClient.Close(); err != nil {
            t.Logf("Failed to close MongoDB client: %v", err)
        }
	}

	return server, cleanup
}

type APITest struct {
	t *testing.T
}

func (a *APITest) doRequest(method, path string, body interface{}) (*http.Response, []byte, error) {
	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
	}

	url := baseURL + path
	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return resp, respBody, nil
}

func (a *APITest) parseResponse(respBody []byte, obj interface{}) (*common.ApiResponse, error) {
	var apiResp common.ApiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, err
	}

	if obj != nil && apiResp.Data != nil {
		dataBytes, err := json.Marshal(apiResp.Data)
		if err != nil {
			return &apiResp, err
		}

		if err := json.Unmarshal(dataBytes, obj); err != nil {
			return &apiResp, err
		}
	}

	return &apiResp, nil
}

func TestJobCRUD(t *testing.T) {
	_, cleanup := setupIntegrationTest(t)
	defer cleanup()

	apiTest := &APITest{t: t}

	job := &common.Job{
		Name:     "test-job-1",
		Command:  "echo hello",
		CronExpr: "*/5 * * * * *",
		Timeout:  60,
	}

	resp, body, err := apiTest.doRequest(http.MethodPost, "/job/save", job)
	require.NoError(t, err, "Failed to create job")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err := apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")

	resp, body, err = apiTest.doRequest(http.MethodGet, "/job/test-job-1", nil)
	require.NoError(t, err, "Failed to get job")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	var getJob common.Job
	apiResp, err = apiTest.parseResponse(body, &getJob)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")
	assert.Equal(t, "test-job-1", getJob.Name, "Job name should match")
	assert.Equal(t, "echo hello", getJob.Command, "Command should match")

	job.Command = "echo hello world"
	resp, body, err = apiTest.doRequest(http.MethodPost, "/job/save", job)
	require.NoError(t, err, "Failed to update job")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err = apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")

	resp, body, err = apiTest.doRequest(http.MethodGet, "/job/test-job-1", nil)
	require.NoError(t, err, "Failed to get updated job")

	var updatedJob common.Job
	apiResp, err = apiTest.parseResponse(body, &updatedJob)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, "echo hello world", updatedJob.Command, "Updated command should match")

	resp, body, err = apiTest.doRequest(http.MethodPost, "/job/disable/test-job-1", nil)
	require.NoError(t, err, "Failed to disable job")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err = apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")

	resp, body, err = apiTest.doRequest(http.MethodGet, "/job/test-job-1", nil)
	require.NoError(t, err, "Failed to get disabled job")

	var disabledJob common.Job
	apiResp, err = apiTest.parseResponse(body, &disabledJob)
	require.NoError(t, err, "Failed to parse response")
	assert.True(t, disabledJob.Disabled, "Job should be disabled")

	resp, body, err = apiTest.doRequest(http.MethodPost, "/job/enable/test-job-1", nil)
	require.NoError(t, err, "Failed to enable job")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err = apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")

	resp, body, err = apiTest.doRequest(http.MethodGet, "/job/test-job-1", nil)
	require.NoError(t, err, "Failed to get enabled job")

	var enabledJob common.Job
	apiResp, err = apiTest.parseResponse(body, &enabledJob)
	require.NoError(t, err, "Failed to parse response")
	assert.False(t, enabledJob.Disabled, "Job should be enabled")

	resp, body, err = apiTest.doRequest(http.MethodDelete, "/job/test-job-1", nil)
	require.NoError(t, err, "Failed to delete job")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err = apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")

	resp, body, err = apiTest.doRequest(http.MethodGet, "/job/test-job-1", nil)
	require.NoError(t, err, "Failed to get deleted job")

	apiResp, err = apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiJobNotExist, apiResp.Code, "Response code should be job not exist")
}

func TestJobList(t *testing.T) {
	_, cleanup := setupIntegrationTest(t)
	defer cleanup()

	apiTest := &APITest{t: t}

	jobs := []*common.Job{
		{Name: "test-job-1", Command: "echo job1", CronExpr: "*/5 * * * * *", Timeout: 60},
		{Name: "test-job-2", Command: "echo job2", CronExpr: "*/10 * * * * *", Timeout: 60},
		{Name: "another-job", Command: "echo another", CronExpr: "*/15 * * * * *", Timeout: 60},
	}

	for _, job := range jobs {
		resp, body, err := apiTest.doRequest(http.MethodPost, "/job/save", job)
		require.NoError(t, err, "Failed to create job")
		require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

		apiResp, err := apiTest.parseResponse(body, nil)
		require.NoError(t, err, "Failed to parse response")
		assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")
	}

	resp, body, err := apiTest.doRequest(http.MethodGet, "/job/list", nil)
	require.NoError(t, err, "Failed to list jobs")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err := apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")

	jobsData, ok := apiResp.Data.([]interface{})
	require.True(t, ok, "Response data should be an array")
	assert.Equal(t, 3, len(jobsData), "Should have 3 jobs")

	resp, body, err = apiTest.doRequest(http.MethodGet, "/job/list?keyword=test", nil)
	require.NoError(t, err, "Failed to search jobs")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err = apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")

	jobsData, ok = apiResp.Data.([]interface{})
	require.True(t, ok, "Response data should be an array")
	assert.Equal(t, 2, len(jobsData), "Should have 2 jobs matching keyword 'test'")
}

func TestWorkersAPI(t *testing.T) {
	_, cleanup := setupIntegrationTest(t)
	defer cleanup()

	apiTest := &APITest{t: t}

	resp, body, err := apiTest.doRequest(http.MethodGet, "/worker/list", nil)
	require.NoError(t, err, "Failed to list workers")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err := apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")

	resp, body, err = apiTest.doRequest(http.MethodGet, "/worker/stats", nil)
	require.NoError(t, err, "Failed to get worker stats")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err = apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")

	statsData, ok := apiResp.Data.(map[string]interface{})
	require.True(t, ok, "Response data should be a map")
	assert.Contains(t, statsData, "total", "Stats should contain total count")
	assert.Contains(t, statsData, "online", "Stats should contain online count")
}

func TestKillJob(t *testing.T) {
	_, cleanup := setupIntegrationTest(t)
	defer cleanup()

	apiTest := &APITest{t: t}

	job := &common.Job{
		Name:     "sleep-job",
		Command:  "sleep 30",
		CronExpr: "*/1 * * * * *",
		Timeout:  60,
	}

	resp, body, err := apiTest.doRequest(http.MethodPost, "/job/save", job)
	require.NoError(t, err, "Failed to create job")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	resp, body, err = apiTest.doRequest(http.MethodPost, "/job/kill/sleep-job", nil)
	require.NoError(t, err, "Failed to kill job")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err := apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")
}

func TestLogAPI(t *testing.T) {
	_, cleanup := setupIntegrationTest(t)
	defer cleanup()

	apiTest := &APITest{t: t}

	resp, body, err := apiTest.doRequest(http.MethodGet, "/log/list?jobName=test-job", nil)
	require.NoError(t, err, "Failed to list logs")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err := apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")

	resp, body, err = apiTest.doRequest(http.MethodGet, "/log/test-job", nil)
	require.NoError(t, err, "Failed to get job log")
	// 这里不验证是否成功，因为可能没有日志

	resp, body, err = apiTest.doRequest(http.MethodGet, "/log/stats/test-job?days=7", nil)
	require.NoError(t, err, "Failed to get log stats")
	require.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be 200")

	apiResp, err = apiTest.parseResponse(body, nil)
	require.NoError(t, err, "Failed to parse response")
	assert.Equal(t, common.ApiSuccess, apiResp.Code, "Response code should be success")
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
