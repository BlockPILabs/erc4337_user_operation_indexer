package indexer

import (
	"errors"
	"github.com/urfave/cli/v2"
	"math"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	DefaultEntryPoints = []string{"0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789"}

	DefaultChainId = map[string]string{
		"ethereum": "1",
		"goerli":   "5",
		"sepolia":  "11155111",

		"polygon":        "137",
		"polygon-mumbai": "80001",

		"optimism":        "10",
		"optimism-goerli": "420",

		"arbitrum":        "421613",
		"arbitrum-one":    "42161",
		"arbitrum-nova":   "42170",
		"arbitrum-goerli": "421613",
		"bsc":             "56",
	}

	DefaultStartBlocks = map[string]int64{
		"ethereum": 17066994,
		"goerli":   8812127,
		"sepolia":  3296058,

		"polygon":        41402415,
		"polygon-mumbai": 34239265,

		"optimism":        93335977,
		"optimism-goerli": 10442160,

		"arbitrum":        79305493,
		"arbitrum-nova":   8945015,
		"arbitrum-goerli": 17068300,
		"bsc":             27251985,
	}
)

type Config struct {
	EntryPoints []string `yaml:"entryPoints"`
	Listen      string
	GrpcListen  string `yaml:"grpcListen"`
	Compress    bool
	Db          DBCfg
	Chains      []ChainCfg
}

type DBCfg struct {
	Engin string
	Ds    string
}

type ChainCfg struct {
	Chain          string
	ChainId        string `yaml:"chainId"`
	Backends       []string
	StartBlock     int64 `yaml:"startBlock"`
	BlockRangeSize int64 `yaml:"blockRangeSize"`
}

func ParseConfigFromCmd(ctx *cli.Context) (*Config, error) {
	if !ctx.IsSet(FlagBackendUrl.Name) {
		return nil, errors.New("backend url is not set, see --backend")
	}
	if !ctx.IsSet(FlagChain.Name) {
		return nil, errors.New("chain is not set, see --chain")
	}

	chain := ctx.String(FlagChain.Name)
	chainId := ""
	if ctx.IsSet(FlagChainId.Name) {
		chainId = ctx.String(FlagChainId.Name)
	} else {
		chainId = DefaultChainId[chain]
	}

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

	blockRange := int64(math.Max(math.Min(5000, float64(ctx.Int64(FlagEthLogsBlockRange.Name))), 1))
	cfg := &Config{
		Listen:     ctx.String(FlagListen.Name),
		GrpcListen: ctx.String(FlagGrpcListen.Name),
		Chains: []ChainCfg{{
			Chain:          chain,
			ChainId:        chainId,
			Backends:       strings.Split(ctx.String(FlagBackendUrl.Name), ","),
			StartBlock:     startBlock,
			BlockRangeSize: blockRange,
		}},
		Db: DBCfg{
			Engin: dbEngin,
			Ds:    dataSource,
		},
		EntryPoints: []string{strings.ToLower(ctx.String(FlagEntryPoint.Name))},
		Compress:    ctx.Bool(FlagCompress.Name),
	}
	return cfg, nil
}

func ParseConfigFromFile(ctx *cli.Context) (*Config, error) {
	if !ctx.IsSet(FlagConfig.Name) {
		return nil, nil
	}
	configFile := ctx.String(FlagConfig.Name)
	content, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	var cfg *Config
	err = yaml.Unmarshal(content, &cfg)
	if err != nil {
		panic(err)
	}

	return cfg, err
}

func ParseConfig(ctx *cli.Context) *Config {
	cfgFile, _ := ParseConfigFromFile(ctx)
	cfgCmd, _ := ParseConfigFromCmd(ctx)
	if cfgFile == nil {
		cfgFile = cfgCmd
	} else {
		for idx, _ := range cfgFile.Chains {
			if cfgFile.Chains[idx].StartBlock <= 0 {
				cfgFile.Chains[idx].StartBlock = DefaultStartBlocks[cfgFile.Chains[idx].Chain]
			}

			if cfgFile.Chains[idx].BlockRangeSize <= 0 {
				cfgFile.Chains[idx].BlockRangeSize = 1000
			}

			cfgFile.Chains[idx].BlockRangeSize = int64(math.Min(5000, float64(cfgFile.Chains[idx].BlockRangeSize)))
		}

		if cfgCmd != nil {
			if ctx.IsSet(FlagListen.Name) {
				cfgFile.Listen = cfgCmd.Listen
			}
			if ctx.IsSet(FlagGrpcListen.Name) {
				cfgFile.GrpcListen = cfgCmd.GrpcListen
			}
			if ctx.IsSet(FlagCompress.Name) {
				cfgFile.Compress = cfgCmd.Compress
			}
			if ctx.IsSet(FlagDbEngin.Name) {
				cfgFile.Db.Engin = cfgCmd.Db.Engin
			}
			if ctx.IsSet(FlagDbDataSource.Name) {
				cfgFile.Db.Ds = cfgCmd.Db.Ds
			}
		}
	}

	if len(cfgFile.EntryPoints) == 0 {
		cfgFile.EntryPoints = DefaultEntryPoints
	}

	return cfgFile
}
