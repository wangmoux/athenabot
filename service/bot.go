package service

import (
	"athenabot/util"
	"context"
	"github.com/bitly/go-simplejson"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"sync"
)

type BotConfig struct {
	update        *tgbotapi.Update
	bot           *tgbotapi.BotAPI
	messageConfig tgbotapi.MessageConfig
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewBotConfig(ctx context.Context, cancel context.CancelFunc, bot *tgbotapi.BotAPI, update *tgbotapi.Update) *BotConfig {
	botConfig := &BotConfig{
		ctx:    ctx,
		cancel: cancel,
		update: update,
		bot:    bot,
		messageConfig: tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID:           update.Message.Chat.ID,
				ReplyToMessageID: update.Message.MessageID,
			},
			Text:     "无言以对",
			Entities: []tgbotapi.MessageEntity{},
		},
	}
	return botConfig
}

func (c *BotConfig) isCloseWork() bool {
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

func (c *BotConfig) sendMessage() {
	msg := tgbotapi.NewMessage(c.update.Message.Chat.ID, c.messageConfig.Text)
	msg = c.messageConfig
	_, err := c.bot.Send(msg)
	if err != nil {
		logrus.Error(err)
	}
	logrus.Debugf("send_msg:%v", util.LogMarshal(msg))
}

func (c *BotConfig) isAdmin(userID int64) bool {
	logrus.Debugf("administrators:%v", util.LogMarshal(groupsAdministratorsCache))
	if group, ok := groupsAdministratorsCache[c.update.Message.Chat.ID]; ok {
		if _, ok := group[userID]; ok {
			return true
		}
	} else {
		req, err := c.bot.Request(tgbotapi.ChatAdministratorsConfig{
			ChatConfig: tgbotapi.ChatConfig{
				ChatID: c.update.Message.Chat.ID,
			},
		})

		if !req.Ok {
			logrus.Errorln(req.ErrorCode, err)
			return false
		}

		resJson := &simplejson.Json{}
		resJson, _ = simplejson.NewJson(req.Result)
		chatAdministrators := resJson.MustArray()
		group := make(groupAdministratorsCache)
		for i := range chatAdministrators {
			id := resJson.GetIndex(i).Get("user").Get("id").MustInt64()
			group[id] = 0
		}
		groupsAdministratorsCache[c.update.Message.Chat.ID] = group
		if _, ok := group[userID]; ok {
			return true
		}
	}
	return false
}

func (c *BotConfig) getUserNameCache(wg *sync.WaitGroup, userID int64) {
	defer wg.Done()
	if _, ok := userNameCache[userID]; !ok {
		req, err := c.bot.Request(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: c.update.Message.Chat.ID,
				UserID: userID,
			},
		})
		if !req.Ok {
			logrus.Errorln(req.ErrorCode, err)
			unknownUserCache[userID] = 0
			return
		}
		userJson := &simplejson.Json{}
		userJson, _ = simplejson.NewJson(req.Result)
		userNameCache[userID] = userJson.Get("user").Get("first_name").MustString() + userJson.Get("user").Get("last_name").MustString()
	}
}
