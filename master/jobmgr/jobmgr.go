package jobmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
)

// JobManager 任务管理器，负责任务的CRUD操作
type JobManager struct {
	etcdClient *etcd.Client       // etcd客户端
	logger     *zap.Logger        // 日志对象
	ctx        context.Context    // 上下文，用于控制退出
	cancelFunc context.CancelFunc // 取消函数
}

// NewJobManager 创建任务管理器
func NewJobManager(etcdClient *etcd.Client, logger *zap.Logger) *JobManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &JobManager{
		etcdClient: etcdClient,
		logger:     logger,
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// SaveJob 保存任务
func (jm *JobManager) SaveJob(job *common.Job) error {
	// 更新任务时间戳
	now := time.Now().Unix()
	if job.CreatedAt == 0 {
		job.CreatedAt = now
	}
	job.UpdatedAt = now

	// 序列化为JSON
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %v", err)
	}

	// 保存到etcd
	jobKey := common.JobSaveDir + job.Name
	_, err = jm.etcdClient.Put(jobKey, string(jobData))
	if err != nil {
		jm.logger.Error("failed to save job",
			zap.String("jobName", job.Name),
			zap.Error(err))
		return err
	}

	jm.logger.Info("job saved successfully", zap.String("jobName", job.Name))
	return nil
}

// DeleteJob 删除任务
func (jm *JobManager) DeleteJob(jobName string) error {
	// 删除etcd中的任务
	jobKey := common.JobSaveDir + jobName
	resp, err := jm.etcdClient.Delete(jobKey)

	if err != nil {
		jm.logger.Error("failed to delete job",
			zap.String("jobName", jobName),
			zap.Error(err))
		return err
	}

	// 检查是否找到并删除了任务
	if resp != nil && resp.Deleted == 0 {
		return common.ErrJobNotFound
	}

	jm.logger.Info("job deleted", zap.String("jobName", jobName))
	return nil
}

// GetJob 获取任务
func (jm *JobManager) GetJob(jobName string) (*common.Job, error) {
	// 从etcd获取任务
	jobKey := common.JobSaveDir + jobName
	resp, err := jm.etcdClient.Get(jobKey)
	if err != nil {
		return nil, err
	}

	// 判断是否存在
	if resp.Count == 0 {
		return nil, common.ErrJobNotFound
	}

	// 反序列化
	job := &common.Job{}
	if err = json.Unmarshal(resp.Kvs[0].Value, job); err != nil {
		jm.logger.Error("failed to unmarshal job data",
			zap.String("jobName", jobName),
			zap.Error(err))
		return nil, fmt.Errorf("failed to unmarshal job data: %v", err)
	}

	return job, nil
}

// ListJobs 获取任务列表
func (jm *JobManager) ListJobs() ([]*common.Job, error) {
	// 从etcd获取所有任务
	resp, err := jm.etcdClient.GetWithPrefix(common.JobSaveDir)
	if err != nil {
		jm.logger.Error("failed to list jobs",
			zap.Error(err))
		return nil, err
	}

	// 解析任务列表
	jobs := make([]*common.Job, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		job := &common.Job{}
		if err = json.Unmarshal(kv.Value, job); err != nil {
			jm.logger.Error("failed to unmarshal job data",
				zap.String("key", string(kv.Key)),
				zap.Error(err))
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// KillJob 强制终止任务
func (jm *JobManager) KillJob(jobName string) error {
	// 创建kill标记
	killKey := common.JobLockDir + jobName

	// 上传一个临时的key，worker节点监听到这个key后会停止对应任务
	err := jm.etcdClient.PutWithLease(killKey, "", 5)
	if err != nil {
		jm.logger.Error("failed to create kill marker",
			zap.String("jobName", jobName),
			zap.Error(err))
		return err
	}

	jm.logger.Info("job kill marker created",
		zap.String("jobName", jobName))

	return nil
}

// DisableJob 禁用任务
func (jm *JobManager) DisableJob(jobName string) error {
	// 先获取任务
	job, err := jm.GetJob(jobName)
	if err != nil {
		return err
	}

	// 设置禁用标记
	job.Disabled = true
	job.UpdatedAt = time.Now().Unix()

	// 保存回etcd
	return jm.SaveJob(job)
}

// EnableJob 启用任务
func (jm *JobManager) EnableJob(jobName string) error {
	// 先获取任务
	job, err := jm.GetJob(jobName)
	if err != nil {
		return err
	}

	// 取消禁用标记
	job.Disabled = false
	job.UpdatedAt = time.Now().Unix()

	// 保存回etcd
	return jm.SaveJob(job)
}

// Stop 停止任务管理器
func (jm *JobManager) Stop() {
	jm.cancelFunc()
	jm.logger.Info("job manager stopped")
}

// SearchJobs 搜索任务
func (jm *JobManager) SearchJobs(keyword string) ([]*common.Job, error) {
	// 获取所有任务
	allJobs, err := jm.ListJobs()
	if err != nil {
		return nil, err
	}

	// 如果关键词为空，返回全部
	if keyword == "" {
		return allJobs, nil
	}

	// 过滤匹配关键词的任务
	matchedJobs := make([]*common.Job, 0)
	for _, job := range allJobs {
		// 检查任务名是否包含关键词
		if containsString(job.Name, keyword) || containsString(job.Command, keyword) {
			matchedJobs = append(matchedJobs, job)
		}
	}

	return matchedJobs, nil
}

// 字符串包含检查，不区分大小写
func containsString(source, substr string) bool {
	return containsSubstring(source, substr)
}

// containsSubstring 检查source是否包含substr，不区分大小写
func containsSubstring(source, substr string) bool {
	// 简单实现，实际场景可能需要更复杂的字符串搜索算法
	sourceLen := len(source)
	substrLen := len(substr)

	if substrLen > sourceLen {
		return false
	}

	// 简单遍历查找
	for i := 0; i <= sourceLen-substrLen; i++ {
		match := true
		for j := 0; j < substrLen; j++ {
			// 不区分大小写比较
			if toLower(source[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}

// toLower 将字符转换为小写
func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
