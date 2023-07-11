package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/memorydb"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/pebble"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/redisdb"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/log"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/web3"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/snappy"
	"math"
	"math/big"
	"strconv"
	"time"
)

var (
	LogDescriptor = "0x49628fd1471006c1482da88028e9ce4dbb080b815c9b0344d39e5a8e6ec1419f"
	_logTopics    = [][]common.Hash{{common.HexToHash(LogDescriptor)}}
	gBlockNumber  int64
	gLatestBlock  int64
)

type Backend struct {
	chainId         string
	db              database.KVStore
	entryPoints     []common.Address
	rpcUrls         []string
	startBlock      int64
	blockRange      int64
	pullingInterval time.Duration

	web3Clients []*web3.Web3

	logger log.Logger

	compress bool
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
	default:
		panic(fmt.Sprintf("Invalid db.engine '%s', allowed 'memory' or 'pebble' or 'redis'", engin))
	}

	return db
}

func NewBackend(cfg *Config) *Backend {
	logger := log.Module("backend")

	var clients []*web3.Web3
	for _, url := range cfg.BackendUrls {
		for n := 0; n < 10; n++ {
			cli, err := web3.NewWeb3Client(url)
			if err != nil {
				logger.Error("error connect rpc", "url", url, "err", err)
				continue
			}

			result, err := cli.Cli().ChainID(context.Background())
			if err != nil {
				logger.Error("error connect rpc", "url", url, "err", err)
				continue
			}
			chainId := result.String()
			if cfg.ChainId != chainId {
				logger.Error(fmt.Sprintf("error connect rpc, chain id %s expect %s", chainId, cfg.ChainId), "url", url)
				break
			}
			clients = append(clients, cli)
			break
		}
	}

	if len(clients) == 0 {
		panic("backend no available rpc")
	}

	return &Backend{
		chainId:         cfg.ChainId,
		db:              NewDb(cfg.DbEngin, cfg.DbDataSource),
		entryPoints:     []common.Address{common.HexToAddress(cfg.EntryPoint)},
		rpcUrls:         cfg.BackendUrls,
		startBlock:      cfg.StartBlock,
		blockRange:      cfg.BlockRangeSize,
		logger:          logger,
		pullingInterval: time.Duration(1000) * time.Millisecond,
		web3Clients:     clients,
	}
}

func (b *Backend) LatestBlockNumber() (uint64, *web3.Web3, error) {
	var cli *web3.Web3
	var blockNumberMax uint64
	var blockNumber uint64
	var err error
	for idx, _ := range b.web3Clients {
		blockNumber, err = b.web3Clients[idx].Cli().BlockNumber(context.Background())
		if err != nil {
			continue
		}
		if blockNumber > blockNumberMax {
			blockNumberMax = blockNumber
			cli = b.web3Clients[idx]
		}
	}

	if cli == nil {
		if err != nil {
			err = errors.New("server error")
		}
	} else {
		err = nil
	}

	return blockNumberMax, cli, err
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
	gBlockNumber = block
	next := []byte(fmt.Sprintf("%v", block))
	err := b.db.Put(DbKeyStartBlock, next)
	if err != nil {
		panic(fmt.Sprintf("error put db key %s: %s", DbKeyStartBlock, err.Error()))
	}
}

func (b *Backend) Run() error {
	for {
		startTime := time.Now()

		latestBlockNumber, cli, err := b.LatestBlockNumber()
		if err != nil {
			continue
		}

		gLatestBlock = int64(latestBlockNumber)

		fromBlock := b.StartBlock()
		toBlock := int64(math.Min(float64(fromBlock+b.blockRange-1), float64(latestBlockNumber)))
		if fromBlock > toBlock {
			//b.logger.Debug(fmt.Sprintf("error block range from > to: %v > %v", fromBlock, toBlock))
			continue
		}

		if toBlock-fromBlock < b.blockRange {
			fromBlock = toBlock - b.blockRange + 1
		}

		b.CallAndSave(fromBlock, toBlock, cli)

		time.Sleep(time.Since(startTime) - b.pullingInterval)
	}
	//return errors.New("backend exited")
}

func (b *Backend) CallAndSave(fromBlock, toBlock int64, cli *web3.Web3) error {
	b.logger.Info(fmt.Sprintf("filter logs range [%v,%v]", fromBlock, toBlock), "url", cli.Url())

	ctx := context.Background()
	param := ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlock),
		ToBlock:   big.NewInt(toBlock),
		Addresses: b.entryPoints,
		Topics:    _logTopics,
	}
	ethlogs, err := cli.Cli().FilterLogs(ctx, param)
	if err != nil {
		b.logger.Error("error filter logs", "err", err, "url", cli.Url())
		return err
	}

	nextBlockNumber := toBlock + 1
	for _, ethlog := range ethlogs {
		hash := ethlog.Topics[1].Hex()
		data, _ := json.Marshal(ethlog)

		if b.compress {
			data = snappy.Encode(nil, data)
		}

		b.db.Put(DbKeyUserOp(hash), data)
		//nextBlockNumber = int64(ethlog.BlockNumber + 1)
	}

	if len(ethlogs) > 0 {
		b.logger.Info("import logs", "size", len(ethlogs))
	}

	b.SetNextStartBlock(nextBlockNumber)
	return nil
}
