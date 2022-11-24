package service

import (
	"github.com/sirupsen/logrus"
	"time"
)

type Chat struct {
	*BotConfig
}

func NewChat(botConfig *BotConfig) *Chat {
	return &Chat{BotConfig: botConfig}
}

type chatLimit struct {
	userID    int64
	count     int
	timestamp int64
}

func (c *Chat) ChatLimit() {
	timestamp := time.Now().Unix()
	if group, ok := groupsChatLimit[c.update.Message.Chat.ID]; ok {
		if group.userID == c.update.Message.From.ID {
			group.count += 1
			if group.count >= 10 {
				if timestamp-group.timestamp < 30 {
					c.messageConfig.Text = "多吃饭少说话"
					c.sendMessage()
					logrus.Infof("chat_limit:%+v", group)
				}
				group.timestamp = timestamp
			}
			return
		}
	}
	groupsChatLimit[c.update.Message.Chat.ID] = &chatLimit{
		userID:    c.update.Message.From.ID,
		count:     1,
		timestamp: timestamp,
	}

}
