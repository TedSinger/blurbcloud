package main

import (
	"github.com/boltdb/bolt"

)

type BlurbDB struct {
	db *bolt.DB
}

func GetBlurbDB(dbName string) BlurbDB {
	db, _ := bolt.Open(dbName, 0600, nil)
	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("Blurbs"))
		return nil
	})
	bdb := BlurbDB{db}
	return bdb
}

func (bdb BlurbDB) Close() {
	bdb.db.Close()
}

func (bdb BlurbDB) readBlurb(blurbId string) []byte {
	var data []byte
	bdb.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Blurbs"))
		data = b.Get([]byte(blurbId))
		return nil
	})
	if data == nil {
		return nil
	} else {
		return data
	}
}

func (bdb BlurbDB) writeBlurb(blurbId string, data []byte) error {
	err := bdb.db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("Blurbs"))
		b := tx.Bucket([]byte("Blurbs"))
		err := b.Put([]byte(blurbId), data)
		return err
	})
	return err
}
