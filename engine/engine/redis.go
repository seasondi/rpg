package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"strings"
	"time"
)

func GetRedisMgr() *redisManager {
	return redisMgr
}

func initRedis() error {
	if GetRedisMgr() == nil {
		redisMgr = new(redisManager)
		return redisMgr.init()
	}
	return nil
}

type redisManager struct {
	clusterClient *redis.ClusterClient
	aloneClient   *redis.Client
}

func (m *redisManager) init() error {
	if GetConfig().Redis == nil {
		return nil
	}
	hosts := strings.Split(cfg.Redis.Hosts, ",")
	if len(hosts) == 0 {
		return errors.New("redis hosts is empty")
	}
	if cfg.Redis.AloneMode == false {
		opts := redis.ClusterOptions{
			Addrs:    hosts,
			Password: cfg.Redis.Password,
		}
		m.clusterClient = redis.NewClusterClient(&opts)
		if ping := m.clusterClient.Ping(context.TODO()); ping.Err() != nil {
			return ping.Err()
		}
		log.Infof("redis inited mode[cluster].")
	} else {
		if cfg.Redis.DB < 0 || cfg.Redis.DB > 15 {
			return fmt.Errorf("invalid redis db: %d, range is [0,15]", cfg.Redis.DB)
		}
		opts := redis.Options{
			Addr:     hosts[0],
			DB:       cfg.Redis.DB,
			Password: cfg.Redis.Password,
		}
		m.aloneClient = redis.NewClient(&opts)
		if ping := m.aloneClient.Ping(context.TODO()); ping.Err() != nil {
			return ping.Err()
		}
		log.Infof("redis inited mode[alone].")
	}
	return nil
}

func (m *redisManager) checkRedis() error {
	if m.clusterClient == nil && m.aloneClient == nil {
		return errors.New("redis not init")
	}
	return nil
}

func (m *redisManager) HGet(ctx context.Context, key string, field string, dst interface{}) error {
	if err := m.checkRedis(); err != nil {
		return err
	}
	var rsp *redis.StringCmd
	if m.clusterClient != nil {
		rsp = m.clusterClient.HGet(ctx, key, field)
	} else {
		rsp = m.aloneClient.HGet(ctx, key, field)
	}
	bytes, err := rsp.Bytes()
	if err == nil {
		err = json.Unmarshal(bytes, dst)
	}
	return err
}

func (m *redisManager) HSet(ctx context.Context, key string, values ...interface{}) error {
	if err := m.checkRedis(); err != nil {
		return err
	}
	if m.clusterClient != nil {
		return m.clusterClient.HSet(ctx, key, values).Err()
	} else {
		return m.aloneClient.HSet(ctx, key, values).Err()
	}
}

func (m *redisManager) HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	if err := m.checkRedis(); err != nil {
		return []interface{}{}, err
	}
	if m.clusterClient != nil {
		return m.clusterClient.HMGet(ctx, key, fields...).Result()
	} else {
		return m.aloneClient.HMGet(ctx, key, fields...).Result()
	}
}

func (m *redisManager) HMSet(ctx context.Context, key string, fields map[string]interface{}) error {
	if err := m.checkRedis(); err != nil {
		return err
	}
	if m.clusterClient != nil {
		return m.clusterClient.HMSet(ctx, key, fields).Err()
	} else {
		return m.aloneClient.HMSet(ctx, key, fields).Err()
	}
}

//HGetAll 即使hash不存在,err也是nil
func (m *redisManager) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if err := m.checkRedis(); err != nil {
		return map[string]string{}, err
	}
	if m.clusterClient != nil {
		return m.clusterClient.HGetAll(ctx, key).Result()
	} else {
		return m.aloneClient.HGetAll(ctx, key).Result()
	}
}

func (m *redisManager) HDel(ctx context.Context, key string, fields ...string) error {
	if err := m.checkRedis(); err != nil {
		return err
	}
	if m.clusterClient != nil {
		return m.clusterClient.HDel(ctx, key, fields...).Err()
	} else {
		return m.aloneClient.HDel(ctx, key, fields...).Err()
	}
}

