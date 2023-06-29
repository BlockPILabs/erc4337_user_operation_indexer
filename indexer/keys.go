package indexer

import "fmt"

var (
	dbKeyPrefix       = ":"
	dbKeyUserOpPrefix = "op:"

	DbKeyStartBlock = dbKeyPrefix + "start-block"
)

func DbKey(key string) string {
	return fmt.Sprintf("%s%s", dbKeyPrefix, key)
}

func DbKeyUserOp(op string) string {
	return fmt.Sprintf("%s%s%s", dbKeyPrefix, dbKeyUserOpPrefix, op)
}
