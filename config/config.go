package config

import (
	"athenabot/util"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"runtime"

	"github.com/sirupsen/logrus"
)

type Modules struct {
	EnableMars           bool `json:"enable_mars"`
	EnableCommand        bool `json:"enable_command"`
	EnablePrivateCommand bool `json:"enable_private_command"`
	EnablePrivateChat    bool `json:"enable_private_chat"`
}

type Webhook struct {
	Endpoint    string `json:"endpoint"`
	CertFile    string `json:"cert_file"`
	CertKeyFile string `json:"cert_key_file"`
	ListenAddr  string `json:"listen_addr"`
	Token       string `json:"token"`
}

type MarsOCR struct {
	EnableOCR       bool    `json:"enable_ocr"`
	EnableWhitelist bool    `json:"enable_whitelist"`
	DocURL          string  `json:"doc_url"`
	DocProvider     string  `json:"doc_provider"`
	OcrURL          string  `json:"ocr_url"`
	OcrProvider     string  `json:"ocr_provider"`
	MinPhrase       int     `json:"min_phrase"`
	MinHitRatio     float32 `json:"min_hit_ratio"`
}

type ChatGuard struct {
	EnableGuard      bool     `json:"enable_guard"`
	EnableWhitelist  bool     `json:"enable_whitelist"`
	GuardServerURL   string   `json:"guard_server_url"`
	CategoriesFilter []string `json:"categories_filter"`
	SafeLabel        string   `json:"safe_label"`
}

type ChatBot struct {
	EnableChatbot    bool   `json:"enable_chat_bot"`
	EnableWhitelist  bool   `json:"enable_whitelist"`
	EnableReply      bool   `json:"enable_reply"`
	ChatBotServerURL string `json:"chat_bot_server_url"`
	ChatRandom       int    `json:"chat_random"`
	ActiveTiming     int    `json:"active_timing"`
}

type InlineQueryResultArticle struct {
	Title       string `json:"title"`
	MessageText string `json:"message_text"`
}

type Config struct {
	EnableWhitelist           bool                       `json:"enable_whitelist"`
	EnableMarsWhitelist       bool                       `json:"enable_mars_whitelist"`
	RedisHost                 string                     `json:"redis_host"`
	BotToken                  string                     `json:"bot_token"`
	KeyTTL                    uint                       `json:"key_ttl"`
	LogLevel                  uint8                      `json:"log_level"`
	DisableCommands           []string                   `json:"disable_commands"`
	DisablePrivateCommands    []string                   `json:"disable_private_commands"`
	Modules                   Modules                    `json:"modules"`
	UpdatesType               string                     `json:"updates_type"`
	Webhook                   Webhook                    `json:"webhook"`
	MarsOCR                   MarsOCR                    `json:"mars_ocr"`
	SudoAdmins                []int64                    `json:"sudo_admins"`
	OwnerID                   int64                      `json:"owner_id"`
	InlineQueryResultArticles []InlineQueryResultArticle `json:"inline_query_result_articles"`
	ChatGuard                 ChatGuard                  `json:"chat_guard"`
	ChatBot                   ChatBot                    `json:"chat_bot"`
}

var (
	Conf                      Config
	DisablePrivateCommandsMap = make(map[string]uint8)
	DisableCommandsMap        = make(map[string]uint8)
	CategoriesFilter          = make(map[string]struct{})
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			fileName := path.Base(frame.File)
			return frame.Function, fileName
		},
	})
	config, err := ioutil.ReadFile(os.Getenv("BOT_CONFIG"))
	if err != nil {
		logrus.Fatalln(err)
	}
	if err := json.Unmarshal(config, &Conf); err != nil {
		logrus.Fatalln(err)
	}

	for _, i := range Conf.DisableCommands {
		DisableCommandsMap[i] = 0
	}
	for _, i := range Conf.DisablePrivateCommands {
		DisablePrivateCommandsMap[i] = 0
	}
	for _, i := range Conf.ChatGuard.CategoriesFilter {
		CategoriesFilter[i] = struct{}{}
	}

	switch {
	case Conf.LogLevel >= 3:
		logrus.SetReportCaller(true)
		logrus.SetLevel(logrus.DebugLevel)
	case Conf.LogLevel == 2:
		logrus.SetLevel(logrus.InfoLevel)
	case Conf.LogLevel == 1:
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.ErrorLevel)
	}
	if Conf.ChatBot.ChatRandom == 0 {
		Conf.ChatBot.ChatRandom = 10
	}
	if Conf.ChatBot.ActiveTiming == 0 {
		Conf.ChatBot.ActiveTiming = 3600
	}
	logrus.Infof("config:%v", util.LogMarshal(Conf))
}
