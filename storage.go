package main

import (
	"github.com/qustavo/dotsql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

)

type BlurbDB struct {
	db        sqlx.DB
	queries map[string]string
}

func GetBlurbDB(dbName, queryFileName string) BlurbDB {
	db := sqlx.MustConnect("sqlite3", dbName)
	dot, _ := dotsql.LoadFromFile(queryFileName) // FIXME: pull routeMapFromDir from parade

	bdb := BlurbDB{*db, dot.QueryMap()}
	println(bdb.queries["init"])
	db.MustExec(bdb.queries["init"])
	
	return bdb
}
