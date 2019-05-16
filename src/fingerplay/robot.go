package main

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	log "code.google.com/p/log4go"
)

var (
	RobotLifetimeSecond int64 = 300
)

type RobotManager struct {
	mux  sync.RWMutex
	uid  int
	idle []*Robot
	idx  int64
}

func NewRobotManager(uid int, lifetimeSecond int64) *RobotManager {
	rm := &RobotManager{}
	rm.uid = uid
	rm.idx = -1

	RobotLifetimeSecond = lifetimeSecond

	return rm
}

func (rm *RobotManager) nextRobotAvatar() *RobotAvatar {
	return Conf.Robots[int(atomic.AddInt64(&(rm.idx), int64(1))%int64(len(Conf.Robots)))]
}

func (rm *RobotManager) GoGoGo(lv int, balance float64) {
	if robot := rm.NextRobot(lv); robot != nil {
		go func() {
			if err := robot.PlayLoop(float64(lv), balance); err != nil {
				log.Error("Robot %d play loop end with error: %s. level=%d", robot.Uid, err, robot.Level)
			} else {
				log.Debug("Robot %d play loop end succeed. level=%d", robot.Uid, robot.Level)
			}

			rm.Idle(robot)
			robot.Logout()
			robot.Leave()
		}()
	}
	return
}

func (rm *RobotManager) NextRobot(lv int) (robot *Robot) {
	rm.mux.Lock()
	if len(rm.idle) > 0 {
		robot = rm.idle[0]
		rm.idle = rm.idle[1:]
	} else {
		robot = NewRobot()
		robot.Uid = rm.uid
	}

	robot.Reset()
	robot.Level = lv

	rm.mux.Unlock()

	return
}

func (rm *RobotManager) Idle(robot *Robot) {
	rm.mux.Lock()
	rm.idle = append(rm.idle, robot)
	rm.mux.Unlock()
}

type Robot struct {
	rand         *rand.Rand
	AccessToken  string  `json:"access_token"`
	Level        int     `json:"level"`
	MatchId      string  `json:"match_id"`
	Round        int     `json:"round"`
	Uid          int     `json:"uid"`
	BeginBalance float64 `json:"begin_balance"`
	EndBalance   float64 `json:"end_balance"`
}

func NewRobot() *Robot {
	robot := &Robot{}
	return robot
}

func (r *Robot) readyWaitTime() time.Duration {
	return time.Duration(r.rand.Intn(4)+4) * time.Second
}

func (r *Robot) PlayLoop(level, balance float64) (err error) {
	request := &LoginAIRequest{}
	request.Uid = r.Uid
	request.Balance = balance
	request.Level = level

	response := &LoginAIResponse{}

	if err = DefaultAccountManager.LoginAI(request, response); err != nil {
		log.Error("LoginAI(%#v, %#v) failed: %s", request, response, err)
		return
	}

	if response.Data.AccessToken == "" {
		log.Error("LoginAI(%#v, %#v) failed: bad access token \"\"", request, response)
		return ErrAccessToken
	}

	log.Debug("LoginAI(%#v, %#v) succeed", request, response)

	r.AccessToken = response.Data.AccessToken

	if err = r.Match(); err != nil {
		log.Error("Robot %s match failed: %s", r.AccessToken, err)
		return
	}

	begin := time.Now().Unix()

	for {
		time.Sleep(r.readyWaitTime())
		if err = r.Ready(); err != nil {
			log.Error("Robot %s ready failed: %s", r.AccessToken, err)
			break
		}

		if time.Now().Unix()-begin >= RobotLifetimeSecond {
			log.Debug("Robot %s exit because of the lifetime is overload", r.AccessToken)
			break
		}
	}

	return
}

func (r *Robot) Match() (err error) {
	request := &MatchRequest{}
	request.Level = r.Level
	request.AccessToken = r.AccessToken

	response := &MatchResponse{}

	if err = DefaultLogicImpl.Match(request, response); err != nil {
		return
	}

	if response.Code != ResponseCodeOK {
		return ErrResponseCodeNotOK
	}

	log.Debug("Robot %d match succeed: %#v, %#v", request, response)

	r.MatchId = response.Data.MatchId
	r.Round = response.Data.Round

	for _, cp := range response.Data.Competitors {
		if cp.AccessToken != "" {
			r.BeginBalance = cp.Balance
		}
	}
	return
}

func (r *Robot) Logout() (err error) {
	request := &LogoutAIRequest{AccessToken: r.AccessToken}
	response := &LogoutAIResponse{}
	if err = DefaultAccountManager.LogoutAI(request, response); err != nil {
		log.Error("LogoutAI failed: %s", err)
	}
	return
}

func (r *Robot) Leave() (err error) {
	request := &LeaveRequest{}
	request.MatchId = r.MatchId
	request.AccessToken = r.AccessToken

	response := &LeaveResponse{}

	if err = DefaultLogicImpl.Leave(request, response); err != nil {
		log.Error("Robot %s leave failed: %s", r.AccessToken, err)
		return
	}

	if response.Code != ResponseCodeOK {
		return ErrResponseCodeNotOK
	}

	return
}

func (r *Robot) Ready() (err error) {
	request := &ReadyRequest{}
	request.Operate = r.NextOperate()
	request.MatchId = r.MatchId
	request.Round = r.Round
	request.AccessToken = r.AccessToken

	response := &ReadyResponse{}

	if err = DefaultLogicImpl.Ready(request, response); err != nil {
		return
	}

	if response.Code != ResponseCodeOK {
		return ErrResponseCodeNotOK
	}

	log.Debug("Robot %d ready succeed: %#v, %#v", request, response)

	r.Round = response.Data.Round

	for _, result := range response.Data.Results {
		if result.AccessToken != "" {
			r.EndBalance = result.Balance
			break
		}
	}
	return
}

func (r *Robot) NextOperate() int {
	n := r.rand.Intn(15000)
	if n < 5000 {
		return Stone
	} else if n < 10000 {
		return Scissors
	} else {
		return Paper
	}
}

func (r *Robot) Reset() {
	r.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	r.AccessToken = ""
	r.Level = 0
	r.MatchId = ""
	r.Round = 0
	r.BeginBalance = 0.0
	r.EndBalance = 0.0
	return
}

type RobotAvatar struct {
	Avatar   string `toml:"avatar"`
	Nickname string `toml:"nickname"`
}

var (
	DefaultRobotManager *RobotManager
)
