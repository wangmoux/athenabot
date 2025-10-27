package service

import (
	"athenabot/config"
	"athenabot/db"
	"athenabot/model"
	"athenabot/util"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
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
					c.sendReplyMessage()
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
			c.sendReplyMessage()
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
			c.sendReplyMessage()
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
	if c.update.Message.IsCommand() {
		return
	}
	if c.update.Message.From.ID == c.bot.Self.ID {
		return
	}
	if len(c.update.Message.Text) < 1 || len(c.update.Message.Text) > 1024 {
		return
	}
	client := &http.Client{
		Timeout: time.Second * 60,
	}
	payload, _ := json.Marshal(model.ChatGuardRequest{Prompt: c.update.Message.Text})
	req, err := http.NewRequest("POST", config.Conf.ChatGuard.GuardServerURL, bytes.NewReader(payload))
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
	if resp.StatusCode != 200 {
		logrus.Error(resp.StatusCode)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Infof("guard: %s by %s", body, c.update.Message.Text)
	guardMessage := model.ChatGuardResponse{}
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
	c.sendMessage()
}

var chatBotLimit = make(chan struct{}, 2)

var chatBotActiveLatestTime sync.Map
var chatBotPseudoRandom sync.Map
var chatBotGroupSessions sync.Map

func (c *ChatConfig) ChatBotHandler() {
	chatBotLimit <- struct{}{}
	go c.chatBotHandler(chatBotLimit)
}

func (c *ChatConfig) chatBotHandler(chatBotLimit chan struct{}) {
	defer func() { <-chatBotLimit }()
	if c.update.Message.IsCommand() {
		return
	}
	if c.update.Message.From.ID == c.bot.Self.ID {
		return
	}
	chatBotActiveLatestTime.Store(c.chatID, time.Now().Unix())
	if len(c.update.Message.Text) < 1 || len(c.update.Message.Text) > 1024 {
		return
	}
	content := &model.ChatbotContent{
		Nickname: c.update.Message.From.FirstName,
		Content:  c.update.Message.Text,
	}
	var isReplyToMessage bool
	if c.update.Message.ReplyToMessage != nil {
		if len(c.update.Message.ReplyToMessage.Text) > 0 && len(c.update.Message.ReplyToMessage.Text) < 1025 {
			isReplyToMessage = true
		}
	}
	if isReplyToMessage {
		content.ReplyToMessage = &model.ChatbotContent{
			Nickname: c.update.Message.ReplyToMessage.From.FirstName,
			Content:  c.update.Message.ReplyToMessage.Text,
		}
	}
	contents := c.addChatBotGroupSessions(content)
	for _, entity := range c.update.Message.Entities {
		if entity.Type == "mention" {
			if strings.Contains(c.update.Message.Text, fmt.Sprintf("@%s", c.bot.Self.UserName)) {
				c.chatBotMentionHandler()
			}
		}
	}
	if isReplyToMessage && config.Conf.ChatBot.EnableReply && c.update.Message.ReplyToMessage.From.ID == c.bot.Self.ID {
		c.chatBotReplyHandler()
		return
	}
	chatRandom := config.Conf.ChatBot.ChatRandom
	_pseudoRandom, ok := chatBotPseudoRandom.Load(c.chatID)
	if !ok {
		_pseudoRandom = 0
		chatBotPseudoRandom.Store(c.chatID, 0)
	}
	pseudoRandom := _pseudoRandom.(int)
	pseudoRandom += 1
	if pseudoRandom >= config.Conf.ChatBot.ChatRandom {
	} else {
		chatBotPseudoRandom.Store(c.chatID, pseudoRandom)
		if rand.Intn(chatRandom) != 0 {
			return
		}
	}
	defer func() {
		chatBotPseudoRandom.Store(c.chatID, 0)
	}()
	logrus.Infof("chat_bot request for %s", c.update.Message.Text)
	c.chat(&model.ChatbotRequest{
		Contents: contents,
		ChatType: "group_chat",
	})
}

func (c *ChatConfig) chatBotReplyHandler() {
	var contents []*model.ChatbotContent
	contents = append(contents, &model.ChatbotContent{
		IsModel:  true,
		Nickname: c.update.Message.ReplyToMessage.From.FirstName,
		Content:  c.update.Message.ReplyToMessage.Text,
	})
	contents = append(contents, &model.ChatbotContent{
		Nickname: c.update.Message.From.FirstName,
		Content:  c.update.Message.Text,
	})
	logrus.Infof("chat_bot_reply request for %s", c.update.Message.Text)
	c.chat(&model.ChatbotRequest{
		Contents: contents,
		ChatType: "group_reply",
	})
}

func (c *ChatConfig) chatBotMentionHandler() {
	var contents []*model.ChatbotContent
	content := strings.Replace(c.update.Message.Text, fmt.Sprintf("@%s", c.bot.Self.UserName), "", -1)
	contents = append(contents, &model.ChatbotContent{
		Nickname: c.update.Message.From.FirstName,
		Content:  content,
	})
	logrus.Infof("chat_bot_mention request for %s", content)
	c.chat(&model.ChatbotRequest{
		Contents: contents,
		ChatType: "group_reply",
	})
}

func (c *ChatConfig) ChatBotActiveHandler() {
	logrus.Infof("new chat_bot_active_handler: %v", c.chatID)
	ticker := time.NewTicker(time.Second * 5)
	for range ticker.C {
		_latestTime, ok := chatBotActiveLatestTime.Load(c.chatID)
		if !ok {
			continue
		}
		latestTime := _latestTime.(int64)
		if time.Now().Unix()-latestTime < int64(config.Conf.ChatBot.ActiveTiming) {
			continue
		}
		chatBotActiveLatestTime.Delete(c.chatID)
		func() {
			contents := "说点什么"
			logrus.Infof("chat_bot_active request for %s", contents)
			c.chat(&model.ChatbotRequest{
				Contents: []*model.ChatbotContent{
					{
						Content: contents,
					},
				},
				ChatType: "group_active",
			})
		}()
	}
}

func (c *ChatConfig) addChatBotGroupSessions(content *model.ChatbotContent) []*model.ChatbotContent {
	contents := c.getChatBotGroupSessions()
	contents = append(contents, content)
	chatBotGroupSessionsLen := len(contents)
	if chatBotGroupSessionsLen > 10 {
		contents = contents[chatBotGroupSessionsLen-10:]
	}
	chatBotGroupSessions.Store(c.chatID, contents)
	return contents
}

func (c *ChatConfig) getChatBotGroupSessions() []*model.ChatbotContent {
	_contents, ok := chatBotGroupSessions.Load(c.chatID)
	if !ok {
		_contents = []*model.ChatbotContent{}
	}
	return _contents.([]*model.ChatbotContent)
}

func (c *ChatConfig) chat(cr *model.ChatbotRequest) {
	client := &http.Client{
		Timeout: time.Second * 60,
	}
	payload, _ := json.Marshal(cr)
	req, err := http.NewRequest("POST", config.Conf.ChatBot.ChatBotServerURL, bytes.NewReader(payload))
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
	if resp.StatusCode != 200 {
		logrus.Error(resp.StatusCode)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Infof("chat_bot response: %s by %s", body, cr.ChatType)
	chatbotMessage := model.ChatbotResponse{}
	_ = json.Unmarshal(body, &chatbotMessage)
	c.messageConfig.Text = chatbotMessage.BotMessage
	c.sendMessage()
	c.addChatBotGroupSessions(&model.ChatbotContent{
		IsModel:  true,
		Nickname: c.bot.Self.FirstName,
		Content:  chatbotMessage.BotMessage,
	})
}
