package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	log "code.google.com/p/log4go"
)

const (
	MatchSessionStatusOK = iota
	MatchSessionStatusDisposed
)

type Logic interface {
	Match(request *MatchRequest, response *MatchResponse) (err error)
	Ready(request *ReadyRequest, response *ReadyResponse) (err error)
	ReadyStatus(request *ReadyStatusRequest, response *ReadyStatusResponse) (err error)
	Leave(request *LeaveRequest, response *LeaveResponse) (err error)
	Ranking(request *RankingRequest, response *RankingResponse) (err error)
	OnlineNumber(request *OnlineNumberRequest, response *OnlineNumberResponse) (err error)
}

func (impl *LogicImpl) OnlineNumber(request *OnlineNumberRequest, response *OnlineNumberResponse) (err error) {
	response.Data.Number = impl.onlineNumber()

	rooms := []*Room{}

	// level 1
	n1 := float64(response.Data.Number) * float64(55) / float64(100)
	rooms = append(rooms, &Room{
		Level:  1,
		Number: int(n1),
	})

	// level 10
	n10 := float64(response.Data.Number) * float64(35) / float64(100)
	rooms = append(rooms, &Room{
		Level:  10,
		Number: int(n10),
	})

	// level 100
	n100 := float64(response.Data.Number) * float64(8) / float64(100)
	rooms = append(rooms, &Room{
		Level:  100,
		Number: int(n100),
	})

	// level 500
	n500 := float64(response.Data.Number) - n1 - n10 - n100
	rooms = append(rooms, &Room{
		Level:  500,
		Number: int(n500),
	})

	response.Data.Rooms = rooms
	return
}

func (impl *LogicImpl) Leave(request *LeaveRequest, response *LeaveResponse) (err error) {
	defer func() {
		if response.Code != ResponseCodeOK {
			log.Error("[%s] Leave => [%s]", getCodeDescription(response.Code), request.AccessToken)
		}
	}()

	ms := impl.getMatchSession(request.MatchId)
	if ms == nil {
		response.Code = ResponseCodeBadMatchId
		return ErrMatchId
	}

	if cp := ms.getCompetitor(request.AccessToken); cp != nil {
		cp.Leave()
	} else {
		response.Code = ResponseCodeBadAccessToken
		return ErrAccessToken
	}

	return
}

func (impl *LogicImpl) Ranking(request *RankingRequest, response *RankingResponse) (err error) {
	defer func() {
		if response.Code != ResponseCodeOK {
			log.Error("[%s] Ranking => [%s]", getCodeDescription(response.Code), request.AccessToken)
		}
	}()

	var (
		results []*ResultLog
	)

	if results, err = DefaultStatisticsManager.Ranking(); err != nil {
		response.Code = ResponseCodeInternalError
	} else {
		response.Data.Results = results
	}

	if len(results) < RankingLimit {
		response.Data.Results = append(response.Data.Results, Conf.FakeRanking[:RankingLimit-len(results)]...)
	}

	// SORT
	sort.Sort(response.Data)

	return
}

func (impl *LogicImpl) Ready(request *ReadyRequest, response *ReadyResponse) (err error) {
	defer func() {
		if response.Code != ResponseCodeOK {
			log.Error("[%s] Ready => [%s][%s]", getCodeDescription(response.Code), request.AccessToken, getOperateDescription(request.Operate))
		}
	}()

	if !impl.allowOperate(request.Operate) {
		response.Code = ResponseCodeBadOperate
		return ErrOperate
	}

	ms := impl.getMatchSession(request.MatchId)
	if ms == nil {
		response.Code = ResponseCodeBadMatchId
		return ErrMatchId
	}

	if request.Round != ms.getRound() {
		response.Code = ResponseCodeBadRound
		return ErrRound
	}

	if cp := ms.getCompetitor(request.AccessToken); cp != nil {
		cp.KeepAlive()
		if cp.Balance < float64(ms.Level) {
			response.Code = ResponseCodeInsufficientBalance
			return ErrInsufficientBalance
		}
	} else {
		response.Code = ResponseCodeBadAccessToken
		return ErrAccessToken
	}

	ch := ms.waitReady(request, impl)
	*response = *(<-ch)

	if response.Code != ResponseCodeOK {
		close(ch)
	}

	return
}

