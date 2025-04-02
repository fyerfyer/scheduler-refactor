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
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
	"github.com/fyerfyer/scheduler-refactor/pkg/mongodb"
	"github.com/fyerfyer/scheduler-refactor/worker/executor"
	"github.com/fyerfyer/scheduler-refactor/worker/jobmgr"
	"github.com/fyerfyer/scheduler-refactor/worker/logsink"
	"github.com/fyerfyer/scheduler-refactor/worker/register"
	"github.com/fyerfyer/scheduler-refactor/worker/scheduler"
)

// 全局组件
type workerContext struct {
	logger      *zap.Logger
	etcdClient  *etcd.Client
	mongoClient *mongodb.Client
	executor    *executor.Executor
	jobManager  *jobmgr.JobManager
	register    *register.Register
	scheduler   *scheduler.Scheduler
	logSink     *logsink.LogSink
}

func main() {
	var (
		configFile string
		err        error
	)

	// 解析命令行参数
	flag.StringVar(&configFile, "config", "./worker.json", "worker config file path")
	flag.Parse()

	// 加载配置
	if err = config.InitConfig(configFile); err != nil {
		panic(err)
	}

	// 创建Worker上下文
	wctx := &workerContext{}

	// 初始化组件
	if err = initWorker(wctx); err != nil {
		panic(err)
	}

	// 启动组件
	startWorker(wctx)

	// 等待退出信号
	waitForExit(wctx)
}

// initLogger 初始化日志
func initLogger() *zap.Logger {
	// 配置zap logger
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, _ := config.Build()
	return logger
}

// initWorker 初始化Worker组件
func initWorker(wctx *workerContext) error {
	var err error

	// 初始化日志
	wctx.logger = initLogger()

	// 初始化etcd客户端
	if wctx.etcdClient, err = etcd.NewClient(); err != nil {
		wctx.logger.Error("failed to create etcd client", zap.Error(err))
		return err
	}

	// 初始化MongoDB客户端
	if wctx.mongoClient, err = mongodb.NewClient(); err != nil {
		wctx.logger.Error("failed to create mongodb client", zap.Error(err))
		return err
	}

	// 初始化执行器
	wctx.executor = executor.NewExecutor(wctx.logger)

	// 初始化任务管理器
	wctx.jobManager = jobmgr.NewJobManager(wctx.etcdClient, wctx.logger)

	// 初始化注册器
	wctx.register = register.NewRegister(wctx.logger, wctx.etcdClient)

	// 初始化调度器
	wctx.scheduler = scheduler.NewScheduler(wctx.logger, wctx.jobManager, wctx.etcdClient, wctx.executor)

	// 初始化日志收集器
	wctx.logSink = logsink.NewLogSink(wctx.mongoClient, wctx.logger)

	return nil
}

// startWorker 启动Worker组件
func startWorker(wctx *workerContext) {
	// 启动Worker注册
	if err := wctx.register.Start(); err != nil {
		wctx.logger.Error("failed to start worker register", zap.Error(err))
		return
	}
	wctx.logger.Info("worker register started")

	// 启动任务调度器
	wctx.scheduler.Start()
	wctx.logger.Info("job scheduler started")

	// 启动日志清理器
	cleanCtx, _ := context.WithCancel(context.Background())
	wctx.logSink.StartLogCleaner(cleanCtx, 7) // 默认保留7天日志
	wctx.logger.Info("log cleaner started")

	// 注册执行结果处理器
	go handleExecuteResults(wctx)

	wctx.logger.Info("worker started successfully",
		zap.String("workerId", config.GlobalConfig.WorkerID),
		zap.Strings("etcdEndpoints", config.GlobalConfig.EtcdEndpoints))
}

// handleExecuteResults 处理任务执行结果
func handleExecuteResults(wctx *workerContext) {
	resultChan := wctx.executor.GetResultChan()

	for result := range resultChan {
		// 查找任务执行信息
		jobInfo, exists := wctx.scheduler.GetExecutingJobs()[result.JobName]
		if exists {
			// 构建日志
			jobLog := executor.BuildJobLog(result, jobInfo)

			// 发送到日志收集器
			wctx.logSink.Append(jobLog)
		}
	}
}

// waitForExit 等待退出信号并优雅关闭
func waitForExit(wctx *workerContext) {
	// 创建接收信号的通道
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	sig := <-sigChan
	wctx.logger.Info("received signal, starting graceful shutdown", zap.String("signal", sig.String()))

	// 给清理操作设定一个超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 首先停止调度器
	wctx.scheduler.Stop()
	wctx.logger.Info("scheduler stopped")

	// 停止Worker注册
	wctx.register.Stop()
	wctx.logger.Info("worker register stopped")

	// 确保日志收集器写入所有缓存日志
	wctx.logSink.Stop()
	wctx.logger.Info("log sink stopped")

	// 关闭MongoDB连接
	if err := wctx.mongoClient.Close(); err != nil {
		wctx.logger.Error("failed to close mongodb connection", zap.Error(err))
	}

	// 关闭etcd连接
	if err := wctx.etcdClient.Close(); err != nil {
		wctx.logger.Error("failed to close etcd connection", zap.Error(err))
	}

	// 等待超时或者所有清理工作完成
	<-ctx.Done()
	wctx.logger.Info("worker shutdown complete")
}
