package indexer

import (
	"encoding/json"
	"github.com/urfave/cli/v2"
)

var (
	FlagVersion = &cli.BoolFlag{
		Name:  "version",
		Usage: "Version",
	}

	FlagListen = &cli.StringFlag{
		Name:  "listen",
		Usage: "listen",
		Value: "127.0.0.1:2052",
	}

	FlagChain = &cli.StringFlag{
		Name:  "chain",
		Usage: "Chain",
		Value: "",
	}

	FlagEntryPoint = &cli.StringFlag{
		Name:  "entrypoint",
		Usage: "Entrypoint contract",
		Value: "0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789",
	}

	FlagBackendUrl = &cli.StringFlag{
		Name:  "backend",
		Usage: "Backend chain rpc provider url",
		Value: "",
	}

	FlagDbPrefix = &cli.StringFlag{
		Name:  "db.prefix",
		Usage: "Backing database prefix",
		Value: "",
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

	usage, _              = json.Marshal(DefaultStartBlocks)
	FlagEthLogsStartBlock = &cli.Int64Flag{
		Name:  "block.start",
		Usage: string(usage),
		Value: 0,
	}

	FlagEthLogsBlockRange = &cli.Int64Flag{
		Name:  "block.range",
		Usage: "eth_getLogs block range",
		Value: 1000,
	}
)