func (impl *LogicImpl) ReadyStatus(request *ReadyStatusRequest, response *ReadyStatusResponse) (err error) {
	defer func() {
		if response.Code != ResponseCodeOK {
			log.Error("[%s] ReadyStatus => [%s]", getCodeDescription(response.Code), request.AccessToken)
		}
	}()

	ms := impl.getMatchSession(request.MatchId)
	if ms == nil {
		response.Code = ResponseCodeBadMatchId
		return ErrMatchId
	}

	if cp := ms.getCompetitor(request.AccessToken); cp != nil {
		cp.KeepAlive()
		response.Data.Status = ms.getOpponentStatus(request.AccessToken)
	} else {
		response.Code = ResponseCodeBadAccessToken
		return ErrAccessToken
	}

	return
}

func (impl *LogicImpl) Match(request *MatchRequest, response *MatchResponse) (err error) {
	defer func() {
		if response.Code != ResponseCodeOK {
			log.Error("[%s] Match => [%d][%s]", getCodeDescription(response.Code), request.Level, request.AccessToken)
		}
	}()

	wl := impl.getWaitingList(request.Level)
	if wl == nil {
		response.Code = ResponseCodeBadLevel
		return ErrLevel
	}

	_request := &DescribeUserRequest{}
	_request.AccessToken = request.AccessToken

	_response := &DescribeUserResponse{}

	if err = impl.accountManager.DescribeUser(_request, _response); err != nil || _response.Code != ResponseCodeOK {
		log.Error("DescribeUser(%#v, %#v) failed: %s", _request, _response, err)
		response.Code = ResponseCodeInternalError
		return
	}

	if _response.Data.Balance < float64(request.Level) {
		response.Code = ResponseCodeInsufficientBalance
		return ErrInsufficientBalance
	}

	ch := wl.WaitMatch(_response.Data.Uid, _response.Data.Balance, request.AccessToken, _response.Data.Nickname, _response.Data.FbOpenId)
	*(response) = *(<-ch)
	close(ch)

	return
}

type LogicImpl struct {
	accountManager       *AccountManager
	waitingListMap       map[int]*WaitingList
	matchMux             sync.RWMutex
	matchSessionMap      map[string]*MatchSession
	matchWaitSecond      int
	operateTimeoutSecond int
	rand                 *rand.Rand
}

func NewLogicImpl(accountManager *AccountManager, levels []int, operateTimeoutSecond, matchWaitSecond int) Logic {
	impl := &LogicImpl{}
	impl.accountManager = accountManager
	impl.waitingListMap = make(map[int]*WaitingList)
	impl.rand = rand.New(rand.NewSource(time.Now().Unix()))
	for _, lv := range levels {
		impl.waitingListMap[lv] = NewWaitingList(lv)
	}
	impl.matchSessionMap = make(map[string]*MatchSession)
	impl.matchWaitSecond = matchWaitSecond
	impl.operateTimeoutSecond = operateTimeoutSecond
	go impl.matchLoop()
	go impl.cleanLoop()

	return impl
}

func (impl *LogicImpl) getMatchWaitSecond() int {
	return impl.matchWaitSecond - impl.rand.Intn(6)
}

func (impl *LogicImpl) onlineNumber() int {
	impl.matchMux.RLock()
	n := len(impl.matchSessionMap)
	impl.matchMux.RUnlock()
	return Conf.BaseOnlineNumbers[time.Now().Hour()%24] + n
}

func (impl *LogicImpl) matchLoop() {
	for {
		time.Sleep(1 * time.Second)
		now := time.Now().Unix()
		for _, wl := range impl.waitingListMap {
			wl.match(impl, now)
		}
	}
}

func (impl *LogicImpl) cleanLoop() {
	for {
		time.Sleep(1 * time.Second)
		now := time.Now().Unix()
		impl.matchMux.Lock()
		for id, ms := range impl.matchSessionMap {
			if ms.clean(now, int64(impl.operateTimeoutSecond+3)) > 0 {
				delete(impl.matchSessionMap, id)
				ms.dispose()
			}
		}
		impl.matchMux.Unlock()
	}
}

func (impl *LogicImpl) getMatchSession(matchId string) (ms *MatchSession) {
	impl.matchMux.RLock()
	ms = impl.matchSessionMap[matchId]
	impl.matchMux.RUnlock()
	return
}

func (impl *LogicImpl) onMatchSuccess(level int, matchId string, round int, competitor1, competitor2 *Competitor) (err error) {
	impl.matchMux.Lock()
	defer impl.matchMux.Unlock()

	_, ok := impl.matchSessionMap[matchId]
	if ok {
		log.Error("onMatchSuccess conflict: the matchId %s has already exists", matchId)
		return ErrMatchId
	}

	impl.matchSessionMap[matchId] = NewMatchSession(level, matchId, round, competitor1, competitor2)

	return
}

