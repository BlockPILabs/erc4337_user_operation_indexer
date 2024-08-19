package indexer

import "fmt"

var (
	dbKeyUserOpPrefix = "op"
)

func DbKeyStartBlock(chain string) string {
	dbKey := fmt.Sprintf("start-block:%s", chain)
	return dbKey
}

func DbKeyUserOp(chain, op string) string {
	dbKey := fmt.Sprintf("%s:%s:%s", chain, dbKeyUserOpPrefix, op)
	return dbKey
}
