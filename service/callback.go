package service

import (
	"athenabot/db"
	"athenabot/util"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type CallBack struct {
	*BotConfig
	callbackData *CallbackData
}

func NewCallBack(botConfig *BotConfig, callbackData *CallbackData) *CallBack {
	return &CallBack{
		BotConfig:    botConfig,
		callbackData: callbackData,
	}
}

func (c *CallBack) ClearMy48hMessage() {
	if c.callbackData.UserID != c.update.CallbackQuery.From.ID {
		return
	}
	commandMessageID, _ := c.callbackData.Data.(float64)
	chat48hMessageKey := util.StrBuilder(chat48hMessageDir, util.NumToStr(c.update.Message.Chat.ID), ":", util.NumToStr(c.callbackData.UserID))
	messageIDs, err := db.RDB.HGetAll(c.ctx, chat48hMessageKey).Result()
	if err != nil {
		logrus.Error(err)
		return
	}
	for k, v := range messageIDs {
		messageID, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		messageTime, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		if time.Now().Unix()-int64(messageTime) > 172800 {
			continue
		}
		if messageID == int(commandMessageID) {
			continue
		}
		c.addDeleteMessageQueue(0, messageID)
	}
	err = db.RDB.Del(c.ctx, chat48hMessageKey).Err()
	if err != nil {
		logrus.Error(err)
	}
}

func (c *CallBack) ClearInactivityUsers() {
	if !c.isAdmin(c.update.CallbackQuery.From.ID) {
		return
	}
	chatUserActivityData, err := c.generateUserActivityData()
	if err != nil {
		logrus.Error(err)
		return
	}
	inactiveDays, _ := c.callbackData.Data.(float64)
	for _, data := range chatUserActivityData {
		if inactiveDays < 30 {
			inactiveDays = 30
		}
		if data.inactiveDays >= int(inactiveDays) {
			res, err := c.bot.Request(tgbotapi.BanChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: c.update.Message.Chat.ID,
					UserID: data.userID,
				},
			})
			if !res.Ok {
				logrus.Errorln(res, err)
			}
			res, err = c.bot.Request(tgbotapi.UnbanChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: c.update.Message.Chat.ID,
					UserID: data.userID,
				},
			})
			if !res.Ok {
				logrus.Errorln(res, err)
			}
		}
	}
}
