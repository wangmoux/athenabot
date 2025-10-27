package model

import (
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	ImageDocIndexName = "mars"
)

type ImageDoc struct {
	ImagePhrases []string  `json:"image_phrases" bson:"image_phrases"`
	MarsID       string    `json:"mars_id" bson:"mars_id"`
	ChatID       int64     `json:"chat_id" bson:"chat_id"`
	CreateTime   time.Time `json:"create_time" bson:"create_time"`
}

type MysqlImageDoc struct {
	ImagePhrases string
	MarsID       string
	ChatID       int64
	CreateTime   string
}

func (MysqlImageDoc) TableName() string {
	return ImageDocIndexName
}

var ImageDocPool = &sync.Pool{
	New: func() any {
		return new(ImageDoc)
	},
}

type UpdateType string
type ChatType string

const (
	MessageType    = "message"
	InlineType     = "inline"
	CallbackType   = "callback"
	PrivateType    = "private"
	GroupType      = "group"
	SupergroupType = "supergroup"
	ChannelType    = "channel"
)

type UpdateConfig struct {
	tgbotapi.Update
	ChatID     int64
	UpdateType string
	ChatType   string
}

type ChatGuardResponse struct {
	SafeLabel  string   `json:"safe_label"`
	Categories []string `json:"categories"`
}

type ChatGuardRequest struct {
	Prompt string `json:"prompt"`
}

type ChatbotResponse struct {
	BotMessage string `json:"bot_message"`
}

type ChatbotContent struct {
	IsModel        bool            `json:"is_model,omitempty"`
	Nickname       string          `json:"nickname"`
	Content        string          `json:"content"`
	ReplyToMessage *ChatbotContent `json:"reply_to_message,omitempty"`
}

type ChatbotRequest struct {
	Contents []*ChatbotContent `json:"contents"`
	ChatType string            `json:"chat_type"`
	Model    string            `json:"model,omitempty"`
}
