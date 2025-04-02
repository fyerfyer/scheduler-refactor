package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/master/api"
	"github.com/fyerfyer/scheduler-refactor/master/jobmgr"
	"github.com/fyerfyer/scheduler-refactor/master/logmgr"
	"github.com/fyerfyer/scheduler-refactor/master/workermgr"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
	"github.com/fyerfyer/scheduler-refactor/pkg/mongodb"
)

// initLogger 初始化日志
func initLogger() *zap.Logger {
	// 配置日志编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建生产环境配置
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// 构建日志
	logger, err := config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	return logger
}

func main() {
	// 解析命令行参数
	configFile := flag.String("config", "./master.json", "master config file path")
	flag.Parse()

	// 初始化日志
	logger := initLogger()
	defer logger.Sync()

	logger.Info("master starting...")

	// 初始化配置
	if err := config.InitConfig(*configFile, false); err != nil {
		logger.Fatal("failed to initialize config", zap.Error(err))
	}

	// 初始化Etcd客户端
	etcdClient, err := etcd.NewClient()
	if err != nil {
		logger.Fatal("failed to connect to etcd", zap.Error(err))
	}
	defer etcdClient.Close()

	// 初始化MongoDB客户端
	mongoClient, err := mongodb.NewClient()
	if err != nil {
		logger.Fatal("failed to connect to mongodb", zap.Error(err))
	}
	defer mongoClient.Close()

	// 初始化组件
	jobManager := jobmgr.NewJobManager(etcdClient, logger)
	logManager := logmgr.NewLogManager(mongoClient, logger)
	workerManager := workermgr.NewWorkerManager(etcdClient, logger)

	// 启动日志清理器
	logManager.StartLogCleaner(30) // 保留30天的日志

	// 创建API服务器
	apiServer := api.NewServer(logger, jobManager, logManager, workerManager)

	// 启动API服务器
	go func() {
		if err := apiServer.Start(); err != nil {
			logger.Fatal("api server error", zap.Error(err))
		}
	}()

	logger.Info("master started", zap.Int("apiPort", config.GlobalConfig.ApiPort))

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down master...")

	// 创建关闭超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 优雅关闭
	apiServer.Stop()
	jobManager.Stop()
	logManager.Stop()
	workerManager.Stop()

	// 等待所有组件关闭
	select {
	case <-ctx.Done():
		logger.Info("shutdown timeout")
	case <-time.After(2 * time.Second): // 给组件一点时间关闭
		logger.Info("master shutdown complete")
	}
}
