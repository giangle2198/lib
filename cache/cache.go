package cache

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis"
	"go.uber.org/zap"
)

type CacheMessage struct {
	redis.Message
}

type SubscribeFunc func(CacheMessage) error

type CacheHelper interface {
	Exists(ctx context.Context, key string) error
	Get(ctx context.Context, key string, value interface{}) error
	GetInterface(ctx context.Context, key string, value interface{}) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, key string) error
	Expire(ctx context.Context, key string, expiration time.Duration) error
	DelMulti(ctx context.Context, keys ...string) error
	GetKeysByPattern(ctx context.Context, pattern string, cursor uint64, limit int64) ([]string, uint64, error)
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	RenameKey(ctx context.Context, oldKey, newKey string) error
	GetType(ctx context.Context, key string) (string, error)
}

type CacheHelperEnhancement interface {
	CacheHelper
	GetTransaction(ctx context.Context, transactionID string) CacheTransactionExecution
	GetPipeline(ctx context.Context, transactionID string) CachePipelineExecution
}

type CacheCommandType string

const (
	CacheCommandTypeGet                    CacheCommandType = "CacheCommandTypeGet"
	CacheCommandTypeGetInterface           CacheCommandType = "CacheCommandTypeGetInterface"
	CacheCommandTypeAddMemberWithScore     CacheCommandType = "CacheCommandTypeAddMemberWithScore"
	CacheCommandTypeGetMembersWithScore    CacheCommandType = "CacheCommandTypeGetMembersWithScore"
	CacheCommandTypeRemoveMembersWithScore CacheCommandType = "CacheCommandTypeRemoveMembersWithScore"
	CacheCommandTypeExpire                 CacheCommandType = "CacheCommandTypeExpire"
	CacheCommandTypeSetNX                  CacheCommandType = "CacheCommandTypeSetNX"
	CacheCommandTypeIncrease               CacheCommandType = "CacheCommandTypeIncrease"
	CacheCommandTypeDel                    CacheCommandType = "CacheCommandTypeDel"
)

type (
	CacheLazyExecute interface {
		Exec(context.Context) ([]CachePipelineResult, error)
		Discard(context.Context) error
	}

	CacheMutilCommandBuilder interface {
		BuildCommand(ctx context.Context, cacheCommandType CacheCommandType, data ...interface{}) error
		GetCommands(context.Context) (CacheLazyExecute, error)
	}

	CacheTransactionExecution interface {
		CacheMutilCommandBuilder
	}

	CachePipelineExecution interface {
		CacheMutilCommandBuilder
	}

	CachePipelineResult struct {
		Result []interface{}
		Err    error
	}

	baseRedisCachePipeline struct {
		redis.Pipeliner
		transactionID string
	}

	redisCacheTransaction struct {
		baseRedisCachePipeline
	}

	redisCachePipeline struct {
		baseRedisCachePipeline
	}

	RedisZSliceResult struct {
		redis.Z
	}
)

func (r *baseRedisCachePipeline) Exec(context.Context) (result []CachePipelineResult, err error) {
	var (
		outputResult []redis.Cmder
	)

	outputResult, err = r.Pipeliner.Exec()
	if err != nil {
		return nil, err
	}

	result = make([]CachePipelineResult, len(outputResult))
	for index, item := range outputResult {
		switch v := item.(type) {
		case *redis.ZSliceCmd:
			resutlReturn := make([]interface{}, len(v.Val()))
			for i, val := range v.Val() {
				resutlReturn[i] = RedisZSliceResult{
					Z: val,
				}
			}
			result[index] = CachePipelineResult{
				Result: resutlReturn,
				Err:    item.Err(),
			}
		case *redis.StringSliceCmd:
			resutlReturn := make([]interface{}, len(v.Val()))
			for i, val := range v.Val() {
				resutlReturn[i] = val
			}
			result[index] = CachePipelineResult{
				Result: resutlReturn,
				Err:    item.Err(),
			}
		case *redis.StringCmd:
			result[index] = CachePipelineResult{
				Result: []interface{}{v.Val()},
				Err:    item.Err(),
			}
		case *redis.IntCmd:
			result[index] = CachePipelineResult{
				Result: []interface{}{v.Val()},
				Err:    item.Err(),
			}
		default:
			result[index] = CachePipelineResult{
				Result: item.Args(),
				Err:    item.Err(),
			}
		}
	}
	return result, nil
}

func (r *baseRedisCachePipeline) Discard(context.Context) error {
	return r.Pipeliner.Discard()
}

func (r *baseRedisCachePipeline) BuildCommand(ctx context.Context, cacheCommandType CacheCommandType, data ...interface{}) (err error) {
	if len(data) == 0 {
		return errors.New("missing data to process")
	}

	var (
		keyCache string = data[0].(string)
		cmd      redis.Cmder
	)

	switch cacheCommandType {
	case CacheCommandTypeGetInterface:
		cmd = r.Pipeliner.Get(keyCache)
	case CacheCommandTypeAddMemberWithScore:
		member := redis.Z{
			Member: data[1],
			Score:  data[2].(float64),
		}
		cmd = r.Pipeliner.ZAdd(keyCache, member)
	case CacheCommandTypeRemoveMembersWithScore:
		var (
			min string = data[1].(string)
			max string = data[2].(string)
		)
		cmd = r.Pipeliner.ZRemRangeByScore(keyCache, min, max)
	case CacheCommandTypeExpire:
		var (
			value    interface{}   = data[1]
			ttl      uint32        = data[2].(uint32)
			duration time.Duration = data[3].(time.Duration)
		)

		cmd = r.Pipeliner.SetNX(keyCache, value, time.Duration(ttl)*(duration))
	case CacheCommandTypeIncrease:
		cmd = r.Pipeliner.Incr(keyCache)
	case CacheCommandTypeDel:
		cmd = r.Pipeliner.Del(keyCache)
	default:
		return errors.New("not found any matched command type to process")
	}
	if cmd != nil && cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

func (r *baseRedisCachePipeline) GetCommands(context.Context) (CacheLazyExecute, error) {
	return r, nil
}

// CacheOption represents cache option
type CacheOption struct {
	Key   string
	Value interface{}
}

func NewCacheHelper(addrs []string, opts ...CacheOption) CacheHelper {
	if len(addrs) > 1 {
		clusterClient, err := initRedisCluster(addrs)
		if err != nil {
			zap.S().Panic("Failed to inti redis cluster", zap.Error(err))
		}
		return &clusterRedisHelper{
			clusterClient: clusterClient,
		}
	}

	// get db config
	var db int = 0
	for _, item := range opts {
		if item.Key == "db" {
			db = item.Value.(int)
		}
	}
	client, err := initRedis(addrs[0], db)
	if err != nil {
		zap.S().Panic("Failed to init redis", zap.Error(err))
	}
	return &redisHelper{
		client: client,
	}
}
