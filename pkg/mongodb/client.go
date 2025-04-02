package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/fyerfyer/scheduler-refactor/common"
	"github.com/fyerfyer/scheduler-refactor/config"
)

// Client MongoDB客户端封装
type Client struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// NewClient 创建MongoDB客户端
func NewClient() (*Client, error) {
	cfg := config.GlobalConfig

	// 创建MongoDB连接上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.MongoConnectTimeout)*time.Millisecond)
	defer cancel()

	// 创建MongoDB客户端
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, common.NewMongoError("connect", "", err)
	}

	// 检查连接
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, common.NewMongoError("ping", "", err)
	}

	// 默认使用"cron"数据库和"job_logs"集合
	database := client.Database("cron")
	collection := database.Collection(common.LogCollectionName)

	// 创建索引
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "jobName", Value: 1},
			{Key: "startTime", Value: -1},
		},
	}

	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return nil, common.NewMongoError("create_index", common.LogCollectionName, err)
	}

	return &Client{
		client:     client,
		database:   database,
		collection: collection,
	}, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.Disconnect(ctx)
}

// InsertOne 插入单个文档
func (c *Client) InsertOne(doc interface{}) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := c.collection.InsertOne(ctx, doc)
	if err != nil {
		return nil, common.NewMongoError("insert", common.LogCollectionName, err)
	}

	return result, nil
}

// InsertMany 批量插入文档
func (c *Client) InsertMany(docs []interface{}) (*mongo.InsertManyResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := c.collection.InsertMany(ctx, docs)
	if err != nil {
		return nil, common.NewMongoError("insert_many", common.LogCollectionName, err)
	}

	return result, nil
}

// Find 查询文档
func (c *Client) Find(filter interface{}, options *options.FindOptions) (*mongo.Cursor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cur, err := c.collection.Find(ctx, filter, options)
	if err != nil {
		return nil, common.NewMongoError("find", common.LogCollectionName, err)
	}

	return cur, nil
}

// FindJobLogs 查询任务日志
func (c *Client) FindJobLogs(jobName string, skip, limit int64) ([]*common.JobLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建查询过滤器
	filter := bson.M{}
	if jobName != "" {
		filter["jobName"] = jobName
	}

	// 设置查询选项
	opts := options.Find().
		SetSort(bson.D{{Key: "startTime", Value: -1}}). // 按开始时间降序排序
		SetSkip(skip).
		SetLimit(limit)

	// 执行查询
	cursor, err := c.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, common.NewMongoError("find_job_logs", common.LogCollectionName, err)
	}
	defer cursor.Close(ctx)

	// 解析结果
	var logs []*common.JobLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, common.NewMongoError("cursor_all", common.LogCollectionName, err)
	}

	return logs, nil
}

// CountJobLogs 计算任务日志总数
func (c *Client) CountJobLogs(jobName string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建查询过滤器
	filter := bson.M{}
	if jobName != "" {
		filter["jobName"] = jobName
	}

	// 计数
	count, err := c.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, common.NewMongoError("count", common.LogCollectionName, err)
	}

	return count, nil
}

// DeleteOldLogs 删除过期日志
func (c *Client) DeleteOldLogs(before time.Time) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 创建过滤器，删除时间戳早于指定时间的日志
	filter := bson.M{"endTime": bson.M{"$lt": before.Unix()}}

	// 执行删除
	result, err := c.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, common.NewMongoError("delete_old_logs", common.LogCollectionName, err)
	}

	return result.DeletedCount, nil
}

// DropCollection 删除集合
func (c *Client) DropCollection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.collection.Drop(ctx)
	if err != nil {
		return common.NewMongoError("drop_collection", common.LogCollectionName, err)
	}

	return nil
}
