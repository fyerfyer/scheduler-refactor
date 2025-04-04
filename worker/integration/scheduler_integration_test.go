package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
)

const (
	apiBaseURL  = "http://localhost:8070/api/v1"
	testJobName = "integration_test_job"
	etcdPrefix  = "/cron/"
	mongoDBURI  = "mongodb://localhost:27017"
)

func setupCleanEnvironment(t *testing.T) (*clientv3.Client, *mongo.Client) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	require.NoError(t, err, "Failed to connect to etcd")

	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoDBURI))
	require.NoError(t, err, "Failed to connect to MongoDB")

	cleanupSchedulerTestData(t, etcdClient, mongoClient)

	return etcdClient, mongoClient
}

func cleanupSchedulerTestData(t *testing.T, etcdClient *clientv3.Client, mongoClient *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jobKey := etcdPrefix + "jobs/" + testJobName
	_, err := etcdClient.Delete(ctx, jobKey)
	if err != nil {
		t.Logf("Warning: Failed to delete test job from etcd: %v", err)
	}

	collection := mongoClient.Database("cron").Collection(common.LogCollectionName)
	_, err = collection.DeleteMany(ctx, bson.M{"jobName": testJobName})
	if err != nil {
		t.Logf("Warning: Failed to delete test job logs from MongoDB: %v", err)
	}
}

func createSchedulerTestJob(t *testing.T) {
	job := map[string]interface{}{
		"name":     testJobName,
		"command":  "echo \"Hello world from integration test\"",
		"cronExpr": "* * * * * *", // Changed to run every second
	}

	jsonData, err := json.Marshal(job)
	require.NoError(t, err, "Failed to marshal job data")

	resp, err := http.Post(
		fmt.Sprintf("%s/job/save", apiBaseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	require.NoError(t, err, "Failed to create test job via API")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK response from job creation")

	var apiResp common.ApiResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	require.NoError(t, err, "Failed to decode API response")
	require.Equal(t, common.ApiSuccess, apiResp.Code, "Job creation was not successful")

	t.Logf("Test job created successfully: %s", testJobName)
}

func getLogCount(t *testing.T) int {
	resp, err := http.Get(fmt.Sprintf("%s/log/stats/%s", apiBaseURL, testJobName))
	require.NoError(t, err, "Failed to get job log stats")
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Raw log stats response: %s", string(body))

	var apiResp common.ApiResponse
	err = json.Unmarshal(body, &apiResp)
	require.NoError(t, err, "Failed to decode API response")

	t.Logf("API response code: %d, message: %s", apiResp.Code, apiResp.Message)

	if stats, ok := apiResp.Data.(map[string]interface{}); ok {
		t.Logf("Stats data: %+v", stats)
		if totalCount, ok := stats["totalCount"].(float64); ok {
			return int(totalCount)
		}
	}
	return 0
}

func TestSecondlyJobExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	config.OutputPaths = []string{"stdout"}
	_, _ = config.Build()

	etcdClient, mongoClient := setupCleanEnvironment(t)
	defer etcdClient.Close()
	defer mongoClient.Disconnect(context.Background())

	createSchedulerTestJob(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	jobKey := etcdPrefix + "jobs/" + testJobName
	resp, err := etcdClient.Get(ctx, jobKey)
	t.Logf("Job verification in etcd - Error: %v, Count: %d", err, resp.Count)
	if resp.Count > 0 {
		t.Logf("Job data in etcd: %s", string(resp.Kvs[0].Value))
	}

	t.Log("Waiting for job to be registered...")
	time.Sleep(3 * time.Second)

	initialCount := getLogCount(t)
	t.Logf("Initial log count: %d", initialCount)

	t.Log("Waiting for job execution...")
	time.Sleep(5 * time.Second)

	firstCheckCount := getLogCount(t)
	t.Logf("Log count after ~5 seconds: %d", firstCheckCount)
	assert.Greater(t, firstCheckCount, initialCount, "Job should have executed at least once within 5 seconds")

	t.Log("Waiting for another job execution...")
	time.Sleep(5 * time.Second)

	secondCheckCount := getLogCount(t)
	t.Logf("Log count after ~10 seconds: %d", secondCheckCount)
	assert.Greater(t, secondCheckCount, firstCheckCount, "Job should have executed again within the second 5 seconds")

	cleanupSchedulerTestData(t, etcdClient, mongoClient)
}

func TestJobLogsContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	etcdClient, mongoClient := setupCleanEnvironment(t)
	defer etcdClient.Close()
	defer mongoClient.Disconnect(context.Background())

	createSchedulerTestJob(t)

	t.Log("Waiting for job execution...")
	time.Sleep(65 * time.Second)

	resp, err := http.Get(fmt.Sprintf("%s/log/%s", apiBaseURL, testJobName))
	require.NoError(t, err, "Failed to get job log")
	defer resp.Body.Close()

	var apiResp common.ApiResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	require.NoError(t, err, "Failed to decode API response")
	require.Equal(t, common.ApiSuccess, apiResp.Code, "Failed to retrieve job log")

	if log, ok := apiResp.Data.(map[string]interface{}); ok {
		if output, ok := log["output"].(string); ok {
			assert.Contains(t, output, "Hello world from integration test", "Job output should contain expected message")
		} else {
			t.Fatal("Log output not found or invalid type")
		}
	} else {
		t.Fatal("Log data not found or invalid type")
	}

	cleanupSchedulerTestData(t, etcdClient, mongoClient)
}
