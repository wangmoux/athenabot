package service

import (
	"athenabot/config"
	"athenabot/db"
	"athenabot/util"
	"context"
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
	"unicode/utf8"
)

type mars struct {
	Count int `json:"count,omitempty"`
	MsgID int `json:"msg_id"`
}

type MarsConfig struct {
	*BotConfig
	ctx         context.Context
	latestMars  mars
	currentMars mars
	marsID      string
	humanChatID int64
}

func NewMarsConfig(ctx context.Context, botConfig *BotConfig) *MarsConfig {
	return &MarsConfig{
		ctx:       ctx,
		BotConfig: botConfig,
	}
}

func (c *MarsConfig) getMars() {
	marsKey := util.StrBuilder(marsKeyDir, util.NumToStr(c.update.Message.Chat.ID), ":", c.marsID)
	res, err := db.RDB.Get(c.ctx, marsKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	_ = json.Unmarshal([]byte(res), &c.latestMars)
}

func (c *MarsConfig) setMars() {
	marsKey := util.StrBuilder(marsKeyDir, util.NumToStr(c.update.Message.Chat.ID), ":", c.marsID)
	c.currentMars.MsgID = c.update.Message.MessageID
	marsJson, _ := json.Marshal(c.currentMars)
	if err := db.RDB.Set(c.ctx, marsKey, marsJson, time.Second*time.Duration(config.Conf.KeyTTL)).Err(); err != nil {
		logrus.Error(err)
	}
}

func (c *MarsConfig) isMarsExists() bool {
	marsKey := util.StrBuilder(marsKeyDir, util.NumToStr(c.update.Message.Chat.ID), ":", c.marsID)
	res, err := db.RDB.Exists(c.ctx, marsKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	if res > 0 {
		return true
	}
	return false
}

func (c *MarsConfig) handleMars() {
	if c.isMarsExists() {
		c.getMars()
		c.currentMars.Count = c.latestMars.Count + 1
		c.setMars()

		humanChatID := c.update.Message.Chat.ID - c.update.Message.Chat.ID - c.update.Message.Chat.ID - 1000000000000
		LatestMarsMessage := util.StrBuilder("https://t.me/c/", util.NumToStr(humanChatID), "/", strconv.Itoa(c.latestMars.MsgID))
		c.messageConfig.Text = util.StrBuilder("这条愚蠢的消息已经火星", strconv.Itoa(c.currentMars.Count), "次了！！!")
		c.messageConfig.Entities = []tgbotapi.MessageEntity{{
			Type:   "text_link",
			URL:    LatestMarsMessage,
			Offset: 11,
			Length: utf8.RuneCountInString(strconv.Itoa(c.currentMars.Count)),
		},
		}
		c.sendMessage()
		logrus.Infof("mars_user=%v mars_id=%v", c.update.Message.From.ID, c.marsID)
		t := newTopConfig(c.ctx, c.BotConfig)
		t.setTop(marsTopKeyDir, c.update.Message.From.ID, 1)
	} else {
		c.setMars()
	}
}

func (c *MarsConfig) HandlePhoto() {
	fileUrl, _ := c.bot.GetFileDirectURL(c.update.Message.Photo[0].FileID)
	pHash, err := util.GetFilePHash(fileUrl)
	if err != nil {
		logrus.Error(err)
	}
	c.marsID = util.NumToStr(pHash)
	logrus.Debugf("mars_id=%v", c.marsID)
	c.handleMars()
}

func (c *MarsConfig) HandleVideo() {
	c.marsID = c.update.Message.Video.FileUniqueID
	c.handleMars()
}
