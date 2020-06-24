package redis

import (
	"github.com/AsimovNetwork/data-sync/library/common"
	"github.com/gomodule/redigo/redis"
	"time"
)

type RedisClient struct{}

type RedisCommand struct {
	Name string
	Args []interface{}
}

var (
	redisPool *redis.Pool
	redisHost string
)

func init() {
	redisHost = common.Cfg.Redis

	redisPool = &redis.Pool{
		MaxIdle:     1,
		MaxActive:   10,
		IdleTimeout: 180 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisHost)
			if err != nil {
				return nil, err
			}

			if _, err := c.Do("AUTH", "bl2a0D9Wy6gP1KUaU-a7"); err != nil {
				c.Close()
				return nil, err
			}
			return c, nil
		},
	}
}

func (redisClient RedisClient) Pipeline(commands ...RedisCommand) error {
	conn := redisPool.Get()
	defer conn.Close()

	var err error
	for _, command := range commands {
		err = conn.Send(command.Name, command.Args...)
		if err != nil {
			return err
		}
	}

	err = conn.Flush()
	if err != nil {
		return nil
	}

	for range commands {
		_, err := conn.Receive()
		if err != nil {
			return err
		}
	}

	return nil
}

func (redisClient RedisClient) Delete(key ...interface{}) error {
	conn := redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", key...)
	return err
}

func (redisClient RedisClient) Set(key string, val int64) error {
	conn := redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("SET", key, val)
	return err
}

func (redisClient RedisClient) BloomFilter(key string, item ...interface{}) error {
	conn := redisPool.Get()
	defer conn.Close()

	args := append([]interface{}{key}, item...)
	_, err := conn.Do("BF.MADD", args...)
	if err != nil {
		return err
	}
	return nil
}

func (redisClient RedisClient) Increase(key string, delta interface{}) error {
	conn := redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("INCRBY", key, delta)
	return err
}

func (redisClient RedisClient) FlushAll() error {
	conn := redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("FLUSHALL")
	return err
}