func (impl *LogicImpl) getWaitingList(lv int) *WaitingList { return impl.waitingListMap[lv] }

func (impl *LogicImpl) allowOperate(op int) bool {
	return op == Stone || op == Paper || op == Scissors
}

func (wl *WaitingList) WaitMatch(uid int, balance float64, accessToken, nickname, fbOpenId string) chan *MatchResponse {
	wd := NewWaitingData(uid, balance, accessToken, nickname, fbOpenId, time.Now().Unix())
	wl.push(wd)
	return wd.ch
}

func (wl *WaitingList) push(wd *WaitingData) {
	wl.mux.Lock()
	wl.list = append(wl.list, wd)
	wl.mux.Unlock()
}

func (wl *WaitingList) shift() (wd *WaitingData) {
	wl.mux.Lock()
	if len(wl.list) > 0 {
		wd = wl.list[0]
		wl.list = wl.list[1:]
	}
	wl.mux.Unlock()
	return
}

func (wl *WaitingList) match(impl *LogicImpl, now int64) {
	wl.mux.Lock()
	for len(wl.list) >= 2 {
		wl.matchOnce(impl)
	}
	if len(wl.list) > 0 {
		wl.cleanTimeout(now)
	}
	if len(wl.list) == 1 && wl.list[0].IsMan() && now-wl.list[0].ts > int64(impl.getMatchWaitSecond()) {
		wl.matchAI(wl.list[0].balance)
	}
	wl.mux.Unlock()
}

func (wl *WaitingList) matchAI(balance float64) {
	DefaultRobotManager.GoGoGo(wl.level, balance)
}

func (wl *WaitingList) matchOnce(impl *LogicImpl) {
	wd1 := wl.list[0]
	wd2 := wl.list[1]
	wl.list = wl.list[2:]

	if wd1.uid == wd2.uid {
		response := &MatchResponse{Code: ResponseCodeKickOut}
		if wd1.Before(wd2) {
			wl.list = append(wl.list, wd2)
			wd1.Notify(response)
			log.Warn("Kick out: old=%#v new=%#v", wd1, wd2)
		} else {
			wl.list = append(wl.list, wd1)
			wd2.Notify(response)
			log.Warn("Kick out: old=%#v new=%#v", wd2, wd1)
		}

		return
	}

	// Robot can not play with robot
	if !wd1.IsMan() && !wd2.IsMan() {
		response := &MatchResponse{Code: ResponseCodeKickOut}
		wd1.Notify(response)
		wd2.Notify(response)
		return
	}

	competitor1 := &Competitor{
		readyCh:     make(chan *ReadyResponse, 1),
		status:      CompetitorStatusIdle,
		uid:         wd1.uid,
		accessToken: wd1.accessToken,
		Balance:     wd1.balance,
		Nickname:    wd1.nickname,
		Avatar:      getAvatarByOpenId(wd1.fbOpenId),
	}

	competitor1.KeepAlive()

	if !competitor1.IsMan() {
		robot := DefaultRobotManager.nextRobotAvatar()
		competitor1.Avatar = robot.Avatar
		competitor1.Nickname = robot.Nickname
	}

	competitor2 := &Competitor{
		readyCh:     make(chan *ReadyResponse, 1),
		status:      CompetitorStatusIdle,
		uid:         wd2.uid,
		accessToken: wd2.accessToken,
		Balance:     wd2.balance,
		Nickname:    wd2.nickname,
		Avatar:      getAvatarByOpenId(wd2.fbOpenId),
	}

	competitor2.KeepAlive()

	if !competitor2.IsMan() {
		robot := DefaultRobotManager.nextRobotAvatar()
		competitor2.Avatar = robot.Avatar
		competitor2.Nickname = robot.Nickname
	}

	matchId := GetGUID()
	round := 0

	var (
		response1, response2 *MatchResponse
	)

	if impl.onMatchSuccess(wl.level, matchId, round, competitor1, competitor2) != nil {
		response1 = &MatchResponse{}
		response1.Code = ResponseCodeBadMatchStatus
		response1.Data.MatchId = ""
		response1.Data.Competitors = nil

		response2 = response1
	} else {
		ts := time.Now().UnixNano() / 1000000
		response1 = &MatchResponse{}
		response1.Data.ServerTimestamp = ts
		response1.Data.ExpireTimestamp = ts + int64(impl.operateTimeoutSecond*1000)
		response1.Data.MatchId = matchId
		response1.Data.Round = round
		response1.Data.TimeoutSecond = impl.operateTimeoutSecond
		response1.Data.Competitors = append(response1.Data.Competitors, &Competitor{
			AccessToken: wd1.accessToken,
			Balance:     wd1.balance,
			Nickname:    competitor1.Nickname,
			Avatar:      competitor1.Avatar,
		}, &Competitor{
			Balance:  wd2.balance,
			Nickname: competitor2.Nickname,
			Avatar:   competitor2.Avatar,
		})

		response2 = &MatchResponse{}
		response2.Data.ServerTimestamp = ts
		response2.Data.ExpireTimestamp = ts + int64(impl.operateTimeoutSecond*1000)
		response2.Data.MatchId = matchId
		response2.Data.Round = round
		response2.Data.TimeoutSecond = impl.operateTimeoutSecond
		response2.Data.Competitors = append(response2.Data.Competitors, &Competitor{
			Balance:  wd1.balance,
			Nickname: competitor1.Nickname,
			Avatar:   competitor1.Avatar,
		}, &Competitor{
			AccessToken: wd2.accessToken,
			Balance:     wd2.balance,
			Nickname:    competitor2.Nickname,
			Avatar:      competitor2.Avatar,
		})
	}

	wd1.Notify(response1)
	wd2.Notify(response2)

	log.Debug("[OK] Match => [%d][%s][%d vs %d]", wl.level, matchId, competitor1.uid, competitor2.uid)
}

