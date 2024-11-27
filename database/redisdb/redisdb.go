package redisdb

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/redis/go-redis/v9"
)

type Database struct {
	db   *redis.Client
	lock sync.RWMutex
}

func NewDatabase(dataSource string) *Database {
	ds, _ := url.Parse(dataSource)
	passwd, _ := ds.User.Password()

	cli := redis.NewClient(&redis.Options{
		Network:  "tcp",
		Addr:     ds.Host,
		Password: passwd,
		DB:       0,

		PoolSize:     1000,
		MinIdleConns: 10,

		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		PoolTimeout:  5 * time.Second,

		MaxRetries:      0,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,

		OnConnect: func(ctx context.Context, conn *redis.Conn) error {
			log.Debug("new redis conn", "conn", conn)
			return nil
		},
	})
	return &Database{
		db: cli,
	}
}

func (db *Database) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.db.Close()

	db.db = nil
	return nil
}

// Has retrieves if a key is present in the key-value store.
func (db *Database) Has(key string) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	v, err := db.db.Exists(context.Background(), key).Result()
	return v == 1, err
}

// Get retrieves the given key if it's present in the key-value store.
func (db *Database) Get(key string, compressed bool) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	v, err := db.db.Get(context.Background(), key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}

	return v, err
}

// Put inserts the given value into the key-value store.
func (db *Database) Put(key string, value []byte, compressed bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	err := db.db.Set(context.Background(), key, value, 0).Err()

	return err
}

// Delete removes the key from the key-value store.
func (db *Database) Delete(key string) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	err := db.db.Del(context.Background(), key).Err()

	return err
}
