package table

import (
	"database/sql"
	"fmt"
	"github.com/BlockPILabs/erc4337_user_operation_indexer/log"
)

type OperationInfo struct {
	UserOpHash    string
	Sender        string
	Paymaster     string
	Nonce         uint64
	Success       uint64
	ActualGasCost uint64
	ActualGasUsed uint64
	TxHash        string
	BlockNumber   uint64
}

func NewOperationInfo(userOpHash string, sender string, paymaster string, nonce uint64, success uint64, actualGasCost uint64, actualGasUsed uint64, txHash string, blockNumber uint64) *OperationInfo {
	return &OperationInfo{
		UserOpHash:    userOpHash,
		Sender:        sender,
		Paymaster:     paymaster,
		Nonce:         nonce,
		Success:       success,
		ActualGasCost: actualGasCost,
		ActualGasUsed: actualGasUsed,
		TxHash:        txHash,
		BlockNumber:   blockNumber,
	}
}

func GetOperationInfoByUserOpHash(userOpHash string, db *sql.DB) (*OperationInfo, error) {
	query := "SELECT user_op_hash, sender, paymaster, nonce, success, actual_gas_cost, actual_gas_used, tx_hash, block_number FROM operation_info WHERE user_op_hash = $1"
	row := db.QueryRow(query, userOpHash)
	if row == nil {
		return nil, nil
	}

	if row.Err() != nil {
		log.Info("err")
	}

	var os []OperationInfo
	for true {
		operation := OperationInfo{}
		err := row.Scan(
			&operation.UserOpHash,
			&operation.Sender,
			&operation.Paymaster,
			&operation.Nonce,
			&operation.Success,
			&operation.ActualGasCost,
			&operation.ActualGasUsed,
			&operation.TxHash,
			&operation.BlockNumber,
		)
		if err != nil {
			break
		}
		os = append(os, operation)
	}

	if len(os) == 0 {
		return nil, nil
	}

	return &os[0], nil
}

func UpdateOperationInfoByUserOpHash(operation *OperationInfo, db *sql.DB) error {
	query := "UPDATE operation_info SET sender = $1, paymaster = $2, nonce = $3, success = $4, actual_gas_cost = $5, actual_gas_used = $6 WHERE user_op_hash = $7"
	_, err := db.Exec(
		query,
		operation.Sender,
		operation.Paymaster,
		operation.Nonce,
		operation.Success,
		operation.ActualGasCost,
		operation.ActualGasUsed,
		operation.UserOpHash,
	)
	if err != nil {
		return err
	}

	return nil
}

func DeleteOperationInfoByUserOpHash(userOpHash string, db *sql.DB) error {
	query := "DELETE FROM operation_info WHERE user_op_hash = $1"
	_, err := db.Exec(query, userOpHash)
	if err != nil {
		return err
	}

	return nil
}

func (o *OperationInfo) String() string {
	return fmt.Sprintf("UserOpHash: %s\n"+
		"Sender: %s\n"+
		"Paymaster: %s\n"+
		"Nonce: %d\n"+
		"Success: %t\n"+
		"ActualGasCost: %d\n"+
		"ActualGasUsed: %d\n"+
		"TxHash: %s\n"+
		"BlockNumber: %d",
		o.UserOpHash, o.Sender, o.Paymaster, o.Nonce, o.Success, o.ActualGasCost, o.ActualGasUsed, o.TxHash, o.BlockNumber)
}

func InsertOperationInfo(operation *OperationInfo, db *sql.DB) error {
	query := "INSERT INTO operation_info (user_op_hash, sender, paymaster, nonce, success, actual_gas_cost, actual_gas_used, tx_hash, block_number) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"
	_, err := db.Exec(
		query,
		operation.UserOpHash,
		operation.Sender,
		operation.Paymaster,
		operation.Nonce,
		operation.Success,
		operation.ActualGasCost,
		operation.ActualGasUsed,
		operation.TxHash,
		operation.BlockNumber,
	)
	if err != nil {
		return err
	}

	return nil
}
