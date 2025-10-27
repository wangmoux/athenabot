package app

import (
	"athenabot/config"
	"athenabot/controller"
	"athenabot/model"
	"context"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func RunBot() {
	bot, err := tgbotapi.NewBotAPI(config.Conf.BotToken)
	if err != nil {
		logrus.Error(err)
		RunBot()
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
		go debugMessage(&update)
		uc := &model.UpdateConfig{
			Update: update,
		}

		if update.InlineQuery != nil {
			uc.ChatID = update.InlineQuery.From.ID
			uc.UpdateType = model.InlineType
			uc.ChatType = update.InlineQuery.ChatType
		}

		if update.CallbackQuery != nil {
			uc.ChatID = update.CallbackQuery.Message.Chat.ID
			uc.UpdateType = model.CallbackType
			uc.ChatType = update.CallbackQuery.Message.Chat.Type
		}

		if update.Message != nil {
			uc.ChatID = update.Message.Chat.ID
			uc.UpdateType = model.MessageType
			uc.ChatType = update.Message.Chat.Type
		}

		if chatCh, ok := chatMap.Load(uc.ChatID); ok {
			chatCh.(chatChannel) <- uc
			continue
		}
		logrus.Infof("new chat_handler:%v", uc.ChatID)
		updateCh := make(chatChannel, 10)
		chatMap.Store(uc.ChatID, updateCh)

		timeout := time.Hour * 48
		if uc.ChatType == model.PrivateType || len(uc.UpdateType) == 0 {
			timeout = time.Second * 60
		}
		ctx, cancel := context.WithCancel(context.Background())
		go chatHandler(ctx, cancel, updateCh, client.GetBot(), uc.ChatID, timeout)
		updateCh <- uc
	}
}

var chatMap sync.Map

type chatChannel chan *model.UpdateConfig

func chatHandler(ctx context.Context, cancel context.CancelFunc, ch chatChannel, bot *tgbotapi.BotAPI, chatID int64, timeout time.Duration) {
	doneTime := time.AfterFunc(timeout, cancel)
	for {
		select {
		case uc := <-ch:
			controller.Controller(ctx, bot, uc)
			doneTime.Reset(timeout)
			//go debug(bot, update)
		case <-ctx.Done():
			chatMap.Delete(chatID)
			logrus.Infof("chat_handler exited:%v", chatID)
			return
		}
	}
}
