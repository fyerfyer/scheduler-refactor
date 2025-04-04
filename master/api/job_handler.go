package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
)

// saveJob 保存任务
func (s *Server) saveJob(c *gin.Context) {
	var job common.Job

	// 解析请求
	if err := c.ShouldBindJSON(&job); err != nil {
		failure(c, common.ApiParamError, "invalid job data: "+err.Error())
		return
	}

	// 验证必要字段
	if job.Name == "" {
		failure(c, common.ApiParamError, "job name is required")
		return
	}

	if job.Command == "" {
		failure(c, common.ApiParamError, "job command is required")
		return
	}

	if job.CronExpr == "" {
		failure(c, common.ApiParamError, "job cron expression is required")
		return
	}

	// 验证cron表达式
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(job.CronExpr); err != nil {
		failure(c, common.ApiParamError, "invalid cron expression: "+err.Error())
		return
	}

	// 保存任务
	if err := s.jobMgr.SaveJob(&job); err != nil {
		s.logger.Error("failed to save job",
			zap.String("jobName", job.Name),
			zap.Error(err))
		failure(c, common.ApiFailure, "failed to save job: "+err.Error())
		return
	}

	success(c, job)
}

// deleteJob 删除任务
func (s *Server) deleteJob(c *gin.Context) {
	jobName := c.Param("name")

	// 删除任务
	if err := s.jobMgr.DeleteJob(jobName); err != nil {
		if errors.Is(err, common.ErrJobNotFound) {
			failure(c, common.ApiJobNotExist, "job does not exist")
		} else {
			s.logger.Error("failed to delete job",
				zap.String("jobName", jobName),
				zap.Error(err))
			failure(c, common.ApiFailure, "failed to delete job: "+err.Error())
		}
		return
	}

	success(c, nil)
}

// listJobs 获取任务列表
func (s *Server) listJobs(c *gin.Context) {
	// 获取查询关键字
	keyword := c.Query("keyword")

	// 获取任务列表
	jobs, err := s.jobMgr.SearchJobs(keyword)
	if err != nil {
		s.logger.Error("failed to list jobs", zap.Error(err))
		failure(c, common.ApiSystemError, "failed to list jobs: "+err.Error())
		return
	}

	success(c, jobs)
}

// getJob 获取任务详情
func (s *Server) getJob(c *gin.Context) {
	jobName := c.Param("name")

	// 获取任务
	job, err := s.jobMgr.GetJob(jobName)
	if err != nil {
		if errors.Is(err, common.ErrJobNotFound) {
			failure(c, common.ApiJobNotExist, "job does not exist")
		} else {
			s.logger.Error("failed to get job",
				zap.String("jobName", jobName),
				zap.Error(err))
			failure(c, common.ApiFailure, "failed to get job: "+err.Error())
		}
		return
	}

	success(c, job)
}

// killJob 强制终止任务
func (s *Server) killJob(c *gin.Context) {
	jobName := c.Param("name")

	// 终止任务
	if err := s.jobMgr.KillJob(jobName); err != nil {
		s.logger.Error("failed to kill job",
			zap.String("jobName", jobName),
			zap.Error(err))
		failure(c, common.ApiJobExecFail, "failed to kill job: "+err.Error())
		return
	}

	success(c, nil)
}

// disableJob 禁用任务
func (s *Server) disableJob(c *gin.Context) {
	jobName := c.Param("name")

	// 禁用任务
	if err := s.jobMgr.DisableJob(jobName); err != nil {
		if errors.Is(err, common.ErrJobNotFound) {
			failure(c, common.ApiJobNotExist, "job does not exist")
		} else {
			s.logger.Error("failed to disable job",
				zap.String("jobName", jobName),
				zap.Error(err))
			failure(c, common.ApiFailure, "failed to disable job: "+err.Error())
		}
		return
	}

	success(c, nil)
}

// enableJob 启用任务
func (s *Server) enableJob(c *gin.Context) {
	jobName := c.Param("name")

	// 启用任务
	if err := s.jobMgr.EnableJob(jobName); err != nil {
		if errors.Is(err, common.ErrJobNotFound) {
			failure(c, common.ApiJobNotExist, "job does not exist")
		} else {
			s.logger.Error("failed to enable job",
				zap.String("jobName", jobName),
				zap.Error(err))
			failure(c, common.ApiFailure, "failed to enable job: "+err.Error())
		}
		return
	}

	success(c, nil)
}
