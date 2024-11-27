package databend

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"github.com/BlockPILabs/erc4337_user_operation_indexer/log"
	_ "github.com/datafuselabs/databend-go"
)

type Database struct {
	db *sql.DB

	quitLock sync.Mutex
	quitChan chan chan error

	log log.Logger
}

func NewDatabendDB(dsn string) (*Database, error) {
	conn, err := sql.Open("databend", dsn)
	if err != nil {
		return nil, err
	}
	logger := log.New("databend")

	db := &Database{
		db:       conn,
		log:      logger,
		quitChan: make(chan chan error),
	}
	return db, nil
}

func (d *Database) Close() error {
	d.quitLock.Lock()
	defer d.quitLock.Unlock()

	if d.quitChan == nil {
		return nil
	}
	errc := make(chan error)
	d.quitChan <- errc
	if err := <-errc; err != nil {
		d.log.Error("Metrics collection failed", "err", err)
	}
	d.quitChan = nil

	return d.db.Close()
}

func (d *Database) Has(key string) (bool, error) {
	query := fmt.Sprintf(`SELECT * FROM indexer WHERE key='%s'`, key)
	row := d.db.QueryRow(query)

	var col1, col2 string
	err := row.Scan(&col1, &col2)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *Database) Get(key string, compressed bool) ([]byte, error) {
	query := fmt.Sprintf(`SELECT * FROM indexer WHERE key='%s'`, key)
	row := d.db.QueryRow(query)

	var col1, col2 string
	err := row.Scan(&col1, &col2)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var result []byte
	if compressed {
		result, err = base64.StdEncoding.DecodeString(col2)
		if err != nil {
			result = []byte(col2)
		}
	} else {
		result = []byte(col2)
	}

	return result, nil
}

func (d *Database) Put(key string, value []byte, compressed bool) error {
	var data string
	if compressed {
		data = base64.StdEncoding.EncodeToString(value)
	} else {
		data = string(value)
	}
	query := fmt.Sprintf(`REPLACE INTO indexer ON (key) VALUES ('%s', '%s')`, key, data)
	_, err := d.db.Exec(query)
	return err
}

func (d *Database) Delete(key string) error {
	query := fmt.Sprintf(`DELETE FROM indexer WHERE key='%s'`, key)
	_, err := d.db.Exec(query)
	return err
}
