package db

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"lib/common"
	"net"
	"reflect"
	"time"

	"github.com/globalsign/mgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

type mongoHelper struct {
	ctx            *context.Context
	ColName        string
	DBName         string
	TemplateObject interface{}
	collection     *mgo.Collection
	db             *mgo.Database
	mSession       *dbSession
}

type dbSession struct {
	session *mgo.Session
}

func NewDBSession(info *mgo.DialInfo, isSSL bool) dbSession {
	if isSSL {
		tlsConfig := &tls.Config{}
		tlsConfig.InsecureSkipVerify = true
		info.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		}
	}

	session, err := mgo.DialWithInfo(info)
	if err != nil {
		fmt.Println("Panic Failed to init mongo", zap.Error(err)) // not log
		panic(err)
	}

	session.SetMode(mgo.Monotonic, true)

	return dbSession{
		session: session,
	}
}
func (m *dbSession) Clone() *dbSession {
	return &dbSession{
		session: m.session.Clone(),
	}
}
func (m *dbSession) Close() {
	m.session.Close()
}
func (m *dbSession) Copy() *dbSession {
	return &dbSession{
		session: m.session.Copy(),
	}
}
func (m *dbSession) GetMGOSession() *mgo.Session {
	return m.session
}
func (m *dbSession) Valid() bool {
	return m.session.Ping() == nil
}

func NewMongoDBHelper(dbSession *dbSession, dbName, colName string, templateObject interface{}) NoSQLDBHelper {
	ctx := context.Background()

	return &mongoHelper{
		ctx:            &ctx,
		db:             dbSession.session.DB(dbName),
		collection:     dbSession.session.DB(dbName).C(colName),
		mSession:       dbSession,
		ColName:        colName,
		DBName:         dbName,
		TemplateObject: templateObject,
	}
}

// convertToObject convert bson to object
func (m *mongoHelper) convertToObject(b bson.M) (interface{}, error) {
	obj := m.NewObject()

	if b == nil {
		return obj, nil
	}

	bytes, err := bson.Marshal(b)
	if err != nil {
		return nil, err
	}

	bson.Unmarshal(bytes, obj)
	return obj, nil
}

