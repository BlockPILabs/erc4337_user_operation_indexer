package web3

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/pgsqldb"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/pgsqldb/consts"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/pgsqldb/table"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"strings"
	"time"
)

const Erc4337Contract = "0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789"

func StartBlockTask() {
	timer := time.NewTicker(30 * time.Second)

	go func() {
		ScanBlock()
		for range timer.C {
			ScanBlock()
		}
	}()

}

func TestFilter() {
	client, err := ethclient.Dial("https://patient-crimson-slug.matic.discover.quiknode.pro/4cb47dc694ccf2998581548feed08af5477aa84b/")
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	param := ethereum.FilterQuery{
		FromBlock: big.NewInt(42689937),
		ToBlock:   big.NewInt(42689939),
		Addresses: []common.Address{common.HexToAddress("0x5ff137d4b0fdcd49dca30c7cf57e578a026d2789")},
		Topics:    [][]common.Hash{{common.HexToHash("0x49628fd1471006c1482da88028e9ce4dbb080b815c9b0344d39e5a8e6ec1419f")}},
	}
	ethlogs, err := client.FilterLogs(ctx, param)
	if err != nil {
		log.Fatal(err)
	}
	for _, ethlog := range ethlogs {
		hash := ethlog.Topics[1].Hex()
		data, _ := json.Marshal(ethlog)
		fmt.Printf("Hash: %s\n", hash)
		fmt.Printf("Data: %s\n", data)
		//nextBlockNumber = int64(ethlog.BlockNumber + 1)
	}

}

func ScanBlock() {
	client, err := ethclient.Dial("https://polygon-mumbai.blockpi.network/v1/rpc/6327e51cf103758afd568c6dac3631f235ec5c22")
	if err != nil {
		log.Fatal(err)
	}

	db, err := pgsqldb.GetDBConnection()
	if err != nil {
		log.Fatal(err)
		return
	}
	blockRecordInfo, _ := table.GetBlockRecordInfoByChain(db, consts.Polygon)
	var blockNumber uint64
	if blockRecordInfo == nil {
		header, err := client.HeaderByNumber(context.Background(), nil)
		if err != nil {
			log.Fatal(err)
		}
		blockNumber = header.Number.Uint64()
	} else {
		blockNumber = blockRecordInfo.LastBlockNumber
		blockNumber++
	}

	txInfos := GetTxInfos(big.NewInt(int64(blockNumber)), client)
	if len(txInfos) > 0 {
		table.InsertBatchTxInfo(txInfos)
	}

	if blockRecordInfo == nil {
		blockRecordInfo = table.NewBlockRecordInfo(consts.Polygon, blockNumber)
		blockRecordInfo.Save(db)
	} else {
		blockRecordInfo.UpdateLastBlockNumber(db, blockNumber)
	}

}

func GetTxInfos(blockNumber *big.Int, client *ethclient.Client) []*table.TxInfo {

	var txInfos []*table.TxInfo
	block, err := client.BlockByNumber(context.Background(), blockNumber)
	if err != nil {
		log.Fatal(err)
	}
	for _, tx := range block.Transactions() {
		//fmt.Printf("Tx Hash: %s\n", tx.Hash().Hex())
		//fmt.Printf("Tx Value: %s\n", tx.Value().String())
		//fmt.Printf("Tx Nonce: %d\n", tx.Nonce())
		//fmt.Printf("Tx Data: %s\n", hex.EncodeToString(tx.Data()))
		if tx.To() == nil {
			continue
		}
		if !strings.EqualFold(tx.To().String(), Erc4337Contract) {
			//continue
		}
		//receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
		//if err != nil {
		//	log.Fatal(err)
		//	continue
		//}
		//receipts := receipt.Logs
		//for _, receipt := range receipts {
		txInfo := table.NewTxInfo(tx.Hash().String(), "", tx.To().String(), "",
			tx.Value().String(), tx.GasPrice().String(), "", "", tx.Gas(), blockNumber.Uint64())
		txInfos = append(txInfos, txInfo)
		//}
	}

	fmt.Printf("txInfoSize: %d\n ", len(txInfos))

	return txInfos
}
