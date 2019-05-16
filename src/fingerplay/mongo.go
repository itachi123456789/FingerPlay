package main

import (
	"gopkg.in/mgo.v2"
)

// mgo使用文档：http://gopkg.in/mgo.v2
type MongoManager struct {
	config  MongoConfig
	session *mgo.Session
}

type MongoConfig struct {
	serverAddr string
}

func NewMongoManager(config MongoConfig) *MongoManager {
	var (
		err error
	)
	mgm := &MongoManager{}
	mgm.config = config

	mgm.session, err = mgo.Dial(mgm.config.serverAddr)
	if err != nil {
		panic(err)
	}

	mgm.session.SetMode(mgo.Eventual, true)
	return mgm
}

func (mgm *MongoManager) GetSession() (session *mgo.Session, err error) {
	return mgm.session.Clone(), nil
}
