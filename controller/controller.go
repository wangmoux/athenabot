package controller

import (
	"athenabot/config"
	"athenabot/service"
	"athenabot/util"
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"sync"
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
	if update.CallbackQuery != nil {
		callbackData, err := service.ParseCallbackData(update.CallbackQuery.Data)
		if err != nil {
			return
		}
		logrus.Infof("callback:%+v", callbackData)
		cb := service.NewCallBack(c, callbackData)
		switch callbackData.Command {
		case "clear-msg":
			cb.ClearMy48hMessage()
		case "clear-users":
			cb.ClearInactivityUsers()
		}
		return
	}

	if update.Message != nil {
		switch update.Message.Chat.Type {
		case "supergroup", "group":
			if config.Conf.DisableWhitelist || isInWhitelist(update.Message.Chat.UserName, update.Message.Chat.ID) {
				func() {
					if ch, ok := asyncMap[update.Message.Chat.ID]; ok {
						ch <- c
						return
					}
					logrus.Infof("new async_controller:%v", update.Message.Chat.ID)
					ch := make(asyncChannel, 10)
					asyncMap[update.Message.Chat.ID] = ch
					go asyncController(ch)
					ch <- c
				}()
				if config.Conf.Modules.EnableMars && c.IsEnableChatService("chat_mars") {
					if len(update.Message.Photo) > 0 {
						service.NewMarsConfig(c).HandlePhoto()
						return
					}
					if update.Message.Video != nil {
						service.NewMarsConfig(c).HandleVideo()
						return
					}

				}
				if config.Conf.Modules.EnableCommand && update.Message.IsCommand() {
					service.NewCommandConfig(c).InCommands()
					return
				}
			}
		case "private":
			if config.Conf.Modules.EnablePrivateCommand && update.Message.IsCommand() {
				service.NewCommandConfig(c).InPrivateCommands()
				return
			}
		}
	}
}

type asyncChannel chan *service.BotConfig

var asyncMap = make(map[int64]asyncChannel)

func asyncController(ch asyncChannel) {
	var asyncControllerOnce sync.Once
	for {
		select {
		case c := <-ch:
			cc := service.NewChatConfig(c)
			asyncControllerOnce.Do(func() {
				go c.DeleteMessageCronHandler()
				if c.IsEnableChatService("clear_my_48h_message") {
					go cc.Delete48hMessageCronHandler()
				}
			})
			if c.IsEnableChatService("chat_member_verify") {
				cc.NewChatMemberVerify()
			}
			if c.IsEnableChatService("clear_my_48h_message") {
				cc.ChatStore48hMessage()
			}
			if c.IsEnableChatService("chat_userprofile_watch") {
				cc.ChatUserprofileWatch()
			}
			if c.IsEnableChatService("chat_limit") {
				cc.ChatLimit()
			}
			if c.IsEnableChatService("chat_blacklist") {
				cc.ChatBlacklistHandler()
			}
			if c.IsEnableChatService("chat_user_activity") {
				cc.ChatUserActivity()
			}
		}
	}
}
