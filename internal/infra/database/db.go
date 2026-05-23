package database

type DB interface {
	CreateBucket(name string) error
	Put(bucketName string, key, value []byte) error
	Get(bucketName string, key []byte) ([]byte, error)
	Delete(bucketName string, key []byte) error
	Close() error
}
