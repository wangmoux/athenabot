package service

import (
	"athenabot/db"
	"athenabot/util"
	"context"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type ChatConfig struct {
	*BotConfig
}

func NewChatConfig(botConfig *BotConfig) *ChatConfig {
	return &ChatConfig{BotConfig: botConfig}
}

type chatLimit struct {
	userID    int64
	count     int
	timestamp int64
}

func (c *ChatConfig) ChatLimit() {
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

func (c *ChatConfig) ChatStore48hMessage() {
	chat48hMessageKey := util.StrBuilder(chat48hMessageDir, util.NumToStr(c.update.Message.Chat.ID), ":", util.NumToStr(c.update.Message.From.ID))
	err := db.RDB.HMSet(c.ctx, chat48hMessageKey, c.update.Message.MessageID, time.Now().Unix()).Err()
	if err != nil {
		logrus.Error(err)
	}
	err = db.RDB.Expire(c.ctx, chat48hMessageKey, time.Second*172800).Err()
	if err != nil {
		logrus.Error(err)
	}
}

func (c *ChatConfig) Delete48hMessageCronHandler() {
	logrus.Infof("new delete_48h_message_cron_handler:%v", c.update.Message.Chat.ID)
	chat48hMessageDeleteCrontabKey := util.StrBuilder(chat48hMessageDeleteCrontabDir, util.NumToStr(c.update.Message.Chat.ID))
	ticker := time.NewTicker(time.Second * 600)
	for range ticker.C {
		crontabUsers, err := db.RDB.SMembers(context.Background(), chat48hMessageDeleteCrontabKey).Result()
		if err != nil {
			logrus.Error(err)
		}
		for _, userID := range crontabUsers {
			chat48hMessageKey := util.StrBuilder(chat48hMessageDir, util.NumToStr(c.update.Message.Chat.ID), ":", userID)
			isMessageIDs, err := db.RDB.Exists(context.Background(), chat48hMessageKey).Result()
			if err != nil {
				logrus.Error(err)
				continue
			}
			if isMessageIDs > 0 {
				messageIDs, err := db.RDB.HGetAll(context.Background(), chat48hMessageKey).Result()
				if err != nil {
					logrus.Error(err)
					continue
				}
				for k, v := range messageIDs {
					messageID, err := strconv.Atoi(k)
					if err != nil {
						continue
					}
					messageTime, err := strconv.Atoi(v)
					if err != nil {
						continue
					}
					t := time.Now().Unix() - int64(messageTime)
					if t > 172800 || t < 169200 {
						continue
					}
					c.addDeleteMessageQueue(0, messageID)
					err = db.RDB.HDel(context.Background(), chat48hMessageKey, util.NumToStr(messageID)).Err()
					if err != nil {
						logrus.Error(err)
					}
				}
			}
		}
	}
}
