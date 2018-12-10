package util

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

type DbConfig struct {
	Dsn     string
	MaxOpen int
	MaxIdle int
}

type MysqlConnPool struct {
	db   *sql.DB
	conf *DbConfig
}

func (mysql *MysqlConnPool) Init(config *DbConfig) (err error) {
	var db *sql.DB
	if db, err = sql.Open("mysql", config.Dsn); err != nil {
		return err
	}

	db.SetMaxOpenConns(config.MaxOpen)
	db.SetMaxIdleConns(config.MaxIdle)

	mysql.db = db
	mysql.conf = config

	return
}

func (mysql *MysqlConnPool) GetDB() *sql.DB {
	return mysql.db
}

func ClearTransaction(tx *sql.Tx) {
	err := tx.Rollback()
	if err != sql.ErrTxDone && err != nil {
		log.Fatalln(err)
	}
}
