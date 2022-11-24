package service

import (
	"athenabot/db"
	"athenabot/util"
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"time"
)

type ChatMemberConfig struct {
	*BotConfig
	ctx context.Context
}

func NewChatMemberConfig(ctx context.Context, botConfig *BotConfig) *ChatMemberConfig {
	return &ChatMemberConfig{
		ctx:       ctx,
		BotConfig: botConfig,
	}
}

func (c *ChatMemberConfig) NewChatMember() {
	for _, user := range c.update.Message.NewChatMembers {
		logrus.Infof("new_user=%v", user.ID)
		req, err := c.bot.Request(tgbotapi.RestrictChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{
				ChatID: c.update.Message.Chat.ID,
				UserID: user.ID,
			},
		})
		if req.Ok {
			width := util.TGNameWidth(c.update.Message.From.FirstName)
			c.messageConfig.Entities = []tgbotapi.MessageEntity{
				{
					Type:   "text_mention",
					Offset: 3,
					Length: width,
					User:   &tgbotapi.User{ID: user.ID},
				},
				{
					Type:   "text_link",
					Offset: 5 + width,
					Length: 2,
					URL:    util.StrBuilder("https://t.me/", c.bot.Self.UserName, "?start=verify_", util.NumToStr(c.update.Message.Chat.ID)),
				},
			}
			c.messageConfig.Text = util.StrBuilder("欢迎 ", c.update.Message.From.FirstName, " 【点我】 完成验证就可以说话了")
			c.sendMessage()
			chatVerifyKey := util.StrBuilder(chatVerifyKeyDir, util.NumToStr(c.update.Message.Chat.ID), ":", util.NumToStr(c.update.Message.From.ID))
			err := db.RDB.Set(c.ctx, chatVerifyKey, c.update.Message.Chat.UserName, time.Second*3600).Err()
			if err != nil {
				logrus.Error(err)
			}
		} else {
			logrus.Errorln(req.ErrorCode, err)
		}
	}
}

func (c *ChatMemberConfig) newChatMemberVerify(chatID int64) {
	logrus.Infof("verify_user=%v", c.update.Message.From.ID)
	chatVerifyKey := util.StrBuilder(chatVerifyKeyDir, util.NumToStr(chatID), ":", util.NumToStr(c.update.Message.From.ID))
	res, err := db.RDB.Exists(c.ctx, chatVerifyKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	if res > 0 {
		groupName, err := db.RDB.Get(c.ctx, chatVerifyKey).Result()
		if err != nil {
			logrus.Error(err)
			return
		}
		req, err := c.bot.Request(tgbotapi.RestrictChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{
				ChatID: chatID,
				UserID: c.update.Message.From.ID,
			},
			Permissions: &tgbotapi.ChatPermissions{
				CanSendMessages:       true,
				CanSendMediaMessages:  true,
				CanSendPolls:          true,
				CanSendOtherMessages:  true,
				CanAddWebPagePreviews: true,
				CanChangeInfo:         false,
				CanInviteUsers:        true,
				CanPinMessages:        false,
			},
		})
		if req.Ok {
			c.messageConfig.Entities = []tgbotapi.MessageEntity{
				{
					Type:   "text_link",
					Offset: 7,
					Length: 2,
					URL:    util.StrBuilder("https://t.me/", groupName),
				},
			}
			c.messageConfig.Text = "你可以说话了【点我】进群"
			c.sendMessage()
			err := db.RDB.Del(c.ctx, chatVerifyKey).Err()
			if err != nil {
				logrus.Error(err)
			}
		} else {
			logrus.Errorln(req.ErrorCode, err)
		}

	}
}
