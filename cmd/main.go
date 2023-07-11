package main

import (
	"fmt"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/indexer"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/version"
	"os"
	"sort"

	"github.com/urfave/cli/v2"
)

func RootApp() *cli.App {
	app := &cli.App{
		Name:      "ERC4337 user operation indexer",
		Copyright: "Copyright 2023 BlockPI Labs",
		Usage:     "",
		Flags: []cli.Flag{
			indexer.FlagVersion,
			indexer.FlagListen,
			indexer.FlagGrpcListen,
			indexer.FlagChain,
			indexer.FlagChainId,
			indexer.FlagEntryPoint,
			indexer.FlagBackendUrl,
			indexer.FlagCompress,
			indexer.FlagDbPrefix,
			indexer.FlagDbEngin,
			indexer.FlagDbDataSource,
			indexer.FlagEthLogsStartBlock,
			indexer.FlagEthLogsBlockRange,
		},
		EnableBashCompletion: true,
		Before: func(ctx *cli.Context) error {
			return nil
		},
		Action: func(ctx *cli.Context) error {
			if ctx.IsSet(indexer.FlagVersion.Name) {
				fmt.Println(version.VersionFull())
				return nil
			}
			return runApp(ctx)
		},
		Commands: []*cli.Command{
			{
				Name: "version",
				Action: func(ctx *cli.Context) error {
					fmt.Println(version.VersionFull())
					return nil
				},
			},
		},
	}
	return app
}

func main() {
	app := RootApp()
	sort.Sort(cli.CommandsByName(app.Commands))
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
