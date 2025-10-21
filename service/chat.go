package service

import (
	"athenabot/config"
	"athenabot/db"
	"athenabot/model"
	"athenabot/util"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-redis/redis/v8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
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
	if len(c.update.Message.Text) == 0 {
		return
	}
	timestamp := time.Now().Unix()
	if group, ok := groupsChatLimit[c.chatID]; ok {
		if group.userID == c.update.Message.From.ID {
			group.count += 1
			if group.count >= 10 {
				if timestamp-group.timestamp < 30 {
					c.messageConfig.Text = "多吃饭少说话"
					c.sendCommandMessage()
					logrus.Infof("chat_limit:%+v", group)
				}
				group.timestamp = timestamp
			}
			return
		}
	}
	groupsChatLimit[c.chatID] = &chatLimit{
		userID:    c.update.Message.From.ID,
		count:     1,
		timestamp: timestamp,
	}

}

func (c *ChatConfig) ChatStore48hMessage() {
	chat48hMessageKey := util.StrBuilder(chat48hMessageDir, util.NumToStr(c.chatID), ":", util.NumToStr(c.update.Message.From.ID))
	err := db.RDB.HMSet(c.ctx, chat48hMessageKey, c.update.Message.MessageID, time.Now().Unix()).Err()
	if err != nil {
		logrus.Error(err)
	}
	err = db.RDB.Expire(c.ctx, chat48hMessageKey, time.Second*172800).Err()
	if err != nil {
		logrus.Error(err)
	}
}

func (c *ChatConfig) Delete48hMessageCronHandler(ctx context.Context) {
	logrus.Infof("new delete_48h_message_cron_handler:%v", c.chatID)
	chat48hMessageDeleteCrontabKey := util.StrBuilder(chat48hMessageDeleteCrontabDir, util.NumToStr(c.chatID))
	ticker := time.NewTicker(time.Second * 600)
	for {
		select {
		case <-ticker.C:
			crontabUsers, err := db.RDB.SMembers(context.Background(), chat48hMessageDeleteCrontabKey).Result()
			if err != nil {
				logrus.Error(err)
			}
			for _, userID := range crontabUsers {
				chat48hMessageKey := util.StrBuilder(chat48hMessageDir, util.NumToStr(c.chatID), ":", userID)
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
		case <-ctx.Done():
			logrus.Infof("delete_48h_message_cron_handler exited:%v", c.chatID)
			return
		}
	}
}

func (c *ChatConfig) chatUserprofileWatchHandler(key, currentName, prefix string, userID int64) {
	keyExists, err := db.RDB.Exists(c.ctx, key).Result()
	if err != nil {
		logrus.Error(err)
	}
	if keyExists > 0 {
		latestName, err := db.RDB.ZRevRange(c.ctx, key, 0, 0).Result()
		if err != nil {
			logrus.Error(err)
		}
		if latestName[0] != currentName {
			c.messageConfig.Text = util.StrBuilder(prefix, " ", latestName[0], " -> ", currentName)
			c.messageConfig.Entities = []tgbotapi.MessageEntity{{
				Type:   "text_mention",
				Offset: 0,
				Length: util.TGNameWidth(prefix),
				User:   &tgbotapi.User{ID: userID},
			}}
			c.sendCommandMessage()
			err = db.RDB.ZAdd(c.ctx, key, &redis.Z{
				Score:  float64(time.Now().Unix()),
				Member: currentName,
			}).Err()
			if err != nil {
				logrus.Error(err)
			}
		}
	} else {
		err = db.RDB.ZAdd(c.ctx, key, &redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: currentName,
		}).Err()
		if err != nil {
			logrus.Error(err)
		}
	}

	err = db.RDB.Expire(c.ctx, key, time.Second*time.Duration(config.Conf.KeyTTL)).Err()
	if err != nil {
		logrus.Error(err)
	}
}

func (c *ChatConfig) ChatUserprofileWatch() {
	usernameKey := util.StrBuilder(chatUserprofileWatchDir, util.NumToStr(c.chatID), ":",
		util.NumToStr(c.update.Message.From.ID), ":", "username")
	currentUsername := c.update.Message.From.UserName
	if len(currentUsername) == 0 {
		currentUsername = "unknown"
	}
	c.chatUserprofileWatchHandler(usernameKey, currentUsername, "用户名更改", c.update.Message.From.ID)

	fullNameKey := util.StrBuilder(chatUserprofileWatchDir, util.NumToStr(c.chatID), ":",
		util.NumToStr(c.update.Message.From.ID), ":", "full_name")
	currentFullName := c.update.Message.From.FirstName + c.update.Message.From.LastName
	c.chatUserprofileWatchHandler(fullNameKey, currentFullName, "昵称更改", c.update.Message.From.ID)
}

