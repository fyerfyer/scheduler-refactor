package config

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
)

// Config 系统配置结构体
type Config struct {
	// master和worker共用配置
	EtcdEndpoints   []string `json:"etcdEndpoints"`   // etcd集群地址
	EtcdDialTimeout int      `json:"etcdDialTimeout"` // etcd连接超时时间(毫秒)

	// worker配置
	WorkerID          string `json:"workerId"`          // worker唯一标识
	HeartbeatInterval int    `json:"heartbeatInterval"` // 心跳间隔(毫秒)
	LogBatchSize      int    `json:"logBatchSize"`      // 日志批处理大小
	LogCommitTimeout  int    `json:"logCommitTimeout"`  // 日志提交超时(毫秒)
	ExecutorThreads   int    `json:"executorThreads"`   // 执行器线程数
	JobLockTTL        int    `json:"jobLockTtl"`        // 任务锁超时时间(秒)

	// master配置
	ApiPort             int    `json:"apiPort"`             // API服务端口
	MongoURI            string `json:"mongoUri"`            // MongoDB连接URI
	MongoConnectTimeout int    `json:"mongoConnectTimeout"` // MongoDB连接超时(毫秒)
}

// 全局配置单例
var GlobalConfig *Config

// InitConfig 初始化配置
func InitConfig(configFile string, parseFlags bool) error {
	// 创建默认配置
	GlobalConfig = &Config{
		EtcdEndpoints:       []string{"localhost:2379"},
		EtcdDialTimeout:     5000,
		WorkerID:            "",
		HeartbeatInterval:   5000,
		LogBatchSize:        100,
		LogCommitTimeout:    1000,
		ExecutorThreads:     10,
		JobLockTTL:          5,
		ApiPort:             8070,
		MongoURI:            "mongodb://localhost:27017",
		MongoConnectTimeout: 5000,
	}

	// 先从配置文件加载
	if configFile != "" {
		if err := loadFromFile(configFile); err != nil {
			return err
		}
	}

	// 再从环境变量加载，环境变量优先级高于配置文件
	loadFromEnv()

	// 最后从命令行参数加载，命令行参数优先级最高
	if parseFlags {
		loadFromFlags()
	}

	// 生成默认的WorkerID（如果未指定）
	if GlobalConfig.WorkerID == "" {
		hostname, _ := os.Hostname()
		GlobalConfig.WorkerID = hostname
	}

	return nil
}

// loadFromFile 从配置文件加载配置
func loadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, GlobalConfig)
}

// loadFromEnv 从环境变量加载配置
func loadFromEnv() {
	// Etcd相关配置
	if etcdEndpoints := os.Getenv("ETCD_ENDPOINTS"); etcdEndpoints != "" {
		GlobalConfig.EtcdEndpoints = []string{etcdEndpoints}
	}
	if timeout := os.Getenv("ETCD_DIAL_TIMEOUT"); timeout != "" {
		if value, err := strconv.Atoi(timeout); err == nil {
			GlobalConfig.EtcdDialTimeout = value
		}
	}

	// Worker配置
	if workerID := os.Getenv("WORKER_ID"); workerID != "" {
		GlobalConfig.WorkerID = workerID
	}
	if interval := os.Getenv("HEARTBEAT_INTERVAL"); interval != "" {
		if value, err := strconv.Atoi(interval); err == nil {
			GlobalConfig.HeartbeatInterval = value
		}
	}
	if batchSize := os.Getenv("LOG_BATCH_SIZE"); batchSize != "" {
		if value, err := strconv.Atoi(batchSize); err == nil {
			GlobalConfig.LogBatchSize = value
		}
	}

	// Master配置
	if port := os.Getenv("API_PORT"); port != "" {
		if value, err := strconv.Atoi(port); err == nil {
			GlobalConfig.ApiPort = value
		}
	}
	if mongoURI := os.Getenv("MONGO_URI"); mongoURI != "" {
		GlobalConfig.MongoURI = mongoURI
	}
}

// loadFromFlags 从命令行参数加载配置
func loadFromFlags() {
	// 定义命令行参数，但不解析
	configFilePtr := flag.String("config", "", "Config file path")
	etcdEndpointsPtr := flag.String("etcd", "", "Etcd endpoints (comma separated)")
	apiPortPtr := flag.Int("api-port", 0, "API server port")
	workerIDPtr := flag.String("worker-id", "", "Worker unique ID")
	mongoURIPtr := flag.String("mongo", "", "MongoDB URI")

	// 解析命令行参数
	flag.Parse()

	// 如果指定了配置文件，且之前没有加载过配置文件，那么加载它
	if *configFilePtr != "" && GlobalConfig.EtcdEndpoints == nil {
		loadFromFile(*configFilePtr)
	}

	// 应用命令行参数（如果有指定的话）
	if *etcdEndpointsPtr != "" {
		GlobalConfig.EtcdEndpoints = []string{*etcdEndpointsPtr}
	}
	if *apiPortPtr != 0 {
		GlobalConfig.ApiPort = *apiPortPtr
	}
	if *workerIDPtr != "" {
		GlobalConfig.WorkerID = *workerIDPtr
	}
	if *mongoURIPtr != "" {
		GlobalConfig.MongoURI = *mongoURIPtr
	}
}
