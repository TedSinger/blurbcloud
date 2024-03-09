package main

import (
	"github.com/qustavo/dotsql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"database/sql"
    "log"
)

type BlurbDB struct {
	db        sqlx.DB
	queries map[string]string
}

func GetBlurbDB(dbName, queryFileName string) BlurbDB {
	_db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatal(err)
	}
	_db.Close()
	db := sqlx.MustConnect("sqlite3", dbName)
	dot, _ := dotsql.LoadFromFile(queryFileName) // FIXME: pull routeMapFromDir from parade

	bdb := BlurbDB{*db, dot.QueryMap()}
	println(bdb.queries["init"])
	db.MustExec(bdb.queries["init"])
	
	return bdb
}
