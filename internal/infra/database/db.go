package database

type DB interface {
	CreateBucket(name string) error
	Put(bucketName string, key, value []byte) error
	Get(bucketName string, key []byte) ([]byte, error)
	Delete(bucketName string, key []byte) error
	Update(fn func(tx Tx) error) error
	View(fn func(tx Tx) error) error
	Close() error
}

type Tx interface {
	CreateBucket(name string) (Bucket, error)
	Bucket(name string) Bucket
}

type Bucket interface {
	Put(key []byte, value []byte) error
	Get(key []byte) []byte
	Delete(key []byte) error
	Cursor() Cursor
}

type Cursor interface {
	First() (key []byte, value []byte)
	Last() (key []byte, value []byte)
	Next() (key []byte, value []byte)
	Prev() (key []byte, value []byte)
	Seek(seek []byte) (key []byte, value []byte)
}
