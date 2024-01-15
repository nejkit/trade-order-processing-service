package storage

import (
	"context"
	"time"
	"trade-order-processing-service/staticerr"

	redisLib "github.com/redis/go-redis/v9"
	logger "github.com/sirupsen/logrus"
)

type LimitOptions struct {
	minPrice float64
	maxPrice float64
}

type TxContainer struct {
	tx redisLib.Pipeliner
}

type RedisClient struct {
	cli *redisLib.Client
}

func NewRedisClient(host string) (*RedisClient, error) {
	cli := redisLib.NewClient(&redisLib.Options{
		Addr:     host,
		Password: "",
		DB:       0,
	})

	pong, err := cli.Ping(context.Background()).Result()

	if err != nil {
		return nil, err
	}

	logger.Infoln(pong)
	return &RedisClient{cli: cli}, nil
}

func (r *RedisClient) setNX(ctx context.Context, key string, value interface{}, expire time.Duration) error {
	setted, err := r.cli.SetNX(ctx, key, value, expire).Result()

	if err != nil {
		return err
	}

	if !setted {
		return staticerr.ErrorResourceIsLocked
	}

	return nil
}

func (r *RedisClient) deleteWithValue(ctx context.Context, key string, value interface{}) error {
	err := r.cli.Watch(ctx, func(tx *redisLib.Tx) error {
		valueFromRedis, err := tx.Get(ctx, key).Result()

		if err != nil {
			return err
		}

		if valueFromRedis != value {
			return staticerr.ErrorResourceIsLocked
		}

		_, err = tx.Del(ctx, key).Result()

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *RedisClient) deleteKey(ctx context.Context, id string) error {
	_, err := r.cli.Del(ctx, id).Result()

	if err != nil {
		return err
	}

	return nil
}

func (x *TxContainer) addInZSet(ctx context.Context, key string, value interface{}, weight float64) *TxContainer {
	x.tx.ZAdd(ctx, key, redisLib.Z{Score: weight, Member: value})

	return x
}

func (x *TxContainer) removeFromZSet(ctx context.Context, key string, value interface{}) *TxContainer {
	x.tx.ZRem(ctx, key, value)

	return x
}

func (x *TxContainer) addInSet(ctx context.Context, key string, value interface{}) *TxContainer {
	x.tx.SAdd(ctx, key, value)

	return x
}

func (x *TxContainer) removeFromSet(ctx context.Context, key string, value interface{}) *TxContainer {
	x.tx.SRem(ctx, key, value)

	return x
}

func (r *RedisClient) addInSet(ctx context.Context, key string, value interface{}) error {
	_, err := r.cli.SAdd(ctx, key, value).Result()

	if err != nil {
		return err
	}

	return nil
}

func (r *RedisClient) removeFromSet(ctx context.Context, key string, value interface{}) error {
	_, err := r.cli.SRem(ctx, key, value).Result()

	if err != nil {
		return err
	}

	return nil
}

func (r *RedisClient) addInHash(ctx context.Context, key string, fieldKey string, fieldValue interface{}) error {
	_, err := r.cli.HSet(ctx, key, fieldKey, fieldValue).Result()

	if err != nil {
		return err
	}

	return nil
}

func (x *TxContainer) addInHash(ctx context.Context, key string, fieldKey string, fieldValue interface{}) *TxContainer {
	x.tx.HSet(ctx, key, fieldKey, fieldValue)

	return x
}

func (r *RedisClient) getFromHash(ctx context.Context, key string, field string) (*string, error) {
	value, err := r.cli.HGet(ctx, key, field).Result()

	if err != nil {
		return nil, err
	}

	return &value, err
}

func (r *RedisClient) removeFromHash(ctx context.Context, key string, field string) error {
	_, err := r.cli.HDel(ctx, key, field).Result()

	if err != nil {
		return err
	}

	return nil
}

func (x *TxContainer) removeFromHash(ctx context.Context, key string, field string) *TxContainer {
	x.tx.HDel(ctx, key, field)

	return x
}

func (r *RedisClient) addInList(ctx context.Context, key string, value interface{}) error {
	_, err := r.cli.LPush(ctx, key, value).Result()

	if err != nil {
		return err
	}

	return nil
}

func (r *RedisClient) getFromList(ctx context.Context, key string) (*string, error) {
	value, err := r.cli.LPop(ctx, key).Result()

	if err != nil {
		return nil, err
	}

	return &value, nil
}

func (r *RedisClient) performTx(ctx context.Context) TxContainer {
	tx := r.cli.TxPipeline()
	return TxContainer{tx: tx}
}

func (x *TxContainer) execTx(ctx context.Context) error {
	_, err := x.tx.Exec(ctx)
	return err
}
