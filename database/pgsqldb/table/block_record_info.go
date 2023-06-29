package table

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type BlockRecordInfo struct {
	ID              int
	Chain           string
	LastBlockNumber uint64
	CreateTime      time.Time
	UpdateTime      time.Time
}

func NewBlockRecordInfo(chain string, lastBlockNumber uint64) *BlockRecordInfo {
	currentTime := time.Now()
	return &BlockRecordInfo{
		Chain:           chain,
		LastBlockNumber: lastBlockNumber,
		CreateTime:      currentTime,
		UpdateTime:      currentTime,
	}
}

func (b *BlockRecordInfo) Save(db *sql.DB) error {
	stmt, err := db.Prepare("INSERT INTO block_record_info (chain, last_block_number, create_time, update_time) VALUES ($1, $2, $3, $4)")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(b.Chain, b.LastBlockNumber, b.CreateTime, b.UpdateTime)
	if err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func GetBlockRecordInfoByChain(db *sql.DB, chain string) (*BlockRecordInfo, error) {
	stmt, err := db.Prepare("SELECT id, chain, last_block_number, create_time, update_time FROM block_record_info WHERE chain = $1")
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(chain)
	if row == nil {
		return nil, err
	}
	return scanBlockRecordInfo(row)
}

func (b *BlockRecordInfo) UpdateLastBlockNumber(db *sql.DB, lastBlockNumber uint64) error {
	stmt, err := db.Prepare("UPDATE block_record_info SET last_block_number = $1, update_time = $2 WHERE id = $3")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(lastBlockNumber, time.Now(), b.ID)
	if err != nil {
		log.Fatal(err)
		return err
	}

	b.LastBlockNumber = lastBlockNumber
	b.UpdateTime = time.Now()

	return nil
}

func scanBlockRecordInfo(row *sql.Row) (*BlockRecordInfo, error) {
	var id int
	var lastBlockNumber uint64
	var chain string
	var createTime, updateTime time.Time

	var os []BlockRecordInfo
	for true {
		err := row.Scan(&id, &chain, &lastBlockNumber, &createTime, &updateTime)
		if err != nil {
			break
		}
		os = append(os, BlockRecordInfo{
			ID:              id,
			Chain:           chain,
			LastBlockNumber: lastBlockNumber,
			CreateTime:      createTime,
			UpdateTime:      updateTime,
		})

	}
	if len(os) == 0 {
		return nil, nil
	}

	return &os[0], nil
}
