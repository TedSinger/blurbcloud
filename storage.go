package main

import (
	"github.com/qustavo/dotsql"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"database/sql"
    "log"
	"embed"
)

type BlurbDB struct {
	db        sqlx.DB
	queries map[string]string
}
//go:embed queries.sql
var queries embed.FS

func GetBlurbDB(dbName, queryFileName string) BlurbDB {
	_db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		log.Fatal(err)
	}
	_db.Close()
	db := sqlx.MustConnect("sqlite3", dbName)
	r, _ := queries.Open("queries.sql")
	dot, _ := dotsql.Load(r)

	bdb := BlurbDB{*db, dot.QueryMap()}
	println(bdb.queries["init"])
	db.MustExec(bdb.queries["init"])
	
	return bdb
}
