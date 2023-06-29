package pgsqldb

import (
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/pgsqldb/consts"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/database/pgsqldb/table"
	"github.com/ethereum/go-ethereum/core/types"
	_ "github.com/lib/pq"
	"log"
	"math/big"
)

const StartBlock = ":start-block"

type Database struct {
	cli *sql.DB
}

func NewDataBase(datasource string) *Database {
	dbCli, err := GetDBCli(datasource)
	if err != nil {
		panic(err)
	}
	return &Database{
		cli: dbCli,
	}
}

func (db *Database) Has(key string) (bool, error) {
	if isBlock(key) {
		blockRecordInfo, err := table.GetBlockRecordInfoByChain(db.cli, consts.Polygon)
		if err != nil {
			return false, err
		}
		return blockRecordInfo != nil, nil
	} else {
		operationInfo, err := table.GetOperationInfoByUserOpHash(key, db.cli)
		if err != nil {
			return false, nil
		}
		if operationInfo == nil {
			return false, nil
		}

		return operationInfo == nil, nil
	}

}

func (db *Database) Get(key string) ([]byte, error) {

	if isBlock(key) {
		blockRecordInfo, err := table.GetBlockRecordInfoByChain(db.cli, consts.Polygon)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		if blockRecordInfo == nil {
			return nil, nil
		}
		byteArray := make([]byte, 8)
		binary.BigEndian.PutUint64(byteArray, blockRecordInfo.LastBlockNumber)

		return byteArray, nil
	} else {
		operationInfo, err := table.GetOperationInfoByUserOpHash(key, db.cli)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		if operationInfo == nil {
			return nil, nil
		}
		return []byte(operationInfo.String()), nil
	}

}

func (db *Database) Put(key string, value []byte) error {
	b8 := [8]byte{}
	copy(b8[:], value)

	if isBlock(key) {
		blockRecordInfo, err := table.GetBlockRecordInfoByChain(db.cli, consts.Polygon)
		if err != nil {
			return err
		}
		if blockRecordInfo != nil {
			blockRecordInfo.UpdateLastBlockNumber(db.cli, binary.BigEndian.Uint64(b8[:]))
		} else {
			blockRecordInfo = table.NewBlockRecordInfo(consts.Polygon, binary.BigEndian.Uint64(b8[:]))
			err := blockRecordInfo.Save(db.cli)
			if err != nil {
				return err
			}
		}
		return nil
	} else {
		var ethLog types.Log
		err := json.Unmarshal(value, &ethLog)
		if err != nil {
			return err
		}
		topics := ethLog.Topics
		dataStr := hex.EncodeToString(ethLog.Data)

		slices := splitString(dataStr, 64)
		var decimals []*big.Int

		for _, slice := range slices {
			decimal := hexToDecimal(slice)
			decimals = append(decimals, decimal)
		}
		operationInfo := table.NewOperationInfo(topics[1].String(), topics[2].String(), topics[3].String(), decimals[0].Uint64(),
			decimals[1].Uint64(), decimals[2].Uint64(), decimals[3].Uint64(), ethLog.TxHash.String(), ethLog.BlockNumber)
		table.InsertOperationInfo(operationInfo, db.cli)
		return nil
	}

}

func hexToDecimal(hex string) *big.Int {
	decimal := new(big.Int)
	decimal.SetString(hex, 16)
	return decimal
}

func splitString(str string, length int) []string {
	var slices []string
	runes := []rune(str)

	for i := 0; i < len(runes); i += length {
		end := i + length
		if end > len(runes) {
			end = len(runes)
		}
		slice := string(runes[i:end])
		slices = append(slices, slice)
	}

	return slices
}

func (db *Database) Delete(key string) error {

	if isBlock(key) {
		return nil
	} else {
		table.DeleteOperationInfoByUserOpHash(key, db.cli)
		return nil
	}

}

func isBlock(key string) bool {
	return key == StartBlock
}

func GetDBConnection() (*sql.DB, error) {
	connStr := "user=postgres password=root dbname=postgres host=127.0.0.1 port=5432 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func GetDBCli(datasource string) (*sql.DB, error) {
	db, err := sql.Open("postgres", datasource)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}
