package service

import (
	"athenabot/config"
	"athenabot/util"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type InlineQueryConfig struct {
	*BotConfig
}

func NewInlineQueryConfig(botConfig *BotConfig) *InlineQueryConfig {
	return &InlineQueryConfig{
		BotConfig: botConfig,
	}
}

func (c *InlineQueryConfig) HandleInlineQuery() {
	inlineConfig := tgbotapi.InlineConfig{
		InlineQueryID: c.update.InlineQuery.ID,
		Results:       inlineQueryResultArticles,
	}
	c.sendRequestMessage(inlineConfig)
}

func init() {
	for i, result := range config.Conf.InlineQueryResultArticles {
		inlineQueryResultArticles = append(inlineQueryResultArticles, tgbotapi.NewInlineQueryResultArticle(util.NumToStr(i), result.Title, result.MessageText))
	}
}
