package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/master/jobmgr"
	"github.com/fyerfyer/scheduler-refactor/master/logmgr"
	"github.com/fyerfyer/scheduler-refactor/master/workermgr"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
	"github.com/fyerfyer/scheduler-refactor/pkg/mongodb"
)

func setupTest(t *testing.T) (*Server, *etcd.Client, *mongodb.Client, func()) {
	gin.SetMode(gin.TestMode)

	config.GlobalConfig = &config.Config{
		EtcdEndpoints:       []string{"localhost:2379"},
		EtcdDialTimeout:     5000,
		ApiPort:             8070,
		MongoURI:            "mongodb://localhost:27017",
		MongoConnectTimeout: 5000,
	}

	logger, _ := zap.NewDevelopment()
	etcdClient, err := etcd.NewClient()
	require.NoError(t, err, "Failed to connect to etcd")
	mongoClient, err := mongodb.NewClient()
	require.NoError(t, err, "Failed to connect to MongoDB")

	jobManager := jobmgr.NewJobManager(etcdClient, logger)
	logManager := logmgr.NewLogManager(mongoClient, logger)
	workerManager := workermgr.NewWorkerManager(etcdClient, logger)

	// 创建API服务器
	apiServer := NewServer(logger, jobManager, logManager, workerManager)

	// 返回清理函数
	cleanup := func() {
		// 清理测试数据
		etcdClient.DeleteWithPrefix(common.JobSaveDir)
	}

	return apiServer, etcdClient, mongoClient, cleanup
}

func TestNewServer(t *testing.T) {
	server, _, _, cleanup := setupTest(t)
	defer cleanup()

	assert.NotNil(t, server, "Server should not be nil")
	assert.NotNil(t, server.engine, "Gin engine should not be nil")
	assert.NotNil(t, server.jobMgr, "Job manager should not be nil")
	assert.NotNil(t, server.logMgr, "Log manager should not be nil")
	assert.NotNil(t, server.workerMgr, "Worker manager should not be nil")
}

func TestSaveJob(t *testing.T) {
	server, _, _, cleanup := setupTest(t)
	defer cleanup()

	job := common.Job{
		Name:      "test-job",
		Command:   "echo hello",
		CronExpr:  "*/5 * * * * *",
		Timeout:   60,
		Disabled:  false,
		CreatedAt: 0,
		UpdatedAt: 0,
	}

	jsonData, err := json.Marshal(job)
	require.NoError(t, err, "Failed to marshal job data")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/job/save", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 发送请求
	server.engine.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "HTTP status code should be 200")

	var response common.ApiResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal response")

	assert.Equal(t, common.ApiSuccess, response.Code, "Response code should be success")

	savedJob, err := server.jobMgr.GetJob("test-job")
	assert.NoError(t, err, "Job should be saved")
	assert.Equal(t, "test-job", savedJob.Name, "Job name should match")
	assert.Equal(t, "echo hello", savedJob.Command, "Command should match")
}

