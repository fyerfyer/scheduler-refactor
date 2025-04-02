package common

import (
	"errors"
	"fmt"
)

var (
	// ErrJobNotFound 任务不存在错误
	ErrJobNotFound = errors.New("job not found")

	// ErrLockAlreadyAcquired 锁已被占用错误
	ErrLockAlreadyAcquired = errors.New("lock already acquired")

	// ErrJobSaveConflict 任务保存冲突错误
	ErrJobSaveConflict = errors.New("job save conflict")

	// ErrInvalidCronExpr 无效的cron表达式错误
	ErrInvalidCronExpr = errors.New("invalid cron expression")

	// ErrJobDisabled 任务被禁用错误
	ErrJobDisabled = errors.New("job disabled")

	// ErrJobExecutionTimeout 任务执行超时错误
	ErrJobExecutionTimeout = errors.New("job execution timeout")
)

// JobError 任务相关自定义错误
type JobError struct {
	JobName string
	Err     error
}

// Error 实现error接口
func (e *JobError) Error() string {
	return fmt.Sprintf("job[%s]error: %v", e.JobName, e.Err)
}

// Unwrap 返回原始错误，用于errors.Is和errors.As支持
func (e *JobError) Unwrap() error {
	return e.Err
}

// NewJobError 创建任务错误
func NewJobError(jobName string, err error) *JobError {
	return &JobError{
		JobName: jobName,
		Err:     err,
	}
}

// EtcdError etcd操作相关错误
type EtcdError struct {
	Operation string
	Key       string
	Err       error
}

// Error 实现error接口
func (e *EtcdError) Error() string {
	return fmt.Sprintf("Etcd %s operation error, key=%s: %v", e.Operation, e.Key, e.Err)
}

// Unwrap 返回原始错误
func (e *EtcdError) Unwrap() error {
	return e.Err
}

// NewEtcdError 创建etcd错误
func NewEtcdError(operation, key string, err error) *EtcdError {
	return &EtcdError{
		Operation: operation,
		Key:       key,
		Err:       err,
	}
}

// MongoError MongoDB操作相关错误
type MongoError struct {
	Operation  string
	Collection string
	Err        error
}

// Error 实现error接口
func (e *MongoError) Error() string {
	return fmt.Sprintf("MongoDB %s operation error, collection=%s: %v", e.Operation, e.Collection, e.Err)
}

// Unwrap 返回原始错误
func (e *MongoError) Unwrap() error {
	return e.Err
}

// NewMongoError 创建MongoDB错误
func NewMongoError(operation, collection string, err error) *MongoError {
	return &MongoError{
		Operation:  operation,
		Collection: collection,
		Err:        err,
	}
}
