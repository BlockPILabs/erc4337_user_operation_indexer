package pebble

import (
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"runtime"
	"sync"
)

const (
	minCache   = 16
	minHandles = 16
)

type Database struct {
	fn string     // filename for reporting
	db *pebble.DB // Underlying pebble storage engine

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database

	log log.Logger // Contextual logger tracking the database path
}

// NewPebbleDb returns a wrapped pebble DB object. The namespace is the prefix that the
// metrics reporting should use for surfacing internal stats.
func NewPebbleDb(file string, cache int, handles int, readonly bool) (*Database, error) {
	// Ensure we have some minimal caching and file guarantees
	if cache < minCache {
		cache = minCache
	}
	if handles < minHandles {
		handles = minHandles
	}
	logger := log.New("database", file)
	logger.Info("Allocated cache and file handles", "cache", common.StorageSize(cache*1024*1024), "handles", handles)

	// The max memtable size is limited by the uint32 offsets stored in
	// internal/arenaskl.node, DeferredBatchOp, and flushableBatchEntry.
	// Taken from https://github.com/cockroachdb/pebble/blob/master/open.go#L38
	maxMemTableSize := 4<<30 - 1 // Capped by 4 GB

	// Two memory tables is configured which is identical to leveldb,
	// including a frozen memory table and another live one.
	memTableLimit := 2
	memTableSize := cache * 1024 * 1024 / 2 / memTableLimit
	if memTableSize > maxMemTableSize {
		memTableSize = maxMemTableSize
	}
	db := &Database{
		fn:       file,
		log:      logger,
		quitChan: make(chan chan error),
	}
	opt := &pebble.Options{
		// Pebble has a single combined cache area and the write
		// buffers are taken from this too. Assign all available
		// memory allowance for cache.
		Cache:        pebble.NewCache(int64(cache * 1024 * 1024)),
		MaxOpenFiles: handles,

		// The size of memory table(as well as the write buffer).
		// Note, there may have more than two memory tables in the system.
		MemTableSize: memTableSize,

		// MemTableStopWritesThreshold places a hard limit on the size
		// of the existent MemTables(including the frozen one).
		// Note, this must be the number of tables not the size of all memtables
		// according to https://github.com/cockroachdb/pebble/blob/master/options.go#L738-L742
		// and to https://github.com/cockroachdb/pebble/blob/master/db.go#L1892-L1903.
		MemTableStopWritesThreshold: memTableLimit,

		// The default compaction concurrency(1 thread),
		// Here use all available CPUs for faster compaction.
		MaxConcurrentCompactions: func() int { return runtime.NumCPU() },

		// Per-level options. Options for at least one level must be specified. The
		// options for the last level are used for all subsequent levels.
		Levels: []pebble.LevelOptions{
			{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
		},
		ReadOnly: readonly,
	}
	// Disable seek compaction explicitly. Check https://github.com/ethereum/go-ethereum/pull/20130
	// for more details.
	opt.Experimental.ReadSamplingMultiplier = -1

	// Open the db and recover any potential corruptions
	innerDB, err := pebble.Open(file, opt)
	if err != nil {
		return nil, err
	}
	db.db = innerDB

	return db, nil
}

func (d *Database) Close() error {
	d.quitLock.Lock()
	defer d.quitLock.Unlock()

	if d.quitChan == nil {
		return nil
	}
	errc := make(chan error)
	d.quitChan <- errc
	if err := <-errc; err != nil {
		d.log.Error("Metrics collection failed", "err", err)
	}
	d.quitChan = nil

	return d.db.Close()
}

func (d *Database) Has(key string) (bool, error) {
	_, closer, err := d.db.Get([]byte(key))

	if err == pebble.ErrNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	closer.Close()
	return true, nil
}

func (d *Database) Get(key string) ([]byte, error) {
	dat, closer, err := d.db.Get([]byte(key))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	ret := make([]byte, len(dat))
	copy(ret, dat)
	closer.Close()
	return ret, nil
}

func (d *Database) Put(key string, value []byte) error {
	return d.db.Set([]byte(key), value, pebble.NoSync)
}

func (d *Database) Delete(key string) error {
	return d.db.Delete([]byte(key), nil)
}
