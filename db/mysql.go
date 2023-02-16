package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type dbHelper struct {
	db *sql.DB
}

func NewMySQLDBHelper(host, username, password, database string, port int) DBHelper {
	db, err := initMysql(host, username, password, database, port)
	if err != nil {
		fmt.Println("Panic Failed to init mysql", zap.Error(err)) // not log
	}
	return &dbHelper{
		db: db,
	}
}

func (h *dbHelper) Open() *sql.DB {
	return h.db
}

func (h *dbHelper) Close() error {
	return h.db.Close()
}

func (h *dbHelper) Begin() (*sql.Tx, error) {
	return h.db.Begin()
}

func (h *dbHelper) Commit(tx *sql.Tx) error {
	return tx.Commit()
}

func (h *dbHelper) RollBack(tx *sql.Tx) error {
	return tx.Rollback()
}
func initMysql(host, username, password, database string, port int) (*sql.DB, error) {
	connectionStr := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v", username, password, host, port, database)

	db, err := sql.Open("mysql", connectionStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
