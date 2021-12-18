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

func (bdb BlurbDB) readBlurb(blurbId string) string {
	var text []byte
	bdb.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Blurbs"))
		text = b.Get([]byte(blurbId))
		return nil
	})
	if text == nil {
		return METABLURB
	} else {
		return string(text)
	}
}

func (bdb BlurbDB) writeBlurb(blurbId, text string) error {
	err := bdb.db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("Blurbs"))
		b := tx.Bucket([]byte("Blurbs"))
		err := b.Put([]byte(blurbId), []byte(text))
		return err
	})
	return err
}
