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
	_, err := bdb.RunQuery(bdb.queries["init"], []map[string]interface{}{{}})
	if err != nil {
		println(err)
		panic(err)	
	}
	
	return bdb
}

func (p *BlurbDB) RunQuery(query string, argLists []map[string]interface{}) ([][]map[string]interface{}, error) {
	if query == "" {
		panic("query was empty")
	}
	tx, err := p.db.Beginx()
	if err != nil {
		return nil, err
	}
	groups := make([][]map[string]interface{}, 0, len(argLists))
	for _, args := range argLists {
		grp := make([]map[string]interface{}, 0) // FIXME: use previous resultset sizes to estimate future ones
		rows, err := tx.NamedQuery(query, args)

		if err != nil {
			tx.Rollback()
			return nil, err
		} else {
			for rows.Next() {
				results := make(map[string]interface{})
				err = rows.MapScan(results)
				if err != nil {
					tx.Rollback()
					return nil, err
				}
				grp = append(grp, results)
			}
		}
		groups = append(groups, grp)
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return groups, nil
}