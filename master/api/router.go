package api

// registerRoutes 注册API路由
func (s *Server) registerRoutes() {
	// API版本分组
	v1 := s.engine.Group("/api/v1")

	// 任务相关接口
	jobGroup := v1.Group("/job")
	{
		jobGroup.POST("/save", s.saveJob)
		jobGroup.DELETE("/:name", s.deleteJob)
		jobGroup.GET("/list", s.listJobs)
		jobGroup.GET("/:name", s.getJob)
		jobGroup.POST("/kill/:name", s.killJob)
		jobGroup.POST("/disable/:name", s.disableJob)
		jobGroup.POST("/enable/:name", s.enableJob)
	}

	// 日志相关接口
	logGroup := v1.Group("/log")
	{
		logGroup.GET("/list", s.listJobLogs)
		logGroup.GET("/:name", s.getJobLog)
		logGroup.GET("/stats/:name", s.getJobLogStats)
	}

	// 工作节点相关接口
	workerGroup := v1.Group("/worker")
	{
		workerGroup.GET("/list", s.listWorkers)
		workerGroup.GET("/stats", s.getWorkerStats)
	}
}
