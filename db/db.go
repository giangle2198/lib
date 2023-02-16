package db

import (
	"database/sql"

	"go.mongodb.org/mongo-driver/mongo"
)

// DBHelper is helper of DB
type DBHelper interface {
	Open() *sql.DB
	Close() error
	Begin() (*sql.Tx, error)
	Commit(tx *sql.Tx) error
	RollBack(tx *sql.Tx) error
}

type NoSQLDBHelper interface {
	Close() error
	Collection(name string) *mongo.Collection
}
