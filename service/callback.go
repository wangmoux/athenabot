package service

import (
	"athenabot/db"
	"athenabot/util"
	"fmt"
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
		//msg.ShowAlert = true
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
		//msg.ShowAlert = true
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

func (c *CallBack) DeleteMarsMessage() {
	if c.callbackData.UserID != c.update.CallbackQuery.From.ID {
		msg := tgbotapi.NewCallback(c.update.CallbackQuery.ID, "你是？")
		c.sendRequestMessage(msg)
		return
	}
	msg := tgbotapi.NewCallback(c.update.CallbackQuery.ID, "嘘！")
	c.sendRequestMessage(msg)
	go func() {
		time.Sleep(time.Second * 1)
		marsMessageID := int(c.callbackData.Data.(float64))
		_, _ = c.bot.Request(tgbotapi.DeleteMessageConfig{
			ChatID:    c.chatID,
			MessageID: marsMessageID,
		})
		_, _ = c.bot.Request(tgbotapi.DeleteMessageConfig{
			ChatID:    c.chatID,
			MessageID: c.update.CallbackQuery.Message.MessageID,
		})
	}()
}

func (c *CallBack) GetUserMars() {
	//if c.callbackData.UserID != c.update.CallbackQuery.From.ID {
	//	msg := tgbotapi.NewCallback(c.update.CallbackQuery.ID, "你是？")
	//	c.sendRequestMessage(msg)
	//	return
	//}
	key := util.StrBuilder(marsTopKeyDir, util.NumToStr(c.chatID))
	userMars, err := db.RDB.ZScore(c.ctx, key, util.NumToStr(c.callbackData.UserID)).Result()
	if err != nil {
		return
	}
	if userMars > 0 {
		if c.callbackData.UserID != c.update.CallbackQuery.From.ID {
			if c.update.CallbackQuery.Message.ReplyToMessage != nil {
				marsName := c.update.CallbackQuery.Message.ReplyToMessage.From.FirstName
				msg := tgbotapi.NewCallback(c.update.CallbackQuery.ID, fmt.Sprintf("%s已经火星%d次了", marsName, int(userMars)))
				c.sendRequestMessage(msg)
			}
			return
		}
		msg := tgbotapi.NewCallback(c.update.CallbackQuery.ID, fmt.Sprintf("你已经火星%d次了", int(userMars)))
		c.sendRequestMessage(msg)
	}
}
