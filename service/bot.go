package service

import (
	"athenabot/config"
	"athenabot/db"
	"athenabot/util"
	"context"
	"encoding/json"
	"github.com/bitly/go-simplejson"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"sync"
	"time"
)

type BotConfig struct {
	update                   tgbotapi.Update
	bot                      *tgbotapi.BotAPI
	messageConfig            tgbotapi.MessageConfig
	ctx                      context.Context
	cancel                   context.CancelFunc
	botMessageCleanCountdown int
	botMessageID             int
}

func NewBotConfig(ctx context.Context, cancel context.CancelFunc, bot *tgbotapi.BotAPI, update tgbotapi.Update) *BotConfig {
	botConfig := &BotConfig{
		ctx:    ctx,
		cancel: cancel,
		update: update,
		bot:    bot,
		messageConfig: tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID:           update.Message.Chat.ID,
				ReplyToMessageID: update.Message.MessageID,
			},
			Text:     "无言以对",
			Entities: []tgbotapi.MessageEntity{},
		},
	}
	return botConfig
}

func (c *BotConfig) isCloseWork() bool {
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

func (c *BotConfig) sendMessage() {
	msg := tgbotapi.NewMessage(c.update.Message.Chat.ID, c.messageConfig.Text)
	msg = c.messageConfig
	req, err := c.bot.Send(msg)
	if err != nil {
		logrus.Error(err)
	} else {
		c.botMessageID = req.MessageID
	}
	logrus.Debugf("send_msg:%v", util.LogMarshal(msg))
}

func (c *BotConfig) isAdmin(userID int64) bool {
	logrus.Debugf("administrators:%v", util.LogMarshal(groupsAdministratorsCache))
	if group, ok := groupsAdministratorsCache[c.update.Message.Chat.ID]; ok {
		if _, _ok := group[userID]; _ok {
			return true
		}
	} else {
		req, err := c.bot.Request(tgbotapi.ChatAdministratorsConfig{
			ChatConfig: tgbotapi.ChatConfig{
				ChatID: c.update.Message.Chat.ID,
			},
		})

		if !req.Ok {
			logrus.Errorln(req.ErrorCode, err)
			return false
		}

		resJson := &simplejson.Json{}
		resJson, _ = simplejson.NewJson(req.Result)
		chatAdministrators := resJson.MustArray()
		_group := make(groupAdministratorsCache)
		for i := range chatAdministrators {
			id := resJson.GetIndex(i).Get("user").Get("id").MustInt64()
			_group[id] = 0
		}
		for _, i := range config.Conf.SudoAdmins {
			_group[i] = 0
		}
		groupsAdministratorsCache[c.update.Message.Chat.ID] = _group
		if _, _ok := _group[userID]; _ok {
			return true
		}
	}
	return false
}

func (c *BotConfig) getUserNameCache(wg *sync.WaitGroup, userID int64) {
	defer wg.Done()
	if _, ok := userNameCache[userID]; !ok {
		req, err := c.bot.Request(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: c.update.Message.Chat.ID,
				UserID: userID,
			},
		})
		if !req.Ok {
			logrus.Errorln(req.ErrorCode, err)
			if strings.Contains(req.Description, "user not found") {
				unknownUserCache[userID] = 0
			}
			return
		}
		userJson := &simplejson.Json{}
		userJson, _ = simplejson.NewJson(req.Result)
		defer userNameCacheLock.Unlock()
		userNameCacheLock.Lock()
		userNameCache[userID] = userJson.Get("user").Get("first_name").MustString()
	}
}

func (c *BotConfig) DeleteMessageCronHandler() {
	logrus.Infof("new delete_message_cron_handler:%v", c.update.Message.Chat.ID)
	deleteMessageKey := util.StrBuilder(deleteMessageKeyDir, util.NumToStr(c.update.Message.Chat.ID))
	ticker := time.NewTicker(time.Second * 5)
	for range ticker.C {
		res, err := db.RDB.HGetAll(context.Background(), deleteMessageKey).Result()
		if err != nil {
			logrus.Error(err)
			continue
		}
		wg := new(sync.WaitGroup)
		for k, v := range res {
			time.Sleep(time.Millisecond * 10)
			wg.Add(1)
			go func(k, v string, wg *sync.WaitGroup) {
				defer wg.Done()
				deleteTime, err := strconv.Atoi(v)
				if err != nil {
					return
				}
				if int64(deleteTime) > time.Now().Unix() {
					return
				}
				messageID, err := strconv.Atoi(k)
				if err != nil {
					return
				}
				req, err := c.bot.Request(tgbotapi.DeleteMessageConfig{
					ChatID:    c.update.Message.Chat.ID,
					MessageID: messageID,
				})
				if !req.Ok {
					logrus.Warnln(req.ErrorCode, err)
				}
				if err := db.RDB.HDel(context.Background(), deleteMessageKey, util.NumToStr(messageID)).Err(); err != nil {
					logrus.Error(err)
				}
			}(k, v, wg)
		}
		wg.Wait()
	}
}

func (c *BotConfig) addDeleteMessageQueue(delay int, messageID int) {
	deleteMessageKey := util.StrBuilder(deleteMessageKeyDir, util.NumToStr(c.update.Message.Chat.ID))
	if err := db.RDB.HMSet(context.Background(), deleteMessageKey, messageID, time.Now().Unix()+int64(delay)).Err(); err != nil {
		logrus.Error(err)
	}
}

func (c *BotConfig) getChatMember(userID int64) (tgbotapi.ChatMember, error) {
	req, err := c.bot.Request(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: c.update.Message.Chat.ID,
			UserID: userID,
		},
	})
	var chatMember tgbotapi.ChatMember
	if req.Ok {
		_ = json.Unmarshal(req.Result, &chatMember)
	} else {
		return chatMember, err
	}
	return chatMember, nil
}

func (c *BotConfig) IsEnableChatService(service string) bool {
	commandSwitchKey := util.StrBuilder(serviceSwitchKeyDir, util.NumToStr(c.update.Message.Chat.ID), ":disable_")
	res, err := db.RDB.Exists(c.ctx, util.StrBuilder(commandSwitchKey, service)).Result()
	if err != nil {
		logrus.Error(err)
		return false
	}
	if res > 0 {
		return false
	}
	return true
}
