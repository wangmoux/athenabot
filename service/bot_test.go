package service

import (
	"athenabot/config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"testing"
)

func TestBotConfig_SendMessage(t *testing.T) {
	c := &BotConfig{
		update: &tgbotapi.Update{
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{
					ID: config.Conf.Whitelist.GroupsId[0],
				},
			},
		},
		bot: nil,
	}
	c.bot, _ = tgbotapi.NewBotAPI(config.Conf.BotToken)
	c.messageConfig.Text = "测试消息"
	c.messageConfig.ChatID = c.update.Message.Chat.ID
	c.sendMessage()
}
