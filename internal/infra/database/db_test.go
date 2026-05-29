package database

import (
	"os"
	"testing"
)

const testDBFile = "test_temp.db"

func TestInitDB(t *testing.T) {
	defer os.Remove(testDBFile)

	db, err := InitDB(testDBFile)
	if err != nil {
		t.Fatalf("Failed to create database")
	}
	defer db.Close()

	if db == nil {
		t.Fatalf("Database instance can not be empty")
	}
}

func TestCreatBucket(t *testing.T) {

	defer os.Remove(testDBFile)
	db, _ := InitDB(testDBFile)
	err := db.CreateBucket("testBucket")
	if err != nil {
		t.Fatalf("Failed to create Bucket")
	}

	defer db.Close()

}

func TestPutAndGet(t *testing.T) {
	defer os.Remove(testDBFile)

	db, _ := InitDB(testDBFile)
	defer db.Close()

	db.CreateBucket("testBucket")
	err := db.Put("testBucket", []byte("testKey"), []byte("testValue"))
	if err != nil {
		t.Fatalf("Failed to put data")
	}

	res, err := db.Get("testBucket", []byte("testKey"))
	if err != nil {
		t.Fatalf("Failed to get data")
	}

	if string(res) != "testValue" {
		t.Fatalf("data do not match, expect: %s, get: %s", "testValue", string(res))
	}
}

func TestDelete(t *testing.T) {
	defer os.Remove(testDBFile)

	db, _ := InitDB(testDBFile)
	defer db.Close()

	db.CreateBucket("testBucket")
	db.Put("testBucket", []byte("testKey"), []byte("testValue"))

	err := db.Delete("testBucket", []byte("testKey"))
	if err != nil {
		t.Fatalf("Failed to delete data")
	}

	val, _ := db.Get("testBucket", []byte("testKey"))
	if val != nil {
		t.Fatalf("deleted value is not nil, is %s", string(val))
	}
}
