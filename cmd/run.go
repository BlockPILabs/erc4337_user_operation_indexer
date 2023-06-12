package main

import (
	"errors"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/indexer"
	"github.com/urfave/cli/v2"
)

func runApp(ctx *cli.Context) error {
	if !ctx.IsSet(indexer.FlagBackendUrl.Name) {
		return errors.New("backend url is not set, see --backend")
	}

	cfg := indexer.ParseConfig(ctx)
	return indexer.Run(cfg)
}
