package indexer

import (
	"github.com/urfave/cli/v2"
	"math"
	"strings"
)

var (
	DefaultStartBlocks = map[string]int64{
		"ethereum":         17066994,
		"ethereum-goerli":  8812127,
		"ethereum-sepolia": 3296058,

		"polygon":        41402415,
		"polygon-mumbai": 34239265,

		"optimism":        93335977,
		"optimism-goerli": 10442160,

		"arbitrum":        79305493,
		"arbitrum-nova":   8945015,
		"arbitrum-goerli": 17068300,
	}
)

type Config struct {
	RpcListen      string
	GrpcListen     string
	EntryPoint     string
	BackendUrl     string
	DbEngin        string
	DbDataSource   string
	StartBlock     int64
	BlockRangeSize int64
}

func ParseConfig(ctx *cli.Context) *Config {
	chain := ctx.String(FlagChain.Name)

	var startBlock int64
	if ctx.IsSet(FlagEthLogsStartBlock.Name) {
		startBlock = ctx.Int64(FlagEthLogsStartBlock.Name)
	} else {
		startBlock = DefaultStartBlocks[chain]
	}

	dbEngin := ctx.String(FlagDbEngin.Name)
	dataSource := ctx.String(FlagDbDataSource.Name)
	if (dbEngin == "pebble" && !ctx.IsSet(FlagDbDataSource.Name)) || len(dataSource) == 0 {
		dataSource = "data/db"
	}

	dbKeyPrefix = ctx.String(FlagDbPrefix.Name)
	if len(dbKeyPrefix) == 0 {
		dbKeyPrefix = chain
	}

	blockRange := int64(math.Max(math.Min(5000, float64(ctx.Int64(FlagEthLogsBlockRange.Name))), 1))
	cfg := &Config{
		RpcListen:      ctx.String(FlagListen.Name),
		GrpcListen:     ctx.String(FlagGrpcListen.Name),
		BackendUrl:     ctx.String(FlagBackendUrl.Name),
		DbEngin:        dbEngin,
		DbDataSource:   dataSource,
		EntryPoint:     strings.ToLower(ctx.String(FlagEntryPoint.Name)),
		StartBlock:     startBlock,
		BlockRangeSize: blockRange,
	}
	return cfg
}
