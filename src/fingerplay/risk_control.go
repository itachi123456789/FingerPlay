package main

import (
	"sync"
	"time"

	"gopkg.in/mgo.v2/bson"

	log "code.google.com/p/log4go"
)

var (
	Collection = "risk_control"
)

type RiskController struct {
	mux sync.RWMutex
	ctx *Context
}

type RiskConfig struct {
	BonusPool float64 `bson:"bonus_pool"`
}

func NewRiskController(ctx *Context) *RiskController {
	rc := &RiskController{}
	rc.ctx = ctx
	return rc
}

func (rc *RiskController) Judge(lv float64, cp1, cp2 *Competitor) int {
	bonusPool := float64(0)
	riskConfig := &RiskConfig{}
	begin := time.Now()
	rc.mux.Lock()
	defer rc.mux.Unlock()
	defer func() {
		log.Debug("Judge cost %2fs, bonus pool: ksh %2f", time.Now().Sub(begin).Seconds(), bonusPool)
		if err := recover(); err != nil {
			log.Error("Judge panic recover: %s", err)
		}
	}()

	session, err := rc.ctx.GetMongoSession()
	if err != nil {
		log.Error("Judge failed: %s", err)
		if !cp1.IsMan() {
			cp1.UpdateOperate(getWonOperate(cp2.GetOperate()))
			return Won
		} else {
			cp2.UpdateOperate(getWonOperate(cp1.GetOperate()))
			return Lost
		}
	}

	if session == nil {
		log.Error("Judge failed: mongodb not connected")
		if !cp1.IsMan() {
			cp1.UpdateOperate(getWonOperate(cp2.GetOperate()))
			return Won
		} else {
			cp2.UpdateOperate(getWonOperate(cp1.GetOperate()))
			return Lost
		}
	}

	defer session.Close()

	if err := session.DB(Conf.MongoDb).C(Collection).Find(nil).One(riskConfig); err != nil {
		log.Error("Judge failed: %s", err)
		if !cp1.IsMan() {
			cp1.UpdateOperate(getWonOperate(cp2.GetOperate()))
			return Won
		} else {
			cp2.UpdateOperate(getWonOperate(cp1.GetOperate()))
			return Lost
		}
	}

	if riskConfig.BonusPool < lv {
		bonusPool = riskConfig.BonusPool + lv
		if err := session.DB(Conf.MongoDb).C(Collection).Update(nil, bson.M{"$set": bson.M{"bonus_pool": bonusPool}}); err != nil {
			log.Error("Judge failed, update bonus pool %2f failed: %s", bonusPool, err)
		} else {
			log.Debug("update bonus pool %2f succeed", bonusPool)
		}

		if !cp1.IsMan() {
			cp1.UpdateOperate(getWonOperate(cp2.GetOperate()))
			return Won
		} else {
			cp2.UpdateOperate(getWonOperate(cp1.GetOperate()))
			return Lost
		}
	} else {
		op1 := cp1.GetOperate()
		op2 := cp2.GetOperate()

		result := Won

		switch op1 {
		case Stone:
			if op2 == Paper {
				result = Lost
			}
			break
		case Paper:
			if op2 == Scissors {
				result = Lost
			}
			break
		case Scissors:
			if op2 == Stone {
				result = Lost
			}
			break
		}

		if !cp1.IsMan() {
			if result == Won {
				bonusPool = riskConfig.BonusPool + lv
			} else {
				bonusPool = riskConfig.BonusPool - lv
			}
		} else {
			if result == Won {
				bonusPool = riskConfig.BonusPool - lv
			} else {
				bonusPool = riskConfig.BonusPool + lv
			}
		}

		if err := session.DB(Conf.MongoDb).C(Collection).Update(nil, bson.M{"$set": bson.M{"bonus_pool": bonusPool}}); err != nil {
			log.Error("Judge failed, update bonus pool %2f failed: %s", bonusPool, err)
		} else {
			log.Debug("update bonus pool %2f succeed", bonusPool)
		}

		return result
	}
}

var (
	DefaultRiskController *RiskController
)