func (wl *WaitingList) cleanTimeout(now int64) {
	wd := wl.list[0]
	if now-wd.GetTs() > 30 {
		wl.list = wl.list[1:]
		response := &MatchResponse{}
		response.Code = ResponseCodeWaitMatchTimeout
		wd.Notify(response)
	}
}

type WaitingData struct {
	ts          int64
	ch          chan *MatchResponse
	uid         int
	balance     float64
	nickname    string
	fbOpenId    string
	accessToken string
}

func NewWaitingData(uid int, balance float64, accessToken, nickname, fbOpenId string, ts int64) *WaitingData {
	wd := &WaitingData{}
	wd.ts = ts
	wd.uid = uid
	wd.balance = balance
	wd.accessToken = accessToken
	wd.nickname = nickname
	wd.fbOpenId = fbOpenId
	wd.ch = make(chan *MatchResponse, 1)
	return wd
}

func (wd *WaitingData) IsMan() bool {
	return wd.uid > 2000
}

func (wd *WaitingData) Before(that *WaitingData) bool {
	return wd.ts < that.ts
}

func (wd *WaitingData) Notify(response *MatchResponse) {
	wd.ch <- response
}

func (wd *WaitingData) GetTs() int64 { return wd.ts }

type MatchSession struct {
	status      int64
	mux         sync.RWMutex
	Level       int
	MatchId     string
	Round       int
	Competitors []*Competitor
}

func NewMatchSession(level int, matchId string, round int, competitor1, competitor2 *Competitor) *MatchSession {
	ms := &MatchSession{}
	ms.Level = level
	ms.MatchId = matchId
	ms.Round = round
	ms.Competitors = append(ms.Competitors, competitor1, competitor2)
	return ms
}

func (ms *MatchSession) clean(now int64, timeout int64) (n int) {
	ms.mux.Lock()
	for _, cp := range ms.Competitors {
		if now-cp.GetKeepAliveTs() > timeout {
			n++
		}
	}
	ms.mux.Unlock()
	return
}

func (ms *MatchSession) getRound() (r int) {
	ms.mux.RLock()
	r = ms.Round
	ms.mux.RUnlock()
	return
}

func (ms *MatchSession) getCompetitor(accessToken string) (cp *Competitor) {
	ms.mux.RLock()
	for _, _cp := range ms.Competitors {
		if _cp.accessToken == accessToken {
			cp = _cp
			break
		}
	}
	ms.mux.RUnlock()
	return
}

