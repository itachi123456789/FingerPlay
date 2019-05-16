package main

import (
	"encoding/json"
	"sync/atomic"
	"time"

	log "code.google.com/p/log4go"
)

const (
	Stone = iota
	Paper
	Scissors
)

const (
	Lost = iota
	Won
	Draw
)

const (
	CompetitorStatusIdle = iota
	CompetitorStatusReady
	CompetitorStatusDisposed
)

type OnlineNumberRequest struct {
}

type OnlineNumberResponse struct {
	Code int                      `json:"code"`
	Msg  string                   `json:"msg"`
	Data OnlineNumberResponseData `json:"data"`
}

type OnlineNumberResponseData struct {
	Number int     `json:"number"`
	Rooms  []*Room `json:"rooms"`
}

type Room struct {
	Level  int `json:"level"`
	Number int `json:"number"`
}

func (response *OnlineNumberResponse) JSON() []byte {
	v, _ := json.Marshal(response)
	return v
}

type LeaveRequest struct {
	AccessToken string `json:"access_token"`
	MatchId     string `json:"match_id"`
}

type LeaveResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (response *LeaveResponse) JSON() []byte {
	v, _ := json.Marshal(response)
	return v
}

type RankingRequest struct {
	AccessToken string `json:"access_token"`
}

type RankingResponse struct {
	Code int                 `json:"code"`
	Msg  string              `json:"msg"`
	Data RankingResponseData `json:"data"`
}

func (response *RankingResponse) JSON() []byte {
	v, _ := json.Marshal(response)
	return v
}

type RankingResponseData struct {
	Results []*ResultLog
}

func (rrd RankingResponseData) Len() int {
	return len(rrd.Results)
}

func (rrd RankingResponseData) Swap(i, j int) {
	rrd.Results[i], rrd.Results[j] = rrd.Results[j], rrd.Results[i]
}

func (rrd RankingResponseData) Less(i, j int) bool {
	return rrd.Results[i].WinAmount < rrd.Results[j].WinAmount
}

type ReadyStatusRequest struct {
	AccessToken string `json:"access_token"`
	MatchId     string `json:"match_id"`
}

type ReadyStatusResponse struct {
	Code int                     `json:"code"`
	Msg  string                  `json:"msg"`
	Data ReadyStatusResponseData `json:"data"`
}

func (response *ReadyStatusResponse) JSON() []byte {
	v, _ := json.Marshal(response)
	return v
}

type ReadyStatusResponseData struct {
	Status int `json:"status"`
}

type MatchRequest struct {
	Level       int    `json:"level"`
	AccessToken string `json:"access_token"`
}

type MatchResponse struct {
	Code int               `json:"code"`
	Msg  string            `json:"msg"`
	Data MatchResponseData `json:"data"`
}

type MatchResponseData struct {
	ServerTimestamp int64         `json:"server_timestamp"`
	ExpireTimestamp int64         `json:"expire_timestamp"`
	MatchId         string        `json:"match_id"`
	Round           int           `json:"round"`
	Competitors     []*Competitor `json:"competitors"`
	TimeoutSecond   int           `json:"timeout_second"`
}

type Competitor struct {
	AccessToken string  `json:"access_token"`
	Balance     float64 `json:"balance"`
	Nickname    string  `json:"nickname"`
	//FbOpenId    string  `json:"fb_open_id"`
	Avatar string `json:"avatar"`

	// DO NOT EDIT THESE FIELD!
	uid         int                 `json:"-"`
	operate     int                 `json:"-"`
	readyCh     chan *ReadyResponse `json:"-"`
	status      int64               `json:"-"`
	accessToken string              `json:"-"`
	keepAliveTs int64               `json:"-"`
}

func (cp *Competitor) IsMan() bool {
	return cp.uid > Conf.MaxRobotUid
}

func (cp *Competitor) KeepAlive() {
	ts := time.Now().Unix()
	atomic.StoreInt64(&(cp.keepAliveTs), ts)
}

func (cp *Competitor) Leave() {
	atomic.StoreInt64(&(cp.keepAliveTs), 0)
}

func (cp *Competitor) GetKeepAliveTs() int64 {
	return atomic.LoadInt64(&(cp.keepAliveTs))
}

func (cp *Competitor) Ready(op int) bool {
	ok := atomic.CompareAndSwapInt64(&(cp.status), CompetitorStatusIdle, CompetitorStatusReady)
	if ok {
		cp.operate = op
	}
	return ok
}

func (cp *Competitor) UpdateOperate(op int) {
	cp.operate = op
}

func (cp *Competitor) Status() int {
	return int(atomic.LoadInt64(&(cp.status)))
}

func (cp *Competitor) Idle() bool {
	return atomic.CompareAndSwapInt64(&(cp.status), CompetitorStatusReady, CompetitorStatusIdle)
}

func (cp *Competitor) IsReady() bool {
	return atomic.LoadInt64(&(cp.status)) == CompetitorStatusReady
}

func (cp *Competitor) GetOperate() int {
	return cp.operate
}

func (cp *Competitor) closeChan() {
	defer func() {
		if err := recover(); err != nil {
			log.Warn("competitor %#v close error: %s", cp, err)
		}
	}()

	close(cp.readyCh)
}

func (cp *Competitor) Dispose() {
	atomic.StoreInt64(&(cp.status), CompetitorStatusDisposed)
}

func (response *MatchResponse) JSON() []byte {
	v, _ := json.Marshal(response)
	return v
}

type ReadyRequest struct {
	Operate     int    `json:"operate"`
	MatchId     string `json:"match_id"`
	Round       int    `json:"round"`
	AccessToken string `json:"access_token"`
}

type ReadyResponse struct {
	Code int               `json:"code"`
	Msg  string            `json:"msg"`
	Data ReadyResponseData `json:"data"`
}

type ReadyResponseData struct {
	ServerTimestamp int64     `json:"server_timestamp"`
	ExpireTimestamp int64     `json:"expire_timestamp"`
	Round           int       `json:"round"`
	Results         []*Result `json:"results"`
}

type Result struct {
	AccessToken string  `json:"access_token"`
	Operate     int     `json:"operate"`
	Status      int     `json:"status"`
	Balance     float64 `json:"balance"`
	Win         float64 `json:"win"`
}

func (response *ReadyResponse) JSON() []byte {
	v, _ := json.Marshal(response)
	return v
}