func (m *redisManager) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := m.checkRedis(); err != nil {
		return err
	}
	if m.clusterClient != nil {
		return m.clusterClient.Set(ctx, key, value, expiration).Err()
	} else {
		return m.aloneClient.Set(ctx, key, value, expiration).Err()
	}
}

func (m *redisManager) Get(ctx context.Context, key string, result interface{}) error {
	if err := m.checkRedis(); err != nil {
		return err
	}
	var r *redis.StringCmd
	if m.clusterClient != nil {
		r = m.clusterClient.Get(ctx, key)
	} else {
		r = m.aloneClient.Get(ctx, key)
	}
	if bytes, err := r.Bytes(); err != nil {
		return err
	} else {
		if err = json.Unmarshal(bytes, result); err != nil {
			return err
		}
	}
	return nil
}

func (m *redisManager) GetBytes(ctx context.Context, key string) ([]byte, error) {
	if err := m.checkRedis(); err != nil {
		return nil, err
	}
	var r *redis.StringCmd
	if m.clusterClient != nil {
		r = m.clusterClient.Get(ctx, key)
	} else {
		r = m.aloneClient.Get(ctx, key)
	}
	return r.Bytes()
}

//MSet要求key必须hash到同一个slot,使用pipeline同时更新多个key
//func (m *redisManager) MSet(pairs ...interface{}) error {
//	if err := m.checkRedis(); err != nil {
//		return err
//	}
//	return m.client.MSet(pairs...).Err()
//}

func (m *redisManager) Del(ctx context.Context, keys ...string) error {
	if err := m.checkRedis(); err != nil {
		return err
	}
	if m.clusterClient != nil {
		return m.clusterClient.Del(ctx, keys...).Err()
	} else {
		return m.aloneClient.Del(ctx, keys...).Err()
	}
}

func (m *redisManager) BRPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	if err := m.checkRedis(); err != nil {
		return []string{}, err
	}
	if m.clusterClient != nil {
		return m.clusterClient.BRPop(ctx, timeout, keys...).Result()
	} else {
		return m.aloneClient.BRPop(ctx, timeout, keys...).Result()
	}
}

func (m *redisManager) LPush(ctx context.Context, key string, values ...interface{}) error {
	if err := m.checkRedis(); err != nil {
		return err
	}
	if m.clusterClient != nil {
		return m.clusterClient.LPush(ctx, key, values...).Err()
	} else {
		return m.aloneClient.LPush(ctx, key, values...).Err()
	}
}

func (m *redisManager) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	if err := m.checkRedis(); err != nil {
		return []string{}, err
	}
	if m.clusterClient != nil {
		return m.clusterClient.LRange(ctx, key, start, stop).Result()
	} else {
		return m.aloneClient.LRange(ctx, key, start, stop).Result()
	}
}

func (m *redisManager) LTrim(ctx context.Context, key string, start, stop int64) error {
	if err := m.checkRedis(); err != nil {
		return err
	}
	if m.clusterClient != nil {
		return m.clusterClient.LTrim(ctx, key, start, stop).Err()
	} else {
		return m.aloneClient.LTrim(ctx, key, start, stop).Err()
	}
}

func (m *redisManager) Pipeline() (redis.Pipeliner, error) {
	if err := m.checkRedis(); err != nil {
		return nil, err
	}
	var pipe redis.Pipeliner
	if m.clusterClient != nil {
		pipe = m.clusterClient.Pipeline()
	} else {
		pipe = m.aloneClient.Pipeline()
	}
	return pipe, nil
}

func (m *redisManager) SetNX(ctx context.Context, key string, val interface{}, expire time.Duration) (bool, error) {
	if err := m.checkRedis(); err != nil {
		return false, err
	}
	if m.clusterClient != nil {
		return m.clusterClient.SetNX(ctx, key, val, expire).Result()
	} else {
		return m.aloneClient.SetNX(ctx, key, val, expire).Result()
	}
}

// --------------------------------------------------------------------------------------------------------------------------------------------------