func (ms *MatchSession) waitReady(request *ReadyRequest, impl *LogicImpl) chan *ReadyResponse {
	competitor := ms.getCompetitor(request.AccessToken)
	if competitor == nil {
		response := &ReadyResponse{}
		response.Code = ResponseCodeBadAccessToken

		// 此处必须是一个无阻塞的chan
		ch := make(chan *ReadyResponse, 1)
		ch <- response

		return ch
	}

	if !competitor.Ready(request.Operate) {
		response := &ReadyResponse{}
		response.Code = ResponseCodeBadReadyStatus

		// 此处必须是一个无阻塞的chan
		ch := make(chan *ReadyResponse, 1)
		ch <- response

		return ch
	}

	ms.checkReady(impl)

	return competitor.readyCh
}

func (ms *MatchSession) checkReady(impl *LogicImpl) {
	ms.mux.Lock()
	defer ms.mux.Unlock()

	if ms.isDisposed() {
		return
	}

	cp1 := ms.Competitors[0]
	cp2 := ms.Competitors[1]

	if !(cp1.IsReady() && cp2.IsReady()) {
		return
	}

	blc1 := cp1.Balance
	blc2 := cp2.Balance

	result := ms.judge(ms.Level, cp1, cp2)

	round := ms.Round

	code := ResponseCodeOK

	win1 := float64(0)
	win2 := float64(0)

	if result != Draw {
		request := &TransferRequest{}
		request.MatchId = ms.MatchId
		request.Round = round
		request.Level = ms.Level
		request.Amount = float64(ms.Level)
		request.FromCost = 0
		request.ToCost = getCost(ms.Level)

		if request.ToCost == 0 {
			log.Warn("Bad level cost 0: %d", ms.Level)
		}

		if result == Won {
			request.FromUid = cp2.uid
			request.FromAccessToken = cp2.accessToken
			request.ToUid = cp1.uid
			request.ToAccessToken = cp1.accessToken
			win1 = float64(ms.Level) - getCost(ms.Level)
			win2 = -float64(ms.Level)
		} else {
			request.FromUid = cp1.uid
			request.FromAccessToken = cp1.accessToken
			request.ToUid = cp2.uid
			request.ToAccessToken = cp2.accessToken
			win1 = -float64(ms.Level)
			win2 = float64(ms.Level) - getCost(ms.Level)
		}

		response := &TransferResponse{}

		if err := impl.accountManager.Transfer(request, response); err != nil {
			log.Error("Transfer failed: %s, request: %#v", err, request)
			code = ResponseCodeInternalError
		} else if response.Code != ResponseCodeOK {
			log.Error("Transfer failed: bad code, request: %#v, response: %#v", request, response)
			code = ResponseCodeInternalError
		} else {
			if result == Won {
				cp1.Balance = response.Data.ToBalance
				cp2.Balance = response.Data.FromBalance
			} else {
				cp1.Balance = response.Data.FromBalance
				cp2.Balance = response.Data.ToBalance
			}
		}

		onIncomingResult(win1, cp1)
		onIncomingResult(win2, cp2)
	}

	ms.Round++

	ts := time.Now().UnixNano() / 1000000

	// response to cp1
	resp1 := &ReadyResponse{}
	resp1.Code = code
	resp1.Data.Round = ms.Round
	resp1.Data.ServerTimestamp = ts
	resp1.Data.ExpireTimestamp = ts + int64(impl.operateTimeoutSecond*1000)

	if code == ResponseCodeOK {

		resp1.Data.Results = append(resp1.Data.Results, &Result{
			AccessToken: cp1.accessToken,
			Operate:     cp1.GetOperate(),
			Status:      result,
			Balance:     cp1.Balance,
			Win:         win1,
		}, &Result{
			Operate: cp2.GetOperate(),
			Status:  ms.getOpponentResult(result),
			Balance: cp2.Balance,
			Win:     win2,
		})
	}

	cp1.readyCh <- resp1

	// response to cp2
	resp2 := &ReadyResponse{}
	resp2.Code = code
	resp2.Data.Round = ms.Round
	resp2.Data.ServerTimestamp = ts
	resp2.Data.ExpireTimestamp = ts + int64(impl.operateTimeoutSecond*1000)

	if code == ResponseCodeOK {
		resp2.Data.Results = append(resp2.Data.Results, &Result{
			Operate: cp1.GetOperate(),
			Status:  result,
			Balance: cp1.Balance,
			Win:     win1,
		}, &Result{
			AccessToken: cp2.accessToken,
			Operate:     cp2.GetOperate(),
			Status:      ms.getOpponentResult(result),
			Balance:     cp2.Balance,
			Win:         win2,
		})
	}

	cp2.readyCh <- resp2

	cp1.Idle()
	cp2.Idle()

	cp1.KeepAlive()
	cp2.KeepAlive()

	log.Debug("[%s] Ready => [%s][%d][%s][%d][%s][%d][%s][%s%.3f]", getCodeDescription(code), getResultDescription(result), ms.Level, ms.MatchId, ms.Round, cp1.accessToken, cp1.uid, getOperateDescription(cp1.GetOperate()), getSign(cp1.Balance-blc1), cp1.Balance-blc1)
	log.Debug("[%s] Ready => [%s][%d][%s][%d][%s][%d][%s][%s%.3f]", getCodeDescription(code), getResultDescription(ms.getOpponentResult(result)), ms.Level, ms.MatchId, ms.Round, cp2.accessToken, cp2.uid, getOperateDescription(cp2.GetOperate()), getSign(cp2.Balance-blc2), cp2.Balance-blc2)
}

