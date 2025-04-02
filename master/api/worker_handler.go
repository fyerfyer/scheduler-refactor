package api

import (
	"github.com/gin-gonic/gin"
)

// listWorkers 获取工作节点列表
func (s *Server) listWorkers(c *gin.Context) {
	// 获取节点列表
	workers := s.workerMgr.ListWorkers()

	// 获取健康状态
	healthStatus := s.workerMgr.CheckWorkers()

	// 构建返回数据
	result := make([]map[string]interface{}, 0, len(workers))
	for _, worker := range workers {
		status, exists := healthStatus[worker.IP]
		if !exists {
			status = "unknown"
		}

		workerInfo := map[string]interface{}{
			"ip":       worker.IP,
			"hostname": worker.Hostname,
			"cpuUsage": worker.CPUUsage,
			"memUsage": worker.MemUsage,
			"lastSeen": worker.LastSeen,
			"status":   status,
		}
		result = append(result, workerInfo)
	}

	success(c, result)
}

// getWorkerStats 获取工作节点统计信息
func (s *Server) getWorkerStats(c *gin.Context) {
	// 获取统计信息
	stats := s.workerMgr.GetWorkerStats()
	success(c, stats)
}
