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
	chain           string
	db              database.KVStore
	entryPoints     []common.Address
	rpcUrls         []string
	startBlock      int64
	startBlockDbKey string
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

func NewBackend(eps []string, chain ChainCfg, db database.KVStore) *Backend {
	logger := log.Module("backend")

	var clients []*web3.Web3
	for _, url := range chain.Backends {
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
			if chain.ChainId != chainId {
				logger.Error(fmt.Sprintf("error connect rpc, chain id %s expect %s", chainId, chain.ChainId), "url", url)
				break
			}
			clients = append(clients, cli)
			break
		}
	}

	if len(clients) == 0 {
		panic("backend no available rpc")
	}

	backend := &Backend{
		chain:           chain.Chain,
		db:              db,
		entryPoints:     nil,
		rpcUrls:         chain.Backends,
		startBlock:      chain.StartBlock,
		blockRange:      chain.BlockRangeSize,
		logger:          logger,
		pullingInterval: time.Duration(1000) * time.Millisecond,
		web3Clients:     clients,

		startBlockDbKey: DbKeyStartBlock(chain.Chain),
	}

	for _, ep := range eps {
		backend.entryPoints = append(backend.entryPoints, common.HexToAddress(ep))
	}

	return backend
}

func (b *Backend) LatestBlockNumber() (uint64, *web3.Web3, error) {
	var cli *web3.Web3
	var blockNumberMax uint64
	var blockNumber uint64
	var err error
	for idx, _ := range b.web3Clients {
		blockNumber, err = b.web3Clients[idx].Cli().BlockNumber(context.Background())
		if err != nil {
			b.logger.Error("error get latest block number", "err", err)
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
	val, err := b.db.Get(b.startBlockDbKey)
	if err != nil {
		panic(fmt.Sprintf("error get db key %s: %s", b.startBlockDbKey, err.Error()))
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
	err := b.db.Put(b.startBlockDbKey, next)
	if err != nil {
		panic(fmt.Sprintf("error put db key %s: %s", DbKeyStartBlock, err.Error()))
	}
}

func (b *Backend) Run() error {
	for {
		startTime := time.Now()

		err := func() error {
			latestBlockNumber, cli, err := b.LatestBlockNumber()
			if err != nil {
				return err
			}

			gLatestBlock = int64(latestBlockNumber)

			fromBlock := b.StartBlock()
			toBlock := int64(math.Min(float64(fromBlock+b.blockRange-1), float64(latestBlockNumber)))
			if fromBlock > toBlock {
				//b.logger.Debug(fmt.Sprintf("error block range from > to: %v > %v", fromBlock, toBlock))
				return fmt.Errorf("error block range from > to: %v > %v", fromBlock, toBlock)
			}

			if toBlock-fromBlock < b.blockRange {
				fromBlock = toBlock - b.blockRange + 1
			}

			return b.CallAndSave(fromBlock, toBlock, cli)
		}()

		if err != nil {
			b.logger.Error(err.Error())
		}

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

		b.db.Put(DbKeyUserOp(b.chain, hash), data)
		//nextBlockNumber = int64(ethlog.BlockNumber + 1)
	}

	if len(ethlogs) > 0 {
		b.logger.Info("import logs", "size", len(ethlogs))
	}

	b.SetNextStartBlock(nextBlockNumber)
	return nil
}