func (ms *MatchSession) getOpponentResult(result int) int {
	if result == Lost {
		return Won
	} else if result == Won {
		return Lost
	} else {
		return Draw
	}
}

func (ms *MatchSession) isDisposed() bool {
	return atomic.LoadInt64(&(ms.status)) == MatchSessionStatusDisposed
}

func (ms *MatchSession) getOpponentStatus(accessToken string) int {
	ms.mux.RLock()
	defer ms.mux.RUnlock()

	for _, cp := range ms.Competitors {
		if cp.accessToken != accessToken {
			return cp.Status()
		}
	}

	return CompetitorStatusDisposed
}

func (ms *MatchSession) dispose() {
	ms.mux.Lock()
	defer ms.mux.Unlock()

	if !atomic.CompareAndSwapInt64(&(ms.status), MatchSessionStatusOK, MatchSessionStatusDisposed) {
		return
	}

	for _, cp := range ms.Competitors {
		if cp.IsReady() {
			response := &ReadyResponse{}
			response.Code = ResponseCodeWaitReadyTimeout
			cp.readyCh <- response
			cp.Idle()
			// 这里不需要关闭chan是因为，receiver会关闭
			log.Debug("dispose competitor %s cp because of ready timeout(has ready), matchId=%s round=%d", cp.accessToken, ms.MatchId, ms.Round)
		} else {
			log.Debug("dispose competitor %s cp because of ready timeout(not ready), matchId=%s round=%d", cp.accessToken, ms.MatchId, ms.Round)
			// 这里需要关闭chan是因为，没有receiver
			cp.closeChan()
		}

		cp.Dispose()
	}
}

func (ms *MatchSession) judge(lv int, cp1, cp2 *Competitor) int {
	op1 := cp1.GetOperate()
	op2 := cp2.GetOperate()

	if op1 == op2 {
		return Draw
	}

	if !cp1.IsMan() || !cp2.IsMan() {
		return DefaultRiskController.Judge(float64(lv), cp1, cp2)
	}

	switch op1 {
	case Stone:
		if op2 == Paper {
			return Lost
		}
		break
	case Paper:
		if op2 == Scissors {
			return Lost
		}
		break
	case Scissors:
		if op2 == Stone {
			return Lost
		}
		break
	}

	return Won
}

type WaitingList struct {
	mux   sync.RWMutex
	level int
	list  []*WaitingData
}

func NewWaitingList(lv int) *WaitingList {
	return &WaitingList{level: lv}
}

func (ms *MatchSession) JSON() []byte {
	v, _ := json.Marshal(ms)
	return v
}

var (
	DefaultLogicImpl Logic
)

var (
	n int64 = 0
)

func getAvatarByOpenId(openId string) string {
	return fmt.Sprintf("https://graph.facebook.com/%s/picture?type=large", openId)
	if openId == "x" {
		i := atomic.AddInt64(&n, 1) % int64(Conf.AvatarNum)
		if i == 0 {
			i = int64(Conf.AvatarNum)
		}
		return fmt.Sprintf(Conf.AvatarUrlTemplate, i)
	} else {
		return fmt.Sprintf("https://graph.facebook.com/%s/picture?type=large", openId)
	}
}

var (
	m int64 = -1
)

func onIncomingResult(winAmount float64, cp *Competitor) {
	resultLog := &ResultLog{}
	resultLog.Uid = cp.uid
	resultLog.Avatar = cp.Avatar
	resultLog.WinAmount = winAmount
	resultLog.Nickname = cp.Nickname
	resultLog.TimeUpdated = time.Now().Unix()
	DefaultStatisticsManager.OnResult(resultLog)
}
