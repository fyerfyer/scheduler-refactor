package jobmgr

import (
	"context"
	"encoding/json"
	"go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"sync"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/pkg/etcd"
)

// JobManager 任务管理器
type JobManager struct {
	etcdClient *etcd.Client          // etcd客户端
	logger     *zap.Logger           // 日志对象
	jobsCache  sync.Map              // 任务缓存，使用sync.Map实现线程安全
	watchChan  clientv3.WatchChan    // 监听任务变化的通道
	eventChan  chan *common.JobEvent // 任务事件通道
	ctx        context.Context       // 上下文，用于控制退出
	cancelFunc context.CancelFunc    // 取消函数
}

// NewJobManager 创建任务管理器
func NewJobManager(etcdClient *etcd.Client, logger *zap.Logger) *JobManager {
	ctx, cancel := context.WithCancel(context.Background())

	jobMgr := &JobManager{
		etcdClient: etcdClient,
		logger:     logger,
		jobsCache:  sync.Map{},
		eventChan:  make(chan *common.JobEvent, 1000),
		ctx:        ctx,
		cancelFunc: cancel,
	}

	// 任务管理器初始化时，先加载所有任务
	jobMgr.loadJobs()

	// 启动任务变化监听
	jobMgr.watchJobs()

	return jobMgr
}

// loadJobs 加载所有任务
func (jm *JobManager) loadJobs() error {
	// 从etcd获取所有任务
	resp, err := jm.etcdClient.GetWithPrefix(common.JobSaveDir)
	if err != nil {
		jm.logger.Error("failed to load jobs",
			zap.Error(err))
		return err
	}

	// 解析任务
	for _, kv := range resp.Kvs {
		job := &common.Job{}
		err = json.Unmarshal(kv.Value, job)
		if err != nil {
			jm.logger.Error("failed to unmarshal job",
				zap.String("jobKey", string(kv.Key)),
				zap.Error(err))
			continue
		}

		// 缓存任务
		jm.jobsCache.Store(job.Name, job)
	}

	jm.logger.Info("jobs loaded", zap.Int("count", len(resp.Kvs)))
	return nil
}

// watchJobs 监听任务变化
func (jm *JobManager) watchJobs() {
	// 监听/cron/jobs/目录的变化
	jm.watchChan = jm.etcdClient.WatchWithPrefix(common.JobSaveDir)

	// 处理监听事件
	go func() {
		for {
			select {
			case <-jm.ctx.Done():
				return
			case watchResp := <-jm.watchChan:
				for _, event := range watchResp.Events {
					jobEvent := jm.handleWatchEvent(event)
					if jobEvent != nil {
						// 推送事件到通道
						select {
						case jm.eventChan <- jobEvent:
							// 写入成功
						default:
							// 通道已满，记录日志
							jm.logger.Warn("event channel is full, dropping event",
								zap.String("jobName", jobEvent.Job.Name))
						}
					}
				}
			}
		}
	}()

	jm.logger.Info("job watcher started")
}

// handleWatchEvent 处理监听事件
func (jm *JobManager) handleWatchEvent(event *clientv3.Event) *common.JobEvent {
	// 提取Job名称
	var jobName string
	if len(event.Kv.Key) > len(common.JobSaveDir) {
		jobName = string(event.Kv.Key[len(common.JobSaveDir):])
	}

	// 判断事件类型
	var jobEvent *common.JobEvent
	switch event.Type {
	case clientv3.EventTypePut: // 保存任务
		job := &common.Job{}
		if err := json.Unmarshal(event.Kv.Value, job); err != nil {
			jm.logger.Error("failed to unmarshal job",
				zap.String("jobName", jobName),
				zap.Error(err))
			return nil
		}

		// 更新缓存
		jm.jobsCache.Store(job.Name, job)

		// 构造事件
		jobEvent = &common.JobEvent{
			EventType: common.JobEventSave,
			Job:       job,
		}

		jm.logger.Info("job saved", zap.String("jobName", job.Name))

	case clientv3.EventTypeDelete: // 删除任务
		// 从缓存中获取已存在的任务
		jobObj, exists := jm.jobsCache.LoadAndDelete(jobName)
		if exists {
			job, ok := jobObj.(*common.Job)
			if ok {
				// 构造事件
				jobEvent = &common.JobEvent{
					EventType: common.JobEventDelete,
					Job:       job,
				}

				jm.logger.Info("job deleted", zap.String("jobName", job.Name))
			}
		}
	}

	return jobEvent
}

// GetJob 获取任务
func (jm *JobManager) GetJob(jobName string) (*common.Job, bool) {
	jobObj, exists := jm.jobsCache.Load(jobName)
	if !exists {
		return nil, false
	}

	job, ok := jobObj.(*common.Job)
	if !ok {
		return nil, false
	}

	return job, true
}

// ListJobs 获取所有任务
func (jm *JobManager) ListJobs() []*common.Job {
	jobs := make([]*common.Job, 0)

	jm.jobsCache.Range(func(key, value interface{}) bool {
		if job, ok := value.(*common.Job); ok {
			jobs = append(jobs, job)
		}
		return true
	})

	return jobs
}

// GetEventChan 获取事件通道
func (jm *JobManager) GetEventChan() <-chan *common.JobEvent {
	return jm.eventChan
}

// Stop 停止任务管理器
func (jm *JobManager) Stop() {
	// 通知所有协程退出
	jm.cancelFunc()
	jm.logger.Info("job manager stopped")
}
