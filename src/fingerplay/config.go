package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"time"

	"github.com/BurntSushi/toml"

	log "code.google.com/p/log4go"
)

var (
	Conf     = &Config{}
	confFile string
)

func init() {
	Conf.ServerName = "fingerplay"
	flag.StringVar(&confFile, "c", "conf/fingerplay.toml", "config file path")
}

type Config struct {
	Debug                bool           `toml:"debug"`
	ServerName           string         `toml:"-"`
	HttpBindAddr         string         `toml:"http_bind_addr"`
	Levels               []int          `toml:"levels"`
	OperateTimeoutSecond int            `toml:"operate_timeout_second"`
	MatchWaitSecond      int            `toml:"match_wait_second"`
	EndpointDescribeUser string         `toml:"endpoint_describe_user"`
	EndpointTransfer     string         `toml:"endpoint_transfer"`
	EndpointLoginAI      string         `toml:"endpoint_login_ai"`
	RobotUid             int            `toml:"robot_uid"`
	RobotFbOpenId        string         `toml:"robot_fb_open_id"`
	RobotLifetimeSecond  int64          `toml:"robot_lifetime_second"`
	MongoServerAddrs     string         `toml:"mongo_server_addrs"`
	MongoDb              string         `toml:"mongo_db"`
	FakeRanking          []*ResultLog   `toml:"fake_ranking"`
	AvatarNum            int            `toml:"avatar_num"`
	AvatarUrlTemplate    string         `toml:"avatar_url_template"`
	Nicknames            []string       `toml:"nicknames"`
	MaxRobotUid          int            `toml:"max_robot_uid"`
	BaseOnlineNumbers    []int          `toml:"base_online_numbers"`
	Robots               []*RobotAvatar `toml:"robot"`
}

func (cfg *Config) JSON() []byte {
	v, _ := json.Marshal(cfg)
	return v
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

func InitConfig() (err error) {
	var (
		v []byte
	)

	if v, err = ioutil.ReadFile(confFile); err != nil {
		return
	}

	_, err = toml.Decode(string(v), Conf)

	Debug = Conf.Debug
	log.Info("Conf: %s", Conf.JSON())
	return
}
