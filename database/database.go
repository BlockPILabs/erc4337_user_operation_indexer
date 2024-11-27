package database

type KVStore interface {
	Has(key string) (bool, error)
	Get(key string, compressed bool) ([]byte, error)
	Put(key string, value []byte, compressed bool) error
	Delete(key string) error
}