func TestListJobs(t *testing.T) {
	server, _, _, cleanup := setupTest(t)
	defer cleanup()

	for i := 1; i <= 3; i++ {
		job := &common.Job{
			Name:      "test-job-" + string(rune(i+'0')),
			Command:   "echo test " + string(rune(i+'0')),
			CronExpr:  "*/5 * * * * *",
			Timeout:   60,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		}

		err := server.jobMgr.SaveJob(job)
		require.NoError(t, err, "Failed to save job for test")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/job/list", nil)

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 发送请求
	server.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTP status code should be 200")

	var response common.ApiResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal response")

	assert.Equal(t, common.ApiSuccess, response.Code, "Response code should be success")

	jobsData, ok := response.Data.([]interface{})
	assert.True(t, ok, "Data should be a job array")
	assert.GreaterOrEqual(t, len(jobsData), 3, "Should have at least 3 jobs")
}

func TestGetJob(t *testing.T) {
	server, _, _, cleanup := setupTest(t)
	defer cleanup()

	job := &common.Job{
		Name:      "test-job",
		Command:   "echo hello",
		CronExpr:  "*/5 * * * * *",
		Timeout:   60,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	err := server.jobMgr.SaveJob(job)
	require.NoError(t, err, "Failed to save job for test")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/job/test-job", nil)
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTP status code should be 200")

	var response common.ApiResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal response")

	assert.Equal(t, common.ApiSuccess, response.Code, "Response code should be success")

	jobData, ok := response.Data.(map[string]interface{})
	assert.True(t, ok, "Data should be a job")
	assert.Equal(t, "test-job", jobData["name"], "Job name should match")
	assert.Equal(t, "echo hello", jobData["command"], "Command should match")
}

func TestGetNonExistentJob(t *testing.T) {
	server, _, _, cleanup := setupTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/job/non-existent-job", nil)

	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTP status code should be 200")

	var response common.ApiResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal response")

	assert.Equal(t, common.ApiJobNotExist, response.Code, "Response code should be job not exist error")
}

func TestDeleteJob(t *testing.T) {
	server, _, _, cleanup := setupTest(t)
	defer cleanup()

	job := &common.Job{
		Name:      "test-job",
		Command:   "echo hello",
		CronExpr:  "*/5 * * * * *",
		Timeout:   60,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	err := server.jobMgr.SaveJob(job)
	require.NoError(t, err, "Failed to save job for test")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/job/test-job", nil)

	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTP status code should be 200")

	var response common.ApiResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal response")

	assert.Equal(t, common.ApiSuccess, response.Code, "Response code should be success")

	_, err = server.jobMgr.GetJob("test-job")
	assert.Error(t, err, "Job should be deleted")
	assert.Equal(t, common.ErrJobNotFound, err, "Error should be job not found")
}

func TestDisableJob(t *testing.T) {
	server, _, _, cleanup := setupTest(t)
	defer cleanup()

	job := &common.Job{
		Name:      "test-job",
		Command:   "echo hello",
		CronExpr:  "*/5 * * * * *",
		Timeout:   60,
		Disabled:  false,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	err := server.jobMgr.SaveJob(job)
	require.NoError(t, err, "Failed to save job for test")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/job/disable/test-job", nil)
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "HTTP status code should be 200")

	var response common.ApiResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal response")

	assert.Equal(t, common.ApiSuccess, response.Code, "Response code should be success")

	job, err = server.jobMgr.GetJob("test-job")
	assert.NoError(t, err, "Job should exist")
	assert.True(t, job.Disabled, "Job should be disabled")
}

func TestEnableJob(t *testing.T) {
	server, _, _, cleanup := setupTest(t)
	defer cleanup()

	job := &common.Job{
		Name:      "test-job",
		Command:   "echo hello",
		CronExpr:  "*/5 * * * * *",
		Timeout:   60,
		Disabled:  true,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	err := server.jobMgr.SaveJob(job)
	require.NoError(t, err, "Failed to save job for test")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/job/enable/test-job", nil)
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTP status code should be 200")

	var response common.ApiResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal response")

	// 验证响应
	assert.Equal(t, common.ApiSuccess, response.Code, "Response code should be success")

	// 验证任务已被启用
	job, err = server.jobMgr.GetJob("test-job")
	assert.NoError(t, err, "Job should exist")
	assert.False(t, job.Disabled, "Job should be enabled")
}

func TestListWorkers(t *testing.T) {
	server, _, _, cleanup := setupTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/worker/list", nil)
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTP status code should be 200")

	var response common.ApiResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal response")

	assert.Equal(t, common.ApiSuccess, response.Code, "Response code should be success")

	_, ok := response.Data.([]interface{})
	assert.True(t, ok, "Data should be a worker array")
}

func TestGetWorkerStats(t *testing.T) {
	server, _, _, cleanup := setupTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/worker/stats", nil)
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTP status code should be 200")

	var response common.ApiResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal response")

	assert.Equal(t, common.ApiSuccess, response.Code, "Response code should be success")

	statsData, ok := response.Data.(map[string]interface{})
	assert.True(t, ok, "Data should be a stats map")
	assert.Contains(t, statsData, "total", "Stats should contain total count")
	assert.Contains(t, statsData, "online", "Stats should contain online count")
}

func TestInvalidRequest(t *testing.T) {
	server, _, _, cleanup := setupTest(t)
	defer cleanup()

	invalidJob := struct {
		Name         string `json:"name"`
		InvalidField int    `json:"invalidField"`
	}{
		Name:         "test-job",
		InvalidField: 123,
	}

	jsonData, err := json.Marshal(invalidJob)
	require.NoError(t, err, "Failed to marshal invalid job data")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/job/save", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HTTP status code should be 200")

	var response common.ApiResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "Failed to unmarshal response")

	assert.Equal(t, common.ApiParamError, response.Code, "Response code should be parameter error")
}
