package web3

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestEthLogs(t *testing.T) {
	web3, _ := NewWeb3Client("http://127.0.0.1：2052")

	ctx := context.Background()
	param := ethereum.FilterQuery{
		FromBlock: big.NewInt(36617935),
		ToBlock:   big.NewInt(36618707),
		Addresses: []common.Address{common.HexToAddress("0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789")},
		Topics:    [][]common.Hash{{common.HexToHash("0x49628fd1471006c1482da88028e9ce4dbb080b815c9b0344d39e5a8e6ec1419f"), common.HexToHash("0xf9b56a7687db6b03d227b3fced3778af9206f0306c18f445e83eefe426e125fd")}},
	}
	logs, _ := web3.Cli().FilterLogs(ctx, param)
	ret, _ := json.Marshal(logs)

	println(len(logs), string(ret))
}
