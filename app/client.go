package app

import (
	"athenabot/config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"net/http"
)

type Client interface {
	Channel() tgbotapi.UpdatesChannel
	GetBot() *tgbotapi.BotAPI
}

type Polling struct {
	bot *tgbotapi.BotAPI
}

func NewPolling(bot *tgbotapi.BotAPI) *Polling {
	return &Polling{bot: bot}
}

type Webhook struct {
	bot *tgbotapi.BotAPI
}

func NewWebhook(bot *tgbotapi.BotAPI) *Webhook {
	return &Webhook{bot: bot}
}

func (c Polling) Channel() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := c.bot.GetUpdatesChan(u)
	return updates
}

func (c Polling) GetBot() *tgbotapi.BotAPI {
	return c.bot
}

func (c Webhook) Channel() tgbotapi.UpdatesChannel {
	setWebhook()
	updates := c.bot.ListenForWebhook("/" + config.Conf.Webhook.Token)
	go func() {
		err := http.ListenAndServe(config.Conf.Webhook.ListenAddr, nil)
		if err != nil {
			logrus.Error(err)
		}
	}()
	return updates
}

func (c Webhook) GetBot() *tgbotapi.BotAPI {
	return c.bot
}
