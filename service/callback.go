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
		msg := tgbotapi.NewCallback(c.update.CallbackQuery.ID, "你是？")
		msg.ShowAlert = true
		c.sendRequestMessage(msg)
		return
	}
	commandMessageID, _ := c.callbackData.Data.(float64)
	chat48hMessageKey := util.StrBuilder(chat48hMessageDir, util.NumToStr(c.chatID), ":", util.NumToStr(c.callbackData.UserID))
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
	_, _ = c.bot.Request(tgbotapi.DeleteMessageConfig{
		ChatID:    c.chatID,
		MessageID: c.update.CallbackQuery.Message.MessageID,
	})
	_, _ = c.bot.Request(tgbotapi.DeleteMessageConfig{
		ChatID:    c.chatID,
		MessageID: int(commandMessageID),
	})
}

func (c *CallBack) ClearInactivityUsers() {
	if !c.isAdminCanRestrictMembers(c.update.CallbackQuery.From.ID) {
		msg := tgbotapi.NewCallback(c.update.CallbackQuery.ID, "你是？")
		msg.ShowAlert = true
		c.sendRequestMessage(msg)
		return
	}
	chatUserActivityData, err := c.generateUserActivityData()
	if err != nil {
		logrus.Error(err)
		return
	}
	inactiveDays, _ := c.callbackData.Data.(float64)
	if inactiveDays < 30 {
		inactiveDays = 30
	}
	for _, data := range chatUserActivityData {
		if data.inactiveDays >= int(inactiveDays) {
			res, err := c.bot.Request(tgbotapi.BanChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: c.chatID,
					UserID: data.userID,
				},
			})
			if !res.Ok {
				logrus.Errorln(res, err)
			}
			res, err = c.bot.Request(tgbotapi.UnbanChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: c.chatID,
					UserID: data.userID,
				},
			})
			if !res.Ok {
				logrus.Errorln(res, err)
			}
		}
	}
}
