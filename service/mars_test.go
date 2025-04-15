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
	defer cancel()
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
	m := NewMarsConfig(NewBotConfig(ctx, bot, update))
	m.HandlePhoto()
}

func TestMarsConfig_handleImageDoc(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	update := tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			Chat: &tgbotapi.Chat{
				ID: -1001546229241,
			},
			From: &tgbotapi.User{},
		},
	}
	m := NewMarsConfig(NewBotConfig(ctx, &tgbotapi.BotAPI{}, update))
	imagePhrases := []string{"用户", "你去月球了", "评论", "1000", "2022-11.11 12:11"}
	m.handleImageDoc(imagePhrases, 123456789)
}