func (c *ChatConfig) ChatBlacklistHandler() {
	if c.update.Message.IsCommand() && c.isAdmin(c.update.Message.From.ID) {
		return
	}
	key := util.StrBuilder(chatBlacklistDir, util.NumToStr(c.chatID))
	keywords, err := db.RDB.ZRevRange(context.Background(), key, 0, -1).Result()
	if err != nil {
		logrus.Error(err)
	}
	var text string
	for _, i := range c.update.Message.Text {
		if unicode.IsSpace(i) || unicode.IsPunct(i) {
			continue
		}
		text += string(i)
	}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			c.messageConfig.Text = "你的发言涉嫌违反群规"
			c.sendCommandMessage()
			break
		}
	}
}

func (c *ChatConfig) ChatUserActivity() {
	key := util.StrBuilder(chatUserActivityDir, util.NumToStr(c.chatID))
	z := &redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: c.update.Message.From.ID,
	}
	err := db.RDB.ZAdd(c.ctx, key, z).Err()
	if err != nil {
		logrus.Error(err)
	}
}

var chatGuardLimit = make(chan struct{}, 2)

func (c *ChatConfig) ChatGuardHandler() {
	chatGuardLimit <- struct{}{}
	go c.chatGuardHandler(chatGuardLimit)
}

func (c *ChatConfig) chatGuardHandler(chatGuardLimit chan struct{}) {
	defer func() { <-chatGuardLimit }()
	if c.update.Message.From.IsBot || c.update.Message.IsCommand() {
		return
	}
	if len(c.update.Message.Text) < 6 || len(c.update.Message.Text) > 256 {
		return
	}
	client := http.Client{}
	payload := strings.NewReader(`{"prompt": "` + c.update.Message.Text + `"}`)
	req, err := http.NewRequest("POST", config.Conf.ChatGuard.GuardServerURL, payload)
	if err != nil {
		logrus.Error(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Infof("guard:%s by %s", string(body), c.update.Message.Text)
	guardMessage := model.ChatGuardMessage{}
	_ = json.Unmarshal(body, &guardMessage)
	if guardMessage.SafeLabel == "" || guardMessage.SafeLabel == "Safe" {
		return
	}
	if config.Conf.ChatGuard.SafeLabel == "Unsafe" && guardMessage.SafeLabel == "Controversial" {
		return
	}
	if len(guardMessage.Categories) == 0 {
		return
	}
	if _, ok := config.CategoriesFilter[guardMessage.Categories[0]]; !ok {
		return
	}

	var categories string
	for i := range guardMessage.Categories {
		if i == len(guardMessage.Categories)-1 {
			categories += guardMessage.Categories[i]
			break
		}
		categories += guardMessage.Categories[i] + ", "
	}
	fullName := c.update.Message.From.FirstName + c.update.Message.From.LastName
	reasonConfirmButton := tgbotapi.NewInlineKeyboardButtonData("删除", generateCallbackData("delete-chat-msg-clean", c.update.Message.From.ID, c.update.Message.MessageID))
	restrictConfirmButton := tgbotapi.NewInlineKeyboardButtonData("禁言", generateCallbackData("restrict-user", c.update.Message.From.ID, fmt.Sprintf("%s", fullName)))
	// banConfirmButton := tgbotapi.NewInlineKeyboardButtonData("封禁", generateCallbackData("ban-user", c.update.Message.From.ID, fmt.Sprintf("%s", fullName)))

	replyMarkup := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{reasonConfirmButton, restrictConfirmButton /*banConfirmButton*/},
	)
	c.messageConfig.ReplyMarkup = replyMarkup
	prefix := "聊天卫士检测到敏感话题"
	c.messageConfig.Text = fmt.Sprintf("%s %s\n%s  “%s”", prefix, categories, fullName, c.update.Message.Text)
	humanChatID := c.chatID - c.chatID - c.chatID - 1000000000000
	LatestMarsMessage := util.StrBuilder("https://t.me/c/", util.NumToStr(humanChatID), "/", util.NumToStr(c.update.Message.MessageID))
	fullNameOffset := util.TGNameWidth(fmt.Sprintf("%s %s\n", prefix, categories))
	c.messageConfig.Entities = []tgbotapi.MessageEntity{
		{
			Type:   "text_mention",
			Offset: fullNameOffset,
			Length: util.TGNameWidth(fullName),
			User:   &tgbotapi.User{ID: c.update.Message.From.ID},
		},
		{
			Type:   "text_link",
			URL:    LatestMarsMessage,
			Offset: fullNameOffset + util.TGNameWidth(fullName) + 2,
			Length: util.TGNameWidth(c.update.Message.Text) + 2,
		}}
	c.update.Message.MessageID = -1
	c.sendCommandMessage()
}
