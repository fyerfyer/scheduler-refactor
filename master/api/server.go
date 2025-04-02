package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/config"
	"github.com/fyerfyer/scheduler-refactor/master/jobmgr"
	"github.com/fyerfyer/scheduler-refactor/master/logmgr"
	"github.com/fyerfyer/scheduler-refactor/master/workermgr"
)

// Server API服务器
type Server struct {
	engine    *gin.Engine              // gin引擎
	logger    *zap.Logger              // 日志对象
	jobMgr    *jobmgr.JobManager       // 任务管理器
	logMgr    *logmgr.LogManager       // 日志管理器
	workerMgr *workermgr.WorkerManager // 工作节点管理器
}

// NewServer 创建API服务器
func NewServer(
	logger *zap.Logger,
	jobMgr *jobmgr.JobManager,
	logMgr *logmgr.LogManager,
	workerMgr *workermgr.WorkerManager,
) *Server {
	// 创建gin引擎
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	// 使用恢复中间件
	engine.Use(gin.Recovery())

	// 创建服务器
	server := &Server{
		engine:    engine,
		logger:    logger,
		jobMgr:    jobMgr,
		logMgr:    logMgr,
		workerMgr: workerMgr,
	}

	// 注册路由
	server.registerRoutes()

	return server
}

// Start 启动API服务器
func (s *Server) Start() error {
	port := config.GlobalConfig.ApiPort
	s.logger.Info("starting API server", zap.Int("port", port))

	return s.engine.Run(fmt.Sprintf(":%d", port))
}

// Stop 停止API服务器
func (s *Server) Stop() {
	s.logger.Info("API server stopped")
}
