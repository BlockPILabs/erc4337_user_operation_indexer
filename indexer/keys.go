package indexer

import "fmt"

var (
	dbKeyPrefix       = ""
	dbKeyUserOpPrefix = "op"

	DbKeyStartBlock = dbKeyPrefix + "start-block"
)

func DbKey(key string) string {
	dbKey := fmt.Sprintf("%s:%s", dbKeyPrefix, key)
	return dbKey
}

func DbKeyUserOp(op string) string {
	dbKey := fmt.Sprintf("%s:%s:%s", dbKeyPrefix, dbKeyUserOpPrefix, op)
	return dbKey
}
