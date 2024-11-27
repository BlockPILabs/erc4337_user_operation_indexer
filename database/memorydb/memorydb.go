package memorydb

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type Database struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func New() *Database {
	return &Database{
		db: map[string][]byte{},
	}
}

func (db *Database) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db = nil
	return nil
}

func (db *Database) Has(key string) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db[key]
	return ok, nil
}

func (db *Database) Get(key string, compressed bool) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[key]; ok {
		return common.CopyBytes(entry), nil
	}

	return nil, nil
}

func (db *Database) Put(key string, value []byte, compressed bool) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[key] = common.CopyBytes(value)
	return nil
}

func (db *Database) Delete(key string) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, key)
	return nil
}