// convertToBson Go object to map (to get / query)
func (m *mongoHelper) convertToBson(ent interface{}) (bson.M, error) {
	if ent == nil {
		return bson.M{}, nil
	}

	sel, err := bson.Marshal(ent)
	if err != nil {
		return nil, err
	}

	obj := bson.M{}
	bson.Unmarshal(sel, &obj)

	return obj, nil
}
func (m *mongoHelper) Close() {
	m.mSession.Close()
}
func (m *mongoHelper) Aggregate(pipeline interface{}, result interface{}) error {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return err
	}
	q := col.Pipe(pipeline)
	err = q.All(result)
	if err != nil {
		return err
	}

	return nil
}
func (m *mongoHelper) Count(query interface{}) (int64, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return 0, err
	}

	count, err := col.Find(query).Count()
	if err != nil {
		return 0, err
	}

	return int64(count), nil
}
func (m *mongoHelper) Create(entity interface{}) (interface{}, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return nil, err
	}

	// convert to bson
	obj, err := m.convertToBson(entity)
	if err != nil {
		return nil, err
	}

	// init time
	if obj["created_time"] == nil {
		obj["created_time"] = time.Now()
		obj["last_updated_time"] = obj["created_time"]
	} else {
		obj["last_updated_time"] = time.Now()
	}

	// insert
	err = col.Insert(obj)
	if err != nil {
		return nil, err
	}

	entity, _ = m.convertToObject(obj)

	list := m.NewList(1)
	listValue := reflect.Append(reflect.ValueOf(list),
		reflect.Indirect(reflect.ValueOf(entity)))

	return listValue.Interface(), nil
}
func (m *mongoHelper) CreateIndex(index mgo.Index) error {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)
	if err == nil {
		crErr := col.EnsureIndex(index)
		return crErr
	}
	return err
}
func (m *mongoHelper) CreateMany(entityList ...interface{}) (interface{}, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return nil, err
	}
	objs := []bson.M{}
	ints := []interface{}{}

	if len(entityList) == 1 {
		rt := reflect.TypeOf(entityList[0])
		switch rt.Kind() {
		case reflect.Slice:
			entityList = entityList[0].([]interface{})
		case reflect.Array:
			entityList = entityList[0].([]interface{})
		}
	}
	for _, ent := range entityList {
		obj, err := m.convertToBson(ent)
		if err != nil {
			return nil, err
		}
		if obj["created_time"] == nil {
			obj["created_time"] = time.Now()
			obj["last_updated_time"] = obj["created_time"]
		} else {
			obj["last_updated_time"] = time.Now()
		}
		objs = append(objs, obj)
		ints = append(ints, obj)
	}

	err = col.Insert(ints...)
	if err != nil {
		return err, nil
	}
	list := m.NewList(len(entityList))
	listValue := reflect.ValueOf(list)
	for _, obj := range objs {
		entity, _ := m.convertToObject(obj)
		listValue = reflect.Append(listValue, reflect.Indirect(reflect.ValueOf(entity)))
	}

	return listValue.Interface(), nil
}
func (m *mongoHelper) Delete(selector interface{}) error {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return err
	}

	err = col.Remove(selector)
	if err != nil {
		return err
	}
	return nil
}
func (m *mongoHelper) Distinct(filter interface{}, key string, result interface{}) error {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)
	if err != nil {
		return err
	}
	err = col.Find(filter).Distinct(key, result)
	if err != nil {
		return err
	}
	return nil
}
func (m *mongoHelper) GetColWith(s *dbSession) (*mgo.Collection, error) {
	if m.collection == nil {
		m.collection = m.db.C(m.ColName)
	}
	return m.collection.With(m.mSession.GetMGOSession()), nil
}
func (m *mongoHelper) GetFreshSession() *dbSession {
	return m.mSession.Copy()
}
func (m *mongoHelper) IncreOne(query interface{}, fieldName string, value int) (interface{}, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)
	if err != nil {
		return nil, err
	}

	updater := bson.M{}
	updater[fieldName] = value
	change := mgo.Change{
		Update:    bson.M{"$inc": updater},
		ReturnNew: true,
		Upsert:    true,
	}

	obj := m.NewObject()
	_, err = col.Find(query).Limit(1).Apply(change, obj)
	list := m.NewList(1)
	listValue := reflect.Append(reflect.ValueOf(list),
		reflect.Indirect(reflect.ValueOf(obj)))
	if err != nil {
		if err.Error() == "not found" {
			return nil, errors.New(common.ReasonNotFound.Code())
		}
		return nil, err
	}
	return listValue.Interface(), nil
}
func (m *mongoHelper) Init(s *dbSession) error {
	if len(m.DBName) == 0 || len(m.ColName) == 0 {
		return errors.New("require valid DB name and collection name")
	}
	m.db = s.session.DB(m.DBName)
	m.collection = m.db.C(m.ColName)
	m.mSession = s
	return nil
}
func (m *mongoHelper) NewList(limit int) interface{} {
	t := reflect.TypeOf(m.TemplateObject)
	return reflect.MakeSlice(reflect.SliceOf(t), 0, limit).Interface()
}
func (m *mongoHelper) NewObject() interface{} {
	t := reflect.TypeOf(m.TemplateObject)
	// fmt.Println(t)
	v := reflect.New(t)
	return v.Interface()
}
func (m *mongoHelper) PullOne(query interface{}, updater interface{}, sortFields []string) (interface{}, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return nil, err
	}

	bUpdater, err := m.convertToBson(updater)
	if err != nil {
		return nil, err
	}

	change := mgo.Change{
		Update: bson.M{
			"$pull": bUpdater,
			"$currentDate": bson.M{
				"last_updated_time": true,
			},
		},
		ReturnNew: true,
	}
	tmp := bson.M{}
	q := col.Find(query)
	q.Limit(1).Sort(sortFields...)
	return m.applyUpdateOne(q, &change, &tmp)
}
func (m *mongoHelper) PushOne(query interface{}, updater interface{}, sortFields []string) (interface{}, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return nil, err
	}

	bUpdater, err := m.convertToBson(updater)
	if err != nil {
		return nil, err
	}

	change := mgo.Change{
		Update: bson.M{
			"$push": bUpdater,
			"$currentDate": bson.M{
				"last_updated_time": true,
			},
		},
		ReturnNew: true,
	}
	tmp := bson.M{}
	q := col.Find(query)
	q.Limit(1).Sort(sortFields...)
	return m.applyUpdateOne(q, &change, &tmp)
}
func (m *mongoHelper) Query(query interface{}, offset int, limit int, reverse bool) (interface{}, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return nil, err
	}

	q := col.Find(query)
	if limit == 0 {
		limit = 1000
	}
	if limit > 0 {
		q.Limit(limit)
	}
	if offset > 0 {
		q.Skip(offset)
	}
	if reverse {
		q.Sort("-_id")
	}

	list := m.NewList(limit)
	err = q.All(&list)

	if err != nil || reflect.ValueOf(list).Len() == 0 {
		return nil, errors.New(common.ReasonNotFound.Code())
	}
	return list, err
}
func (m *mongoHelper) QueryOne(query interface{}) (interface{}, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return nil, err
	}

	q := col.Find(query)
	q.Limit(1)

	list := m.NewList(1)
	err = q.All(&list)
	if err != nil || reflect.ValueOf(list).Len() == 0 {
		return nil, errors.New(common.ReasonNotFound.Code())
	}
	return list, nil
}

