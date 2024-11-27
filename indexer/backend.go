package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/BlockPILabs/erc4337_user_operation_indexer/database"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/databend"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/memorydb"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/pebble"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/redisdb"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/log"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/web3"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/snappy"
	"github.com/spf13/cast"
)

var (
	LogDescriptor = "0x49628fd1471006c1482da88028e9ce4dbb080b815c9b0344d39e5a8e6ec1419f"
	_logTopics    = [][]common.Hash{{common.HexToHash(LogDescriptor)}}

	_httpTimeout      = time.Second * 10
	nexBlockNumberMap = sync.Map{}
	gBlockNumberMap   = sync.Map{}
	gLatestBlockMap   = sync.Map{}
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

	headers map[string]string
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
	case "databend":
		db, err = databend.NewDatabendDB(dataSource)
		if err != nil {
			panic(fmt.Sprintf("error create databend db, %v", err))
		}
	default:
		panic(fmt.Sprintf("Invalid db.engine '%s', allowed 'memory' or 'pebble' or 'redis' or 'databend'", engin))
	}

	return db
}

func parseUrl(str string) (*url.URL, error) {
	if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
		str = "http://" + str
	}

	return url.Parse(str)
}

func NewBackend(headers []HeadersCfg, eps []string, chain ChainCfg, db database.KVStore, compress bool) *Backend {
	logger := log.Module("backend")
	var clients []*web3.Web3
	for _, uri := range chain.Backends {
		url, err := parseUrl(uri)
		if err != nil {
			logger.Error("invalid backend", "url", uri, "err", err, "chain", chain.Chain)
			continue
		}

		cli, err := web3.NewWeb3Client(url.String())
		if err != nil {
			logger.Error("error connect rpc", "url", url, "err", err, "chain", chain.Chain)
			continue
		}

		for _, header := range headers {
			ok, _ := regexp.Match(header.Host, []byte(url.Host))
			if ok {
				headers := map[string]string{}
				for i := 0; i < len(header.Headers); i += 2 {
					headers[header.Headers[i]] = header.Headers[i+1]
				}
				cli.SetHeaders(headers)
			}
		}

		result, err := cli.Cli().ChainID(context.Background())
		if err != nil {
			logger.Error("error connect rpc", "url", url, "err", err, "chain", chain.Chain)
			continue
		}
		chainId := result.String()
		if chain.ChainId != chainId {
			logger.Error(fmt.Sprintf("error connect rpc, chain id %s expect %s", chainId, chain.ChainId), "url", url, "chain", chain.Chain)
			continue
		}
		clients = append(clients, cli)
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
		compress:        compress,
		pullingInterval: time.Millisecond * time.Duration(chain.PullingInterval),
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
			b.logger.Warn("error get latest block number", "err", err)
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
	v, ok := nexBlockNumberMap.Load(b.startBlockDbKey)
	if ok {
		blockNumber := v.(int64)
		return blockNumber
	}

	val, err := b.db.Get(b.startBlockDbKey, false)
	if err != nil {
		panic(fmt.Sprintf("error get db key %s: %s", b.startBlockDbKey, err.Error()))
	}
	if len(val) == 0 {
		return b.startBlock
	}

	blockNumber := int64(math.Max(float64(cast.ToInt64(string(val))-b.blockRange), 0))

	return blockNumber
}

func (b *Backend) SetNextStartBlock(block int64) {
	gBlockNumberMap.Store(b.chain, block)
	next := []byte(fmt.Sprintf("%v", block))
	err := b.db.Put(b.startBlockDbKey, next, false)
	if err != nil {
		panic(fmt.Sprintf("error put db key %s: %s", b.startBlockDbKey, err.Error()))
	}
	nexBlockNumberMap.Store(b.startBlockDbKey, block)
}

func (b *Backend) Run() error {
	for {
		startTime := time.Now()

		err := func() error {
			latestBlockNumber, cli, err := b.LatestBlockNumber()
			if err != nil {
				return err
			}

			gLatestBlockMap.Store(b.chain, int64(latestBlockNumber))

			fromBlock := b.StartBlock()
			if fromBlock == int64(latestBlockNumber) {
				return nil
			}

			toBlock := int64(math.Min(float64(fromBlock+b.blockRange-1), float64(latestBlockNumber)))
			if fromBlock > toBlock {
				//b.logger.Debug(fmt.Sprintf("error block range from > to: %v > %v", fromBlock, toBlock))
				//return fmt.Errorf("error block range from > to: %v > %v", fromBlock, toBlock)
				return nil
			}

			//if toBlock-fromBlock < b.blockRange {
			//	fromBlock = toBlock - b.blockRange + 1
			//}

			return b.CallAndSave(fromBlock, toBlock, cli)
		}()

		if err != nil {
			b.logger.Error(err.Error())
		}
		time.Sleep(b.pullingInterval - time.Since(startTime))
	}
	//return errors.New("backend exited")
}

func (b *Backend) CallAndSave(fromBlock, toBlock int64, cli *web3.Web3) error {
	b.logger.Info(fmt.Sprintf("filter logs range [%v,%v]", fromBlock, toBlock), "url", cli.Url(), "chain", b.chain)

	ctx, cancelFunc := context.WithTimeout(context.Background(), _httpTimeout)
	defer cancelFunc()

	param := ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlock),
		ToBlock:   big.NewInt(toBlock),
		Addresses: b.entryPoints,
		Topics:    _logTopics,
	}
	ethlogs, err := cli.Cli().FilterLogs(ctx, param)
	if err != nil {
		b.logger.Error("error filter logs", "err", err, "url", cli.Url(), "chain", b.chain)
		return err
	}

	nextBlockNumber := toBlock
	for _, ethlog := range ethlogs {
		hash := ethlog.Topics[1].Hex()
		data, _ := json.Marshal(ethlog)

		if b.compress {
			data = snappy.Encode(nil, data)
		}

		b.db.Put(DbKeyUserOp(b.chain, hash), data, b.compress)
		//nextBlockNumber = int64(ethlog.BlockNumber + 1)
	}

	if len(ethlogs) > 0 {
		b.logger.Info("import logs", "size", len(ethlogs), "chain", b.chain)
	}

	b.SetNextStartBlock(nextBlockNumber)
	return nil
}
