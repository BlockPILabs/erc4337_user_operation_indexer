package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/memorydb"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/pebble"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/pgsqldb"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/redisdb"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/log"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/web3"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math"
	"math/big"
	"strconv"
	"time"
)

var (
	_logDescriptor = "0x49628fd1471006c1482da88028e9ce4dbb080b815c9b0344d39e5a8e6ec1419f"
	_logTopics     = [][]common.Hash{{common.HexToHash(_logDescriptor)}}
)

type Backend struct {
	db              database.KVStore
	EntryPoints     []common.Address
	rpcUrl          string
	startBlock      int64
	blockRange      int64
	PullingInterval time.Duration

	web3Cli *web3.Web3

	logger log.Logger
}

func NewDb(engin, dataSource string) database.KVStore {
	var db database.KVStore
	var err error
	switch engin {
	case "memory":
		db = memorydb.New()
	case "redis":
		db = redisdb.NewDatabase(dataSource)
	case "pebble":
		db, err = pebble.NewPebbleDb(dataSource, 16, 16, false)
		if err != nil {
			panic(fmt.Sprintf("error create pebble db, %v", err))
		}
	case "postgres":
		db = pgsqldb.NewDataBase(dataSource)
	default:
		panic(fmt.Sprintf("Invalid db.engine '%s', allowed 'memory' or 'pebble' or 'redis'", engin))
	}

	return db
}

func NewBackend(cfg *Config) *Backend {
	web3Cli, _ := web3.NewWeb3Client(cfg.BackendUrl)
	return &Backend{
		db:              NewDb(cfg.DbEngin, cfg.DbDataSource),
		EntryPoints:     []common.Address{common.HexToAddress(cfg.EntryPoint)},
		rpcUrl:          cfg.BackendUrl,
		startBlock:      cfg.StartBlock,
		blockRange:      cfg.BlockRangeSize,
		logger:          log.Module("backend"),
		PullingInterval: time.Duration(1000) * time.Millisecond,
		web3Cli:         web3Cli,
	}
}

func (b *Backend) LatestBlockNumber() (uint64, error) {
	blockNumber, err := b.web3Cli.Cli().BlockNumber(context.Background())
	if err != nil {
		return 0, err
	}
	return blockNumber, nil
}

func (b *Backend) StartBlock() int64 {
	val, err := b.db.Get(DbKeyStartBlock)
	if err != nil {
		panic(fmt.Sprintf("error get db key %s: %s", DbKeyStartBlock, err.Error()))
	}
	if len(val) == 0 {
		return b.startBlock
	}

	blockNumber, _ := strconv.ParseInt(string(val), 10, 64)
	return blockNumber
}

func (b *Backend) SetNextStartBlock(block int64) {
	next := []byte(fmt.Sprintf("%v", block))
	err := b.db.Put(DbKeyStartBlock, next)
	if err != nil {
		panic(fmt.Sprintf("error put db key %s: %s", DbKeyStartBlock, err.Error()))
	}
}

func (b *Backend) Run() error {
	for {
		startTime := time.Now()

		latestBlockNumber, err := b.LatestBlockNumber()
		if err != nil {
			continue
		}
		fromBlock := b.StartBlock()
		toBlock := int64(math.Min(float64(fromBlock+b.blockRange-1), float64(latestBlockNumber)))
		if fromBlock > toBlock {
			//b.logger.Debug(fmt.Sprintf("error block range from > to: %v > %v", fromBlock, toBlock))
			continue
		}

		if toBlock-fromBlock < b.blockRange {
			fromBlock = toBlock - b.blockRange + 1
		}

		b.CallAndSave(fromBlock, toBlock)

		time.Sleep(time.Since(startTime) - b.PullingInterval)
	}
	//return errors.New("backend exited")
}

func (b *Backend) CallAndSave(fromBlock, toBlock int64) error {
	b.logger.Info(fmt.Sprintf("filter logs range [%v,%v]", fromBlock, toBlock))

	ctx := context.Background()
	param := ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlock),
		ToBlock:   big.NewInt(toBlock),
		Addresses: b.EntryPoints,
		Topics:    _logTopics,
	}
	ethlogs, err := b.web3Cli.Cli().FilterLogs(ctx, param)
	if err != nil {
		b.logger.Error("error filter logs", "err", err)
		return err
	}

	nextBlockNumber := toBlock + 1
	for _, ethlog := range ethlogs {
		hash := ethlog.Topics[1].Hex()
		data, _ := json.Marshal(ethlog)
		b.db.Put(DbKeyUserOp(hash), data)
		//nextBlockNumber = int64(ethlog.BlockNumber + 1)
	}

	if len(ethlogs) > 0 {
		b.logger.Info("import logs", "size", len(ethlogs))
	}

	b.SetNextStartBlock(nextBlockNumber)
	return nil
}