func deleteEmpty(s []string) []string {
	var result []string
	for _, str := range s {
		if str != "" {
			result = append(result, str)
		}
	}
	return result
}

func (m *mongoHelper) QueryS(query interface{}, offset int, limit int, sortFields ...string) (interface{}, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return nil, err
	}

	q := col.Find(query)
	if limit == 0 {
		limit = 1000
	}
	if limit > 0 {
		q.Limit(limit)
	}
	if offset > 0 {
		q.Skip(offset)
	}
	if len(sortFields) > 0 {
		sortFields = deleteEmpty(sortFields)
		q.Sort(sortFields...)
	}

	list := m.NewList(limit)
	err = q.All(&list)

	if err != nil || reflect.ValueOf(list).Len() == 0 {
		return nil, errors.New(common.ReasonNotFound.Code())
	}
	return list, nil
}
func (m *mongoHelper) Update(query interface{}, updater interface{}) error {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return err
	}

	obj, err := m.convertToBson(updater)
	if err != nil {
		return err
	}
	obj["last_updated_time"] = time.Now()

	info, err := col.UpdateAll(query, bson.M{
		"$set": obj,
	})
	if err != nil {
		return err
	}

	if info.Matched == 0 {
		return errors.New(common.ReasonNotFound.Code())
	}

	return nil
}

func (m *mongoHelper) applyUpdateOne(q *mgo.Query, change *mgo.Change, newResult interface{}) (interface{}, error) {
	obj := m.NewObject()
	info, err := q.Apply(*change, newResult)
	if (err != nil && err.Error() == "not found") || (info != nil && info.Matched == 0) {
		return nil, errors.New(common.ReasonNotFound.Code())
	}

	if err == nil {
		bytes, mErr := bson.Marshal(newResult)
		if mErr == nil {
			bson.Unmarshal(bytes, obj)
			list := m.NewList(1)
			listValue := reflect.Append(reflect.ValueOf(list),
				reflect.Indirect(reflect.ValueOf(obj)))
			return listValue.Interface(), nil
		}
	}

	return nil, err
}

func (m *mongoHelper) UpdateOne(query interface{}, updater interface{}) (interface{}, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return nil, err
	}

	bUpdater, err := m.convertToBson(updater)
	if err != nil {
		return nil, err
	}
	bUpdater["last_updated_time"] = time.Now()

	change := mgo.Change{
		Update:    bson.M{"$set": bUpdater},
		ReturnNew: true,
	}
	tmp := bson.M{}
	q := col.Find(query)
	q.Limit(1)
	return m.applyUpdateOne(q, &change, &tmp)
}
func (m *mongoHelper) UpdateOneSort(query interface{}, sortFields []string, updater interface{}) (interface{}, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return nil, err
	}

	bUpdater, err := m.convertToBson(updater)
	if err != nil {
		return nil, err
	}
	bUpdater["last_updated_time"] = time.Now()

	change := mgo.Change{
		Update:    bson.M{"$set": bUpdater},
		ReturnNew: true,
	}
	tmp := bson.M{}
	q := col.Find(query)
	q.Limit(1).Sort(sortFields...)
	return m.applyUpdateOne(q, &change, &tmp)
}
func (m *mongoHelper) UpsertOne(query interface{}, updater interface{}) (interface{}, error) {
	s := m.GetFreshSession()
	defer s.Close()
	col, err := m.GetColWith(s)

	if err != nil {
		return nil, err
	}

	bUpdater, err := m.convertToBson(updater)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	bUpdater["last_updated_time"] = now

	change := mgo.Change{
		Update: bson.M{
			"$set": bUpdater,
			"$setOnInsert": bson.M{
				"created_time": now,
			},
		},
		ReturnNew: true,
		Upsert:    true,
	}

	obj := m.NewObject()
	tmp := bson.M{}
	_, err = col.Find(query).Limit(1).Apply(change, &tmp)
	if err == nil {
		bytes, err := bson.Marshal(tmp)
		if err == nil {
			bson.Unmarshal(bytes, obj)
			list := m.NewList(1)
			listValue := reflect.Append(reflect.ValueOf(list),
				reflect.Indirect(reflect.ValueOf(obj)))
			return listValue.Interface(), nil
		}

	}

	if err.Error() == "not found" {
		return nil, errors.New(common.ReasonNotFound.Code())
	}
	return nil, err
}
