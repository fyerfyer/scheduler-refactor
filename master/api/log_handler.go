package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"

	"github.com/fyerfyer/scheduler-refactor/common"
)

// listJobLogs 获取任务日志列表
func (s *Server) listJobLogs(c *gin.Context) {
	jobName := c.Query("jobName")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", strconv.Itoa(common.DefaultPageSize)))

	// 获取日志
	logs, total, err := s.logMgr.ListLogs(jobName, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list job logs",
			zap.String("jobName", jobName),
			zap.Error(err))
		failure(c, common.ApiDbError, "failed to list job logs: "+err.Error())
		return
	}

	// 构建分页数据
	result := map[string]interface{}{
		"logs":  logs,
		"total": total,
		"page":  page,
		"size":  pageSize,
	}

	success(c, result)
}

// getJobLog 获取任务最新日志
func (s *Server) getJobLog(c *gin.Context) {
	jobName := c.Param("name")

	// 获取最新日志
	log, err := s.logMgr.GetJobLog(jobName)
	if err != nil {
		if errors.Is(err, common.ErrJobNotFound) {
			failure(c, common.ApiJobNotExist, "no logs found for job")
		} else {
			s.logger.Error("failed to get job log",
				zap.String("jobName", jobName),
				zap.Error(err))
			failure(c, common.ApiDbError, "failed to get job log: "+err.Error())
		}
		return
	}

	success(c, log)
}

// getJobLogStats 获取任务日志统计
func (s *Server) getJobLogStats(c *gin.Context) {
	jobName := c.Param("name")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	// 获取统计信息
	stats, err := s.logMgr.GetLogStatistics(jobName, days)
	if err != nil {
		s.logger.Error("failed to get job log statistics",
			zap.String("jobName", jobName),
			zap.Error(err))
		failure(c, common.ApiDbError, "failed to get job log statistics: "+err.Error())
		return
	}

	success(c, stats)
}
