package controller

import (
	"athenabot/config"
	"athenabot/model"
	"athenabot/service"
	"athenabot/util"
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"sync"
)

func Controller(ctx context.Context, bot *tgbotapi.BotAPI, uc *model.UpdateConfig) {
	update := uc.Update
	logrus.DebugFn(util.LogMarshalFn(update))
	if len(uc.UpdateType) == 0 || uc.ChatID == 0 {
		logrus.Debugf("unsupported update:%s", util.LogMarshal(update))
		return
	}
	c := service.NewBotConfig(ctx, bot, update)
	c.SetChatID(uc.ChatID)

	switch uc.UpdateType {
	case model.InlineType:
		service.NewInlineQueryConfig(c).HandleInlineQuery()
	case model.CallbackType:
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
		case "delete-mars-msg":
			cb.DeleteMarsMessage()
		case "get-user-mars":
			cb.GetUserMars()
		}
	case model.MessageType:
		switch uc.ChatType {
		case model.PrivateType:
			if config.Conf.Modules.EnablePrivateCommand && c.IsCommand() {
				service.NewCommandConfig(c).InPrivateCommands()
				return
			}
		default:
			if c.IsGroupWhitelist(update.Message.Chat.UserName) {
				func() {
					if ch, ok := asyncMap.Load(uc.ChatID); ok {
						ch.(asyncChannel) <- c
						return
					}
					logrus.Infof("new async_controller:%v", update.Message.Chat.ID)
					ch := make(asyncChannel, 10)
					asyncMap.Store(uc.ChatID, ch)
					go asyncController(ctx, ch, uc.ChatID)
					ch <- c
				}()
				if config.Conf.Modules.EnableMars && c.IsEnableChatService("chat_mars") && c.IsMarsWhitelist(update.Message.Chat.UserName) {
					if len(update.Message.Photo) > 0 {
						service.NewMarsConfig(c).HandlePhoto()
						return
					}
					if update.Message.Video != nil {
						service.NewMarsConfig(c).HandleVideo()
						return
					}

				}
				if config.Conf.Modules.EnableCommand && c.IsCommand() {
					service.NewCommandConfig(c).InCommands()
					return
				}
			}
		}
	}
}

type asyncChannel chan *service.BotConfig

var asyncMap = sync.Map{}

func asyncController(ctx context.Context, ch asyncChannel, chatID int64) {
	var asyncControllerOnce sync.Once
	for {
		select {
		case c := <-ch:
			cc := service.NewChatConfig(c)
			asyncControllerOnce.Do(func() {
				go c.DeleteMessageCronHandler(ctx)
				if c.IsEnableChatService("clear_my_48h_message") {
					go cc.Delete48hMessageCronHandler(ctx)
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
		case <-ctx.Done():
			asyncMap.Delete(chatID)
			logrus.Infof("async_controller exited:%v", chatID)
			return
		}
	}
}
