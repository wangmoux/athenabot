package controller

import (
	"athenabot/config"
	"athenabot/service"
	"athenabot/util"
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func isInWhitelist(ChatUserName string, chatID int64) bool {
	if len(ChatUserName) > 1 {
		if _, ok := config.WhitelistUsernameMap[ChatUserName]; ok {
			return true
		}
	}
	if _, ok := config.WhitelistIdMap[chatID]; ok {
		return true
	}
	return false
}

func Controller(ctx context.Context, cancel context.CancelFunc, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	logrus.DebugFn(util.LogMarshalFn(update))
	c := service.NewBotConfig(ctx, cancel, bot, update)
	if config.Conf.DisableWhitelist || isInWhitelist(update.Message.Chat.UserName, update.Message.Chat.ID) {
		if config.Conf.Modules.EnableChatLimit {
			go service.NewChat(c).ChatLimit()
		}
		if config.Conf.Modules.EnableMars {
			if len(update.Message.Photo) > 0 {
				service.NewMarsConfig(ctx, c).HandlePhoto()
				return
			}
		}
		if update.Message.Video != nil {
			service.NewMarsConfig(ctx, c).HandleVideo()
			return
		}
		if config.Conf.Modules.EnableCommand && update.Message.IsCommand() {
			service.NewCommandConfig(ctx, c).InCommands()
			return
		}

		if config.Conf.Modules.EnableMemberVerify && len(update.Message.NewChatMembers) > 0 {
			service.NewChatMemberConfig(ctx, c).NewChatMember()
			return
		}
	}
	if update.Message.Chat.Type == "private" {
		if config.Conf.Modules.EnablePrivateCommand && update.Message.IsCommand() {
			service.NewCommandConfig(ctx, c).InPrivateCommands()
			return

		}
	}
}
