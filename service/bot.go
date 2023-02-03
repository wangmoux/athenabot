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
	administratorsCacheKey := util.StrBuilder(administratorsCacheDir, util.NumToStr(c.update.Message.Chat.ID))
	keyExists, err := db.RDB.Exists(c.ctx, administratorsCacheKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	if keyExists > 0 {
		isAdministrator, err := db.RDB.SIsMember(c.ctx, administratorsCacheKey, userID).Result()
		if err != nil {
			logrus.Error(err)
		}
		if isAdministrator {
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
		chatAdministratorsMap := make(map[int64]uint8)
		for i := range chatAdministrators {
			id := resJson.GetIndex(i).Get("user").Get("id").MustInt64()
			chatAdministratorsMap[id] = 0
		}
		for _, id := range config.Conf.SudoAdmins {
			chatAdministratorsMap[id] = 0
		}
		for id := range chatAdministratorsMap {
			err := db.RDB.SAdd(c.ctx, administratorsCacheKey, id).Err()
			if err != nil {
				logrus.Error(err)
			}
		}
		err = db.RDB.Expire(c.ctx, administratorsCacheKey, time.Second*86400).Err()
		if err != nil {
			logrus.Error(err)
		}
		if _, _ok := chatAdministratorsMap[userID]; _ok {
			return true
		}
	}
	return false
}

func (c *BotConfig) getUserNameCache(wg *sync.WaitGroup, userID int64, cache *userNameCache) {
	defer wg.Done()
	userNameCacheKey := util.StrBuilder(userNameCacheDir, util.NumToStr(userID))
	keyExists, err := db.RDB.Exists(c.ctx, userNameCacheKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	var userName string
	if keyExists > 0 {
		userName, err = db.RDB.Get(c.ctx, userNameCacheKey).Result()
		if err != nil {
			logrus.Error(err)
		}
	} else {
		req, err := c.bot.Request(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: c.update.Message.Chat.ID,
				UserID: userID,
			},
		})
		if req.Ok {
			userJson := &simplejson.Json{}
			userJson, _ = simplejson.NewJson(req.Result)
			userName = userJson.Get("user").Get("first_name").MustString()
			if len(userName) > 0 {
				err = db.RDB.Set(c.ctx, userNameCacheKey, userName, time.Second*86400).Err()
				if err != nil {
					logrus.Error(err)
				}
			}
		} else {
			logrus.Errorln(req.ErrorCode, err)
		}
	}
	cache.userName[userID] = userName
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
