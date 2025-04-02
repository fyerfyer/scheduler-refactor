package etcd

import (
	"context"
	"time"

	"go.etcd.io/etcd/client/v3"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
)

// Client Etcd客户端封装
type Client struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

// EtcdConfig Etcd配置
type EtcdConfig struct {
	Endpoints   []string
	DialTimeout time.Duration
}

// NewClient 创建Etcd客户端
func NewClient() (*Client, error) {
	cfg := config.GlobalConfig

	// 创建etcd客户端配置
	clientConfig := clientv3.Config{
		Endpoints:   cfg.EtcdEndpoints,
		DialTimeout: time.Duration(cfg.EtcdDialTimeout) * time.Millisecond,
	}

	// 创建etcd客户端
	client, err := clientv3.New(clientConfig)
	if err != nil {
		return nil, common.NewEtcdError("connect", "", err)
	}

	// 创建KV和Lease的API子集
	kv := clientv3.NewKV(client)
	lease := clientv3.NewLease(client)
	watcher := clientv3.NewWatcher(client)

	// 返回封装后的客户端
	return &Client{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.client.Close()
}

// Get 获取键值
func (c *Client) Get(key string) (*clientv3.GetResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.kv.Get(ctx, key)
	if err != nil {
		return nil, common.NewEtcdError("get", key, err)
	}

	return resp, nil
}

// GetWithPrefix 获取前缀匹配的键值
func (c *Client) GetWithPrefix(prefix string) (*clientv3.GetResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.kv.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, common.NewEtcdError("getWithPrefix", prefix, err)
	}

	return resp, nil
}

// Put 设置键值
func (c *Client) Put(key, value string) (*clientv3.PutResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.kv.Put(ctx, key, value)
	if err != nil {
		return nil, common.NewEtcdError("put", key, err)
	}

	return resp, nil
}

// PutWithLease 设置带租约的键值
func (c *Client) PutWithLease(key, value string, ttl int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建租约
	leaseResp, err := c.lease.Grant(ctx, ttl)
	if err != nil {
		return common.NewEtcdError("lease.grant", key, err)
	}

	// 设置带租约的键值
	_, err = c.kv.Put(ctx, key, value, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		return common.NewEtcdError("putWithLease", key, err)
	}

	return nil
}

// KeepAlive 保持租约活跃
func (c *Client) KeepAlive(leaseID clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	ch, err := c.lease.KeepAlive(context.Background(), leaseID)
	if err != nil {
		return nil, common.NewEtcdError("keepAlive", "", err)
	}

	return ch, nil
}

// Delete 删除键值
func (c *Client) Delete(key string) (*clientv3.DeleteResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.kv.Delete(ctx, key)
	if err != nil {
		return nil, common.NewEtcdError("delete", key, err)
	}

	return resp, nil
}

// Watch 监听键值变化
func (c *Client) Watch(key string) clientv3.WatchChan {
	return c.watcher.Watch(context.Background(), key)
}

// WatchWithPrefix 监听前缀下的键值变化
func (c *Client) WatchWithPrefix(prefix string) clientv3.WatchChan {
	return c.watcher.Watch(context.Background(), prefix, clientv3.WithPrefix())
}

// TryAcquireLock 尝试获取分布式锁
func (c *Client) TryAcquireLock(lockKey string, ttl int64) (clientv3.LeaseID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建租约
	leaseResp, err := c.lease.Grant(ctx, ttl)
	if err != nil {
		return 0, common.NewEtcdError("lease.grant", lockKey, err)
	}

	// 尝试获取锁（创建key）
	txn := c.client.Txn(ctx)
	txn = txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0))
	txn = txn.Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseResp.ID)))
	txn = txn.Else(clientv3.OpGet(lockKey))

	txnResp, err := txn.Commit()
	if err != nil {
		return 0, common.NewEtcdError("txn", lockKey, err)
	}

	// 判断事务是否成功
	if !txnResp.Succeeded {
		return 0, common.ErrLockAlreadyAcquired
	}

	return leaseResp.ID, nil
}

// ReleaseLock 释放分布式锁
func (c *Client) ReleaseLock(lockKey string, leaseID clientv3.LeaseID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 撤销租约
	_, err := c.lease.Revoke(ctx, leaseID)
	if err != nil {
		return common.NewEtcdError("revoke", lockKey, err)
	}

	return nil
}

// DeleteWithPrefix 删除前缀匹配的所有键值
func (c *Client) DeleteWithPrefix(prefix string) (*clientv3.DeleteResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.kv.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, common.NewEtcdError("deleteWithPrefix", prefix, err)
	}

	return resp, nil
}
