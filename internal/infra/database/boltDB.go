package database

import (
	bolt "go.etcd.io/bbolt"
)

type boltDB struct {
	client *bolt.DB
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
		err := b.Put(key, value)
		return err
	})
}

func (db *boltDB) Get(bucketName string, key []byte) ([]byte, error) {
	var result []byte
	err := db.client.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
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
		err := b.Delete(key)
		return err
	})
}

func (db *boltDB) Close() error {
	return db.client.Close()
}
