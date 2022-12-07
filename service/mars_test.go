package service

import (
	"athenabot/config"
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestMarsConfig_HandlePhoto(t *testing.T) {
	bot, err := tgbotapi.NewBotAPI(config.Conf.BotToken)
	if err != nil {
		logrus.Panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	update := tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			Chat: &tgbotapi.Chat{
				ID: -1001546229241,
			},
			From: &tgbotapi.User{},
			Photo: []tgbotapi.PhotoSize{
				{
					FileID: "AgACAgUAAx0CXCmV-QACHnVjkDuJNyIAARqPztL5JLRfKfgQlqUAAsuxMRsJHXhUv3xtBLHNHRUBAAMCAAN4AAMrBA",
				},
			},
		},
	}
	m := NewMarsConfig(ctx, NewBotConfig(ctx, cancel, bot, update))
	m.HandlePhoto()
}
