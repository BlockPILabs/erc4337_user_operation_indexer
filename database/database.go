package database

type KVStore interface {
	Has(key string) (bool, error)
	Get(key string) ([]byte, error)
	Put(key string, value []byte) error
	Delete(key string) error
}
