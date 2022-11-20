package app

import (
	"athenabot/config"
	"athenabot/controller"
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
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
			if _, ok := chatMap[update.Message.Chat.ID]; ok {
				chatMap[update.Message.Chat.ID] <- update
				continue
			}
			logrus.Infof("new chat_handler=%v", update.Message.Chat.ID)
			updateCh := make(chatChannel, 10)
			chatMap[update.Message.Chat.ID] = updateCh
			go chatHandler(chatMap[update.Message.Chat.ID], client.GetBot())
			chatMap[update.Message.Chat.ID] <- update
		}
	}
}

var chatMap = make(map[int64]chatChannel)

type chatChannel chan tgbotapi.Update

func chatHandler(ch chatChannel, bot *tgbotapi.BotAPI) {
	var chatID int64
	var ttl int64 = 600
	for {
		select {
		case update := <-ch:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			controller.Controller(ctx, cancel, bot, &update)
			chatID = update.Message.Chat.ID
			if update.Message.Chat.Type == "private" {
				ttl = 60
			} else {
				ttl = 600
			}
		case <-time.After(time.Second * time.Duration(ttl)):
			logrus.Infof("close chat_handler=%v", chatID)
			delete(chatMap, chatID)
			close(ch)
			return
		}
	}
}
