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
	logrus.Infof("bot:%v", bot.Self.UserName)
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
		if update.CallbackQuery != nil {
			update.Message = update.CallbackQuery.Message
		}
		if update.Message != nil {
			if chatCh, ok := chatMap[update.Message.Chat.ID]; ok {
				chatCh <- update
				continue
			}
			logrus.Infof("new chat_handler:%v", update.Message.Chat.ID)
			updateCh := make(chatChannel, 10)
			chatMap[update.Message.Chat.ID] = updateCh
			go chatHandler(updateCh, client.GetBot())
			updateCh <- update
		}
	}
}

var chatMap = make(map[int64]chatChannel)

type chatChannel chan tgbotapi.Update

func chatHandler(ch chatChannel, bot *tgbotapi.BotAPI) {
	for {
		select {
		case update := <-ch:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
			controller.Controller(ctx, cancel, bot, update)
			//go debug(bot, update)
		}
	}
}
