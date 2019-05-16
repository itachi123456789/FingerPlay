package main

import (
	"database/sql"

	"github.com/garyburd/redigo/redis"

	"gopkg.in/mgo.v2"
)

type Context struct {
	//mm  *MysqlManager
	//rm  *RedisManager
	mgm *MongoManager
}

func NewContext(mgm *MongoManager) *Context {
	ctx := &Context{}
	ctx.mgm = mgm
	return ctx
}

func (ctx *Context) GetMysqlSession() (db *sql.DB, err error) {
	//return ctx.mm.GetSession()
	return nil, nil
}

func (ctx *Context) GetRedisSession() (conn redis.Conn, err error) {
	//return ctx.rm.GetSession()
	return nil, nil
}

func (ctx *Context) GetMongoSession() (session *mgo.Session, err error) {
	return ctx.mgm.GetSession()
}

var (
	DefaultContext *Context
)
