package config

import (
	"athenabot/util"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"runtime"
)

type Whitelist struct {
	GroupsId       []int64  `json:"groups_id,omitempty"`
	GroupsUsername []string `json:"groups_username,omitempty"`
}

type Modules struct {
	EnableMemberVerify   bool `json:"enable_member_verify"`
	EnableMars           bool `json:"enable_mars"`
	EnableCommand        bool `json:"enable_command"`
	EnablePrivateCommand bool `json:"enable_private_command"`
}

type Webhook struct {
	Endpoint    string `json:"endpoint"`
	CertFile    string `json:"cert_file"`
	CertKeyFile string `json:"cert_key_file"`
	ListenAddr  string `json:"listen_addr"`
	Token       string `json:"token"`
}

type Config struct {
	Whitelist        Whitelist `json:"whitelist"`
	DisableWhitelist bool      `json:"disable_whitelist"`
	RedisHost        string    `json:"redis_host"`
	BotToken         string    `json:"bot_token"`
	KeyTTL           uint      `json:"key_ttl"`
	LogLevel         uint8     `json:"log_level"`
	Commands         []string  `json:"commands"`
	PrivateCommands  []string  `json:"private_commands"`
	Modules          Modules   `json:"modules"`
	UpdatesType      string    `json:"updates_type"`
	Webhook          Webhook   `json:"webhook"`
}

var Conf Config
var PrivateCommandsMap = make(map[string]uint8)
var CommandsMap = make(map[string]uint8)
var WhitelistUsernameMap = make(map[string]int)
var WhitelistIdMap = make(map[int64]int)

func init() {
	config, err := ioutil.ReadFile(os.Getenv("BOT_CONFIG"))
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(config, &Conf); err != nil {
		panic(err)
	}

	for _, i := range Conf.Commands {
		CommandsMap[i] = 0
	}
	for _, i := range Conf.PrivateCommands {
		PrivateCommandsMap[i] = 0
	}
	for _, i := range Conf.Whitelist.GroupsUsername {
		WhitelistUsernameMap[i] = 0
	}
	for _, i := range Conf.Whitelist.GroupsId {
		WhitelistIdMap[i] = 0
	}

	switch {
	case Conf.LogLevel >= 3:
		logrus.SetLevel(logrus.DebugLevel)
	case Conf.LogLevel == 2:
		logrus.SetLevel(logrus.InfoLevel)
	case Conf.LogLevel == 1:
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.ErrorLevel)
	}
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			fileName := path.Base(frame.File)
			return frame.Function, fileName
		},
	})

	logrus.Debugf("config:%v", util.LogMarshal(Conf))
}
