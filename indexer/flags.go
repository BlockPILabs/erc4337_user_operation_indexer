package indexer

import (
	"encoding/json"
	"github.com/urfave/cli/v2"
)

func _mustMarshal(src any) []byte {
	result, _ := json.Marshal(src)
	return result
}

var (
	FlagVersion = &cli.BoolFlag{
		Name:  "version",
		Usage: "Version",
	}

	FlagConfig = &cli.StringFlag{
		Name:  "config",
		Usage: "config file",
		Value: "",
	}

	FlagListen = &cli.StringFlag{
		Name:  "listen",
		Usage: "listen",
		Value: "127.0.0.1:2052",
	}

	FlagGrpcListen = &cli.StringFlag{
		Name:  "grpc.listen",
		Usage: "grpc.listen",
		Value: "127.0.0.1:2053",
	}

	FlagReadonly = &cli.BoolFlag{
		Name:  "readonly",
		Usage: "readonly",
		Value: false,
	}

	FlagChain = &cli.StringFlag{
		Name:  "chain",
		Usage: "ChainCfg",
		Value: "",
	}

	FlagChainId = &cli.StringFlag{
		Name:  "chain.id",
		Usage: string(_mustMarshal(DefaultChainId)),
		Value: "",
	}

	FlagEntryPoint = &cli.StringFlag{
		Name:  "entrypoint",
		Usage: "Entrypoint contract",
		Value: "0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789",
	}

	FlagBackendUrl = &cli.StringFlag{
		Name:  "backend",
		Usage: "Backends chain rpc provider url",
		Value: "",
	}

	FlagCompress = &cli.BoolFlag{
		Name:  "compress",
		Usage: "compress",
		Value: false,
	}

	FlagDbEngin = &cli.StringFlag{
		Name:  "db.engin",
		Usage: "Backing database implementation to use ('memory' or 'redis' or 'pebble')",
		Value: "memory",
	}

	FlagDbDataSource = &cli.StringFlag{
		Name:  "db.ds",
		Usage: "mysql://user:passwd@hostname:port/databasename, redis://passwd@host:port",
		Value: "",
	}

	FlagEthLogsStartBlock = &cli.Int64Flag{
		Name:  "block.start",
		Usage: string(_mustMarshal(DefaultStartBlocks)),
		Value: 0,
	}

	FlagEthLogsBlockRange = &cli.Int64Flag{
		Name:  "block.range",
		Usage: "eth_getLogs block range",
		Value: 1000,
	}
)
