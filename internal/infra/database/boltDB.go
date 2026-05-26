package database

import (
	bolt "go.etcd.io/bbolt"
)

type boltDB struct {
	client *bolt.DB
}

type Transaction struct {
	rawTx *bolt.Tx
}

type boltBucket struct {
	rawBucket *bolt.Bucket
}

type boltCursor struct {
	rawCursor *bolt.Cursor
}

func NewDB(path string) (DB, error) {
	client, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &boltDB{client: client}, nil
}

func (db *boltDB) CreateBucket(name string) error {
	return db.client.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(name))
		return err
	})
}

func (db *boltDB) Put(bucketName string, key, value []byte) error {
	return db.client.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Put(key, value)
		return err
	})
}

func (db *boltDB) Get(bucketName string, key []byte) ([]byte, error) {
	var result []byte
	err := db.client.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return ErrBucketNotFound
		}
		data := b.Get(key)

		//for avoiding
		if data != nil {
			result = make([]byte, len(data))
			copy(result, data)
		}

		return nil
	})
	return result, err
}

func (db *boltDB) Delete(bucketName string, key []byte) error {
	return db.client.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Delete(key)
		return err
	})
}

func (db *boltDB) Update(fn func(tx Tx) error) error {
	return db.client.Update(func(rawTX *bolt.Tx) error {
		wrapedTx := &Transaction{rawTx: rawTX}
		return fn(wrapedTx)
	})
}

func (db *boltDB) View(fn func(tx Tx) error) error {
	return db.client.View(func(rawTX *bolt.Tx) error {
		wrapedTx := &Transaction{rawTx: rawTX}
		return fn(wrapedTx)
	})
}

func (db *boltDB) Close() error {
	return db.client.Close()
}

func (tx *Transaction) CreateBucket(name string) (Bucket, error) {
	b, err := tx.rawTx.CreateBucketIfNotExists([]byte(name))
	if err != nil {
		return nil, err
	}
	return &boltBucket{rawBucket: b}, nil
}

func (tx *Transaction) Bucket(name string) Bucket {
	b := tx.rawTx.Bucket([]byte(name))
	return &boltBucket{rawBucket: b}
}

func (b *boltBucket) Put(key []byte, value []byte) error {
	if b == nil || b.rawBucket == nil {
		return ErrBucketNotFound
	}
	return b.rawBucket.Put(key, value)
}

func (b *boltBucket) Get(key []byte) []byte {
	if b == nil || b.rawBucket == nil {
		return nil
	}
	var result []byte
	value := b.rawBucket.Get(key)
	if value != nil {
		result = make([]byte, len(value))
		copy(result, value)
		return result
	}
	return nil
}

func (b *boltBucket) Delete(key []byte) error {
	if b == nil || b.rawBucket == nil {
		return ErrBucketNotFound
	}
	return b.rawBucket.Delete(key)
}

func (b *boltBucket) Cursor() Cursor {
	if b == nil || b.rawBucket == nil {
		return &boltCursor{}
	}
	return &boltCursor{rawCursor: b.rawBucket.Cursor()}
}

func (c *boltCursor) First() (key []byte, value []byte) {
	if c == nil || c.rawCursor == nil {
		return nil, nil
	}
	return c.rawCursor.First()
}

func (c *boltCursor) Last() (key []byte, value []byte) {
	if c == nil || c.rawCursor == nil {
		return nil, nil
	}
	return c.rawCursor.Last()
}

func (c *boltCursor) Prev() (key []byte, value []byte) {
	if c == nil || c.rawCursor == nil {
		return nil, nil
	}
	return c.rawCursor.Prev()
}

func (c *boltCursor) Next() (key []byte, value []byte) {
	if c == nil || c.rawCursor == nil {
		return nil, nil
	}
	return c.rawCursor.Next()
}

func (c *boltCursor) Seek(key []byte) (k []byte, value []byte) {
	if c == nil || c.rawCursor == nil {
		return nil, nil
	}
	return c.rawCursor.Seek(key)
}
