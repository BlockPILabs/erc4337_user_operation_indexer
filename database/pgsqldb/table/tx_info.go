package table

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type TxInfo struct {
	ID          int
	TxHash      string
	BlockNumber uint64
	TxFrom      string
	TxTo        string
	TxLogs      string
	TxValue     string
	Gas         uint64
	GasPrice    string
	GasLimit    string
	Topics      string
	CreateTime  time.Time
}

func NewTxInfo(txHash, txFrom, txTo, txLogs, txValue, gasPrice, gasLimit, topics string, gas, blockNumber uint64) *TxInfo {
	return &TxInfo{
		TxHash:      txHash,
		BlockNumber: blockNumber,
		TxFrom:      txFrom,
		TxTo:        txTo,
		TxLogs:      txLogs,
		TxValue:     txValue,
		Gas:         gas,
		GasPrice:    gasPrice,
		GasLimit:    gasLimit,
		Topics:      topics,
	}
}

func InsertBatchTxInfo(txInfos []*TxInfo) error {

	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=root dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmt := `
		INSERT INTO tx_info (tx_hash, tx_from, tx_to, tx_logs, tx_value, gas, gas_price, gas_limit) 
		VALUES
	`

	valuePlaceholders := ""
	for i := 0; i < len(txInfos); i++ {
		valuePlaceholders += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d),", (i*8)+1, (i*8)+2, (i*8)+3, (i*8)+4, (i*8)+5, (i*8)+6, (i*8)+7, (i*8)+8)
	}
	valuePlaceholders = valuePlaceholders[:len(valuePlaceholders)-1]

	var values []interface{}
	for _, txInfo := range txInfos {
		values = append(values, txInfo.TxHash, txInfo.TxFrom, txInfo.TxTo, txInfo.TxLogs, txInfo.TxValue, txInfo.Gas, txInfo.GasPrice, txInfo.GasLimit)
	}

	fullStmt := stmt + valuePlaceholders

	_, err = db.Exec(fullStmt, values...)
	if err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func (t *TxInfo) Save(db *sql.DB) error {
	stmt, err := db.Prepare("INSERT INTO tx_info (tx_hash, tx_from, tx_to, tx_logs, tx_value, gas, gas_price, gas_limit, block_number) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(t.TxHash, t.TxFrom, t.TxTo, t.TxLogs, t.TxValue, t.Gas, t.GasPrice, t.GasLimit, t.BlockNumber)
	if err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}
