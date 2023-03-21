package db

import (
	"database/sql"

	"github.com/globalsign/mgo"
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
	Close()
	Aggregate(pipeline interface{}, result interface{}) error
	Count(query interface{}) (int64, error)
	Create(entity interface{}) (interface{}, error)
	CreateIndex(index mgo.Index) error
	CreateMany(entityList ...interface{}) (interface{}, error)
	Delete(selector interface{}) error
	Distinct(filter interface{}, key string, result interface{}) error
	GetColWith(*dbSession) (*mgo.Collection, error)
	GetFreshSession() *dbSession
	IncreOne(query interface{}, fieldName string, value int) (interface{}, error)
	// Init(s *DBSession) error
	NewList(limit int) interface{}
	NewObject() interface{}
	PullOne(query interface{}, updater interface{}, sortFields []string) (interface{}, error)
	PushOne(query interface{}, updater interface{}, sortFields []string) (interface{}, error)
	Query(query interface{}, offset int, limit int, reverse bool) (interface{}, error)
	QueryOne(query interface{}) (interface{}, error)
	QueryS(query interface{}, offset int, limit int, sortFields ...string) (interface{}, error)
	Update(query interface{}, updater interface{}) error
	UpdateOne(query interface{}, updater interface{}) (interface{}, error)
	UpdateOneSort(query interface{}, sortFields []string, updater interface{}) (interface{}, error)
	UpsertOne(query interface{}, updater interface{}) (interface{}, error)
}
