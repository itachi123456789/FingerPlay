package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	log "code.google.com/p/log4go"
)

const (
	RankingLimit = 10
)

type ResultLog struct {
	Uid         int     `bson:"uid" json:"-" toml:"-"`
	Avatar      string  `bson:"avatar" json:"avatar" toml:"avatar"`
	WinAmount   float64 `bson:"win_amount" json:"win_amount" toml:"win_amount"`
	Nickname    string  `bson:"nickname" json:"nickname" toml:"nickname"`
	TimeUpdated int64   `bson:"time_updated" json:"time_updated" toml:"time_updated"`
}

type StatisticsManager struct {
	ctx *Context
	q   chan *ResultLog
}

func NewStatisticsManager(ctx *Context) *StatisticsManager {
	sm := &StatisticsManager{}
	sm.ctx = ctx
	sm.q = make(chan *ResultLog, 20480)
	go sm.loop()
	return sm
}

func (sm *StatisticsManager) loop() {
	session, err := sm.ctx.GetMongoSession()
	if err != nil {
		panic(err)
	}
	if session == nil {
		panic("mongodb not connected")
	}
	defer session.Close()

	db := Conf.MongoDb
	co := "ranking"
	for result := range sm.q {
		condition := bson.M{
			"uid": result.Uid,
		}

		_result := &ResultLog{}
		if err := session.DB(db).C(co).Find(condition).One(_result); err != nil {
			if mgo.ErrNotFound == err {
				if err = session.DB(db).C(co).Insert(result); err != nil {
					log.Error("Insert failed: %#v, error: %s", result, err)
				}
			} else {
				log.Error("Find failed: %s, error: %s", result.Uid, err)
			}
		} else {
			if err = session.DB(db).C(co).Update(condition, bson.M{"$set": bson.M{
				"win_amount":   _result.WinAmount + result.WinAmount,
				"time_updated": result.TimeUpdated,
			}}); err != nil {
				log.Error("Update failed: %s, error: %s", result.Uid, err)
			}
		}
	}
}

func (sm *StatisticsManager) Ranking() (results []*ResultLog, err error) {
	session, err := sm.ctx.GetMongoSession()
	if err != nil {
		panic(err)
	}
	if session == nil {
		panic("mongodb not connected")
	}
	defer session.Close()

	db := Conf.MongoDb
	co := "ranking"

	results = []*ResultLog{}

	if err = session.DB(db).C(co).Find(bson.M{"win_amount": bson.M{"$gt": 0}}).Sort("-win_amount").Limit(RankingLimit).All(&results); err != nil {
		log.Error("Find ranking failed: %s", err)
	}

	return
}

func (sm *StatisticsManager) OnResult(result *ResultLog) (err error) {
	select {
	case sm.q <- result:
	default:
		log.Error("OnResult(%#v) failed: the queue is full", result)
	}
	return
}

var (
	DefaultStatisticsManager *StatisticsManager
)
