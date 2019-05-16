package main

import (
	"math/rand"
	"sync"
	"time"

	log "code.google.com/p/log4go"
)

const (
	RobotSessionStatusUnused = 0
	RobotSessionStatusUsed   = 1
)

type RobotSession struct {
	Balance     float64
	AccessToken string
}

func NewRobotSession() *RobotSession {
	rs := &RobotSession{}
	rs.AccessToken = GetGUID()
	return rs
}

func (rs *RobotSession) Reset() {
	rs.Balance = 0
}

type AccountManager struct {
	endpointDescribeUser string
	endpointTransfer     string
	endpointLoginAI      string
	robotSessionMux      sync.RWMutex
	idleRobotSessions    []*RobotSession
	robotSessionMap      map[string]*RobotSession
	rand                 *rand.Rand
}

func NewAccountManager(endpointDescribeUser, endpointTransfer, endpointLoginAI string) *AccountManager {
	am := &AccountManager{}
	am.endpointDescribeUser = endpointDescribeUser
	am.endpointTransfer = endpointTransfer
	am.endpointLoginAI = endpointLoginAI
	am.robotSessionMap = make(map[string]*RobotSession)
	am.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	return am
}

func (am *AccountManager) GetRobotSession(accessToken string) (rs *RobotSession) {
	am.robotSessionMux.RLock()
	rs = am.robotSessionMap[accessToken]
	am.robotSessionMux.RUnlock()
	return
}

func (am *AccountManager) DescribeUser(request *DescribeUserRequest, response *DescribeUserResponse) (err error) {
	rs := am.GetRobotSession(request.AccessToken)
	if rs != nil {
		//Uid      int     `json:"uid"`
		//FbOpenId string  `json:"fb_open_id"`
		//Nickname string  `json:"nickname"`
		//Balance  float64 `json:"balance"`
		response.Data.Uid = Conf.RobotUid
		response.Data.FbOpenId = Conf.RobotFbOpenId
		//response.Nickname
		response.Data.Balance = rs.Balance

		return
	}
	return Post(am.endpointDescribeUser, request, response)
}

func (am *AccountManager) Transfer(request *TransferRequest, response *TransferResponse) (err error) {
	fromRs := am.GetRobotSession(request.FromAccessToken)
	toRs := am.GetRobotSession(request.ToAccessToken)

	err = Post(am.endpointTransfer, request, response)

	if fromRs != nil {
		fromRs.Balance -= (request.Amount + request.FromCost)
		response.Data.FromBalance = fromRs.Balance
	}

	if toRs != nil {
		toRs.Balance += (request.Amount - request.ToCost)
		response.Data.ToBalance = toRs.Balance
	}

	return
}

func (am *AccountManager) LogoutAI(request *LogoutAIRequest, response *LogoutAIResponse) (err error) {
	am.robotSessionMux.Lock()
	rs := am.robotSessionMap[request.AccessToken]
	if rs != nil {
		delete(am.robotSessionMap, request.AccessToken)
		am.idleRobotSessions = append(am.idleRobotSessions, rs)
	}
	am.robotSessionMux.Unlock()
	if rs != nil {
		log.Debug("LogoutAI ok: accesstoken=%s", rs.AccessToken)
	} else {
		log.Debug("LogoutAI failed: not found accesstoken=%s", request.AccessToken)
		err = ErrAccessToken
	}

	return
}

func (am *AccountManager) LoginAI(request *LoginAIRequest, response *LoginAIResponse) (err error) {
	var (
		rs *RobotSession
	)

	am.robotSessionMux.Lock()
	if len(am.idleRobotSessions) > 0 {
		rs = am.idleRobotSessions[0]
		am.idleRobotSessions = am.idleRobotSessions[1:]
	}
	am.robotSessionMux.Unlock()

	if rs == nil {
		rs = NewRobotSession()
	} else {
		rs.Reset()
	}

	factor := 1 + float64(am.rand.Intn(10)+1)/float64(11)

	if request.Balance < request.Level {
		request.Balance = request.Level
	}

	for rs.Balance < request.Level {
		if am.rand.Intn(4) >= 2 {
			rs.Balance = float64(uint64(request.Balance * factor))
		} else {
			rs.Balance = float64(uint64(request.Balance / factor))
		}
	}

	am.robotSessionMux.Lock()
	am.robotSessionMap[rs.AccessToken] = rs
	am.robotSessionMux.Unlock()

	response.Data.AccessToken = rs.AccessToken

	log.Debug("LoginAI: accesstoken=%s", rs.AccessToken)
	return
	//return Post(am.endpointLoginAI, request, response)
}

type DescribeUserRequest struct {
	AccessToken string `json:"access_token"`
}

type DescribeUserResponse struct {
	Code int                      `json:"code"`
	Msg  string                   `json:"msg"`
	Data DescribeUserResponseData `json:"data"`
}

type DescribeUserResponseData struct {
	Uid      int     `json:"uid"`
	FbOpenId string  `json:"fb_open_id"`
	Nickname string  `json:"nickname"`
	Balance  float64 `json:"balance"`
}

type TransferRequest struct {
	MatchId         string  `json:"-"`
	Round           int     `json:"-"`
	Level           int     `json:"-"`
	FromUid         int     `json:"from_uid"`
	FromAccessToken string  `json:"-"`
	ToUid           int     `json:"to_uid"`
	ToAccessToken   string  `json:"-"`
	Amount          float64 `json:"amount"`
	FromCost        float64 `json:"from_cost"`
	ToCost          float64 `json:"to_cost"`
	// just for robot fake balance
	AccessToken string `json:"-"`
}

type TransferResponse struct {
	Code int                  `json:"code"`
	Msg  string               `json:"msg"`
	Data TransferResponseData `json:"data"`
}

type TransferResponseData struct {
	FromBalance float64 `json:"from_balance"`
	ToBalance   float64 `json:"to_balance"`
}

type LogoutAIRequest struct {
	AccessToken string `json:"-"`
}

type LogoutAIResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type LoginAIRequest struct {
	Uid     int     `json:"uid"`
	Balance float64 `json:"-"`
	Level   float64 `json:"-"`
}

type LoginAIResponse struct {
	Code int                 `json:"code"`
	Msg  string              `json:"msg"`
	Data LoginAIResponseData `json:"data"`
}

type LoginAIResponseData struct {
	AccessToken string `json:"access_token"`
}

var (
	DefaultAccountManager *AccountManager
)
