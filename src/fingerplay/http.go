package main

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/valyala/fasthttp"

	log "code.google.com/p/log4go"
)

var (
	POST    = []byte("POST")
	GET     = []byte("GET")
	OPTIONS = []byte("OPTIONS")
)

type HttpApi struct {
	bindAddr string
}

func NewHttpApi(bindAddr string) *HttpApi {
	api := &HttpApi{}
	api.bindAddr = bindAddr
	return api
}

func (api *HttpApi) Start() (err error) {
	go func() {
		log.Info("fasthttp listen at %s", api.bindAddr)
		if err := fasthttp.ListenAndServe(api.bindAddr, api.fastHttpHandler); err != nil {
			log.Error("fasthttp.ListenAndServe(%q) failed: %s", api.bindAddr, err)
		}
	}()
	return
}

func (api *HttpApi) fastHttpHandler(ctx *fasthttp.RequestCtx) {

	begin := time.Now()

	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Origin, Cookie, Accept, multipart/form-data, application/json, Content-Type")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "POST,GET,OPTIONS")

	defer func() {
		log.Debug("%2fs %s %s %s", time.Now().Sub(begin).Seconds(), string(ctx.Method()), string(ctx.Path()), string(ctx.PostBody()))
	}()

	if bytes.Equal(ctx.Method(), OPTIONS) {
		return
	}

	if bytes.Equal(ctx.Method(), GET) {
		ctx.Write([]byte("Method Not Allowed"))
		return
	}
	ctx.Response.Header.Set("Content-Type", "application/json")

	switch string(ctx.Path()) {
	case "/fingerplay/v1/match":
		api.handleMatch(ctx)
		break
	case "/fingerplay/v1/ready":
		api.handleReady(ctx)
		break
	case "/fingerplay/v1/ready/status":
		api.handleReadyStatus(ctx)
		break
	case "/fingerplay/v1/ranking":
		api.handleRanking(ctx)
		break
	case "/fingerplay/v1/leave":
		api.handleLeave(ctx)
		break
	case "/fingerplay/v1/online/number":
		api.handleOnlineNumber(ctx)
		break
	default:
		log.Error("unknown url: %s", ctx.Path())
	}
}

func parse(request interface{}, ctx *fasthttp.RequestCtx) (err error) {
	if err = json.Unmarshal(ctx.PostBody(), request); err != nil {
		log.Error("json.Unmarshal(%q) failed: %s", ctx.PostBody(), err)
	}

	return
}

func (api *HttpApi) handleMatch(ctx *fasthttp.RequestCtx) {
	var (
		request  = &MatchRequest{}
		response = &MatchResponse{}
	)

	if err := parse(request, ctx); err != nil {
		response.Code = ResponseCodeBadRequestFormat
		goto out
	}

	if err := DefaultLogicImpl.Match(request, response); err != nil {
		log.Error("DefaultLogicImpl.Match failed: %s, request: %#v", err, request)
	}

out:
	ctx.Write(response.JSON())
}

func (api *HttpApi) handleReady(ctx *fasthttp.RequestCtx) {
	var (
		request  = &ReadyRequest{}
		response = &ReadyResponse{}
	)

	if err := parse(request, ctx); err != nil {
		response.Code = ResponseCodeBadRequestFormat
		goto out
	}

	if err := DefaultLogicImpl.Ready(request, response); err != nil {
		log.Error("DefaultLogicImpl.Ready failed: %s, request: %#v", err, request)
	}

out:
	ctx.Write(response.JSON())
}

func (api *HttpApi) handleReadyStatus(ctx *fasthttp.RequestCtx) {
	var (
		request  = &ReadyStatusRequest{}
		response = &ReadyStatusResponse{}
	)

	if err := parse(request, ctx); err != nil {
		response.Code = ResponseCodeBadRequestFormat
		goto out
	}

	if err := DefaultLogicImpl.ReadyStatus(request, response); err != nil {
		log.Error("DefaultLogicImpl.ReadyStatus failed: %s, request: %#v", err, request)
	}

out:
	ctx.Write(response.JSON())
}

func (api *HttpApi) handleRanking(ctx *fasthttp.RequestCtx) {
	var (
		request  = &RankingRequest{}
		response = &RankingResponse{}
	)

	if err := parse(request, ctx); err != nil {
		response.Code = ResponseCodeBadRequestFormat
		goto out
	}

	if err := DefaultLogicImpl.Ranking(request, response); err != nil {
		log.Error("DefaultLogicImpl.Ranking failed: %s, request: %#v", err, request)
	}

out:
	ctx.Write(response.JSON())
}

func (api *HttpApi) handleLeave(ctx *fasthttp.RequestCtx) {
	var (
		request  = &LeaveRequest{}
		response = &LeaveResponse{}
	)

	if err := parse(request, ctx); err != nil {
		response.Code = ResponseCodeBadRequestFormat
		goto out
	}

	if err := DefaultLogicImpl.Leave(request, response); err != nil {
		log.Error("DefaultLogicImpl.Leave failed: %s, request: %#v", err, request)
	}

out:
	ctx.Write(response.JSON())
}

func (api *HttpApi) handleOnlineNumber(ctx *fasthttp.RequestCtx) {
	var (
		request  = &OnlineNumberRequest{}
		response = &OnlineNumberResponse{}
	)

	if err := parse(request, ctx); err != nil {
		response.Code = ResponseCodeBadRequestFormat
		goto out
	}

	if err := DefaultLogicImpl.OnlineNumber(request, response); err != nil {
		log.Error("DefaultLogicImpl.OnlineNumber failed: %s, request: %#v", err, request)
	}

out:
	ctx.Write(response.JSON())
}

func InitHttp(bindAddr string) (err error) {
	return NewHttpApi(bindAddr).Start()
}
