package app

import (
	"athenabot/config"
	"athenabot/controller"
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

func RunBot() {
	bot, err := tgbotapi.NewBotAPI(config.Conf.BotToken)
	if err != nil {
		logrus.Panic(err)
	}
	bot.Debug = false
	logrus.Infof("bot=%v", bot.Self.UserName)
	switch config.Conf.UpdatesType {
	case "webhook":
		logrus.Info("updates_type=webhook")
		updatesHandler(NewWebhook(bot))
	default:
		logrus.Info("updates_type=polling")
		updatesHandler(NewPolling(bot))
	}
}

func updatesHandler(client Client) {
	for update := range client.Channel() {
		if update.Message != nil {
			if _chatCh, ok := chatMap.Load(update.Message.Chat.ID); ok {
				if chatCh, _ok := _chatCh.(chatChannel); _ok {
					chatCh <- update
					continue
				}
			}
			logrus.Infof("new chat_handler=%v", update.Message.Chat.ID)
			updateCh := make(chatChannel, 10)
			chatMap.Store(update.Message.Chat.ID, updateCh)
			go chatHandler(updateCh, client.GetBot())
			updateCh <- update
		}
	}
}

var chatMap sync.Map

type chatChannel chan tgbotapi.Update

func chatHandler(ch chatChannel, bot *tgbotapi.BotAPI) {
	var chatID int64
	var ttl int64 = 600
	for {
		select {
		case update := <-ch:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			controller.Controller(ctx, cancel, bot, update)
			chatID = update.Message.Chat.ID
			if update.Message.Chat.Type == "private" {
				ttl = 60
			} else {
				ttl = 600
			}
			//go debug(bot, update)
		case <-time.After(time.Second * time.Duration(ttl)):
			logrus.Infof("close chat_handler=%v", chatID)
			chatMap.Delete(chatID)
			close(ch)
			return
		}
	}
}
