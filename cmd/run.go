package main

import (
	"github.com/BlockPILabs/erc4337_user_operation_indexer/indexer"
	"github.com/urfave/cli/v2"
)

func runApp(ctx *cli.Context) error {

	cfg := indexer.ParseConfig(ctx)

	return indexer.Run(cfg)
}
