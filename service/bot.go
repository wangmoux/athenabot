package service

import (
	"athenabot/config"
	"athenabot/db"
	"athenabot/util"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type BotConfig struct {
	update                   tgbotapi.Update
	bot                      *tgbotapi.BotAPI
	messageConfig            tgbotapi.MessageConfig
	ctx                      context.Context
	cancel                   context.CancelFunc
	botMessageCleanCountdown int
	botMessageID             int
	chatID                   int64
	sendMessageLimit         chan struct{}
}

func NewBotConfig(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) *BotConfig {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	botConfig := &BotConfig{
		ctx:              ctx,
		cancel:           cancel,
		update:           update,
		bot:              bot,
		sendMessageLimit: make(chan struct{}, 1),
	}
	return botConfig
}

func (c *BotConfig) SetChatID(chatID int64) {
	c.chatID = chatID
}

func (c *BotConfig) sendReplyMessage() {
	c.setReplyMessage(c.update.Message.MessageID)
	c.sendMessage()
}

func (c *BotConfig) setReplyMessage(replyMessageID int) {
	c.messageConfig.ReplyToMessageID = replyMessageID
}

func (c *BotConfig) sendMessage() {
	c.sendMessageLimit <- struct{}{}
	defer func() {
		c.messageConfig = tgbotapi.MessageConfig{}
		<-c.sendMessageLimit
	}()
	msg := c.messageConfig
	msg.ChatID = c.chatID
	if msg.Text == "" {
		logrus.Warnln("send empty message")
		msg.Text = "无言以对"
	}
	req, err := c.bot.Send(msg)
	if err != nil {
		logrus.Error(err)
	} else {
		c.botMessageID = req.MessageID
	}
	logrus.Debugf("send_msg:%v", util.LogMarshal(msg))
}

func (c *BotConfig) sendRequestMessage(ct tgbotapi.Chattable) {
	req, err := c.bot.Request(ct)
	if !req.Ok {
		logrus.Errorln(req.ErrorCode, err)
	}
	logrus.Debugf("send_msg:%v", util.LogMarshal(ct))
}

func (c *BotConfig) isAdmin(userID int64) bool {
	if c.isSudoAdmin(userID) {
		return true
	}
	if userID == 1087968824 {
		return true
	}
	var isAdmin bool
	administratorsCacheKey := util.StrBuilder(administratorsCacheDir, util.NumToStr(c.chatID))
	keyExists, err := db.RDB.Exists(c.ctx, administratorsCacheKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	if keyExists > 0 {
		isAdmin, err = db.RDB.HExists(c.ctx, administratorsCacheKey, util.NumToStr(userID)).Result()
		if err != nil {
			logrus.Error(err)
		}

	} else {
		req, err := c.bot.Request(tgbotapi.ChatAdministratorsConfig{
			ChatConfig: tgbotapi.ChatConfig{
				ChatID: c.chatID,
			},
		})

		if !req.Ok {
			logrus.Errorln(req.ErrorCode, err)
		}
		var chatMembers []tgbotapi.ChatMember
		err = json.Unmarshal(req.Result, &chatMembers)
		if err != nil {
			logrus.Error(err)
		}
		for _, members := range chatMembers {
			if members.User.ID == userID {
				isAdmin = true
			}
			membersStr, _ := json.Marshal(members)
			err := db.RDB.HSet(c.ctx, administratorsCacheKey, members.User.ID, membersStr).Err()
			if err != nil {
				logrus.Error(err)
			}
		}
		err = db.RDB.Expire(c.ctx, administratorsCacheKey, time.Second*3600).Err()
		if err != nil {
			logrus.Error(err)
		}
	}
	return isAdmin
}

func (c *BotConfig) isAdminCanRestrictMembers(userID int64) bool {
	if c.isSudoAdmin(userID) {
		return true
	}
	if userID == 1087968824 {
		return true
	}
	if !c.isAdmin(userID) {
		return false
	}
	administratorsCacheKey := util.StrBuilder(administratorsCacheDir, util.NumToStr(c.chatID))
	req, err := db.RDB.HGet(c.ctx, administratorsCacheKey, util.NumToStr(userID)).Result()
	if err != nil {
		logrus.Error(err)
		return false
	}
	var chatMembers *tgbotapi.ChatMember
	err = json.Unmarshal([]byte(req), &chatMembers)
	if err != nil {
		logrus.Error(err)
		return false
	}
	return chatMembers.CanRestrictMembers
}

func (c *BotConfig) getUserCache(userID int64) *tgbotapi.ChatMember {
	chatMember := &tgbotapi.ChatMember{
		User: &tgbotapi.User{},
	}
	userCacheKey := util.StrBuilder(usersCacheDir, util.NumToStr(c.chatID), ":", util.NumToStr(userID))
	keyExists, err := db.RDB.Exists(c.ctx, userCacheKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	if keyExists > 0 {
		userReq, err := db.RDB.Get(c.ctx, userCacheKey).Result()
		if err != nil {
			logrus.Error(err)
		}
		err = json.Unmarshal([]byte(userReq), &chatMember)
		if err != nil {
			logrus.Error(err)
		}
		return chatMember
	} else {
		req, err := c.bot.Request(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: c.chatID,
				UserID: userID,
			},
		})
		if req.Ok {
			err = json.Unmarshal(req.Result, &chatMember)
			if err != nil {
				logrus.Error(err)
			}
			if len(chatMember.User.UserName) == 0 {
				chatMember.User.UserName = "unknown"
			}
			if len(chatMember.User.FirstName)+len(chatMember.User.LastName) == 0 {
				chatMember.User.FirstName = "无名氏"
			}
			err = db.RDB.Set(c.ctx, userCacheKey, string(req.Result), time.Second*3600).Err()
			if err != nil {
				logrus.Error(err)
			}
		} else {
			logrus.Errorln(req.ErrorCode, err)
			if strings.HasSuffix(req.Description, userIsNotFoundMessage) {
				chatMember.Status = "deleted"
				chatMember.User.UserName = "unknown"
				chatMember.User.FirstName = "无名氏"
			}
		}
	}
	return chatMember
}

func (c *BotConfig) DeleteMessageCronHandler(ctx context.Context) {
	logrus.Infof("new delete_message_cron_handler:%v", c.chatID)
	deleteMessageKey := util.StrBuilder(deleteMessageKeyDir, util.NumToStr(c.chatID))
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
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
						ChatID:    c.chatID,
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
		case <-ctx.Done():
			logrus.Infof("delete_message_cron_handler exited:%v", c.chatID)
			return
		}
	}
}

func (c *BotConfig) addDeleteMessageQueue(delay int, messageID int) {
	deleteMessageKey := util.StrBuilder(deleteMessageKeyDir, util.NumToStr(c.chatID))
	if err := db.RDB.HMSet(context.Background(), deleteMessageKey, messageID, time.Now().Unix()+int64(delay)).Err(); err != nil {
		logrus.Error(err)
	}
}

func (c *BotConfig) getChatMember(userID int64) (tgbotapi.ChatMember, error) {
	req, err := c.bot.Request(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: c.chatID,
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
	if service == "enable" || service == "disable" {
		return true
	}
	commandSwitchKey := util.StrBuilder(serviceSwitchKeyDir, util.NumToStr(c.chatID), ":disable_")

	allRes, err := db.RDB.Exists(c.ctx, util.StrBuilder(commandSwitchKey, "all")).Result()
	if err != nil {
		logrus.Error(err)
		return false
	}

	serviceRes, err := db.RDB.Exists(c.ctx, util.StrBuilder(commandSwitchKey, service)).Result()
	if err != nil {
		logrus.Error(err)
		return false
	}

	if allRes+serviceRes > 0 {
		return false
	}
	return true
}

func (c *BotConfig) sudoAdmins() {
	sudoAdminOnce.Do(func() {
		sudoAdministratorsKey := util.StrBuilder(sudoAdministratorsDir, ":", util.NumToStr(c.bot.Self.ID))
		keyExists, err := db.RDB.Exists(c.ctx, sudoAdministratorsKey).Result()
		if err != nil {
			logrus.Error(err)
		}
		if keyExists > 0 {
			err := db.RDB.Del(c.ctx, sudoAdministratorsKey).Err()
			if err != nil {
				logrus.Error(err)
			}
		}
		for _, i := range config.Conf.SudoAdmins {
			err := db.RDB.SAdd(c.ctx, sudoAdministratorsKey, i).Err()
			if err != nil {
				logrus.Error(err)
			}
		}
	})
}

func (c *BotConfig) isSudoAdmin(userID int64) bool {
	c.sudoAdmins()
	sudoAdministratorsKey := util.StrBuilder(sudoAdministratorsDir, ":", util.NumToStr(c.bot.Self.ID))
	userExists, err := db.RDB.SIsMember(c.ctx, sudoAdministratorsKey, userID).Result()
	if err != nil {
		logrus.Error(err)
	}
	return userExists
}

func (c *BotConfig) IsGroupWhitelist(username string) bool {
	if !config.Conf.EnableWhitelist {
		return true
	}
	groupWhitelistKey := util.StrBuilder(groupWhitelistDir, util.NumToStr(c.bot.Self.ID))
	isMember, err := db.RDB.SIsMember(c.ctx, groupWhitelistKey, username).Result()
	if err != nil {
		logrus.Error(err)
	}
	return isMember
}

func (c *BotConfig) IsMarsWhitelist(username string) bool {
	if !config.Conf.EnableMarsWhitelist {
		return true
	}
	marsWhitelistKey := util.StrBuilder(marsWhitelistDir, util.NumToStr(c.bot.Self.ID))
	isMember, err := db.RDB.SIsMember(c.ctx, marsWhitelistKey, username).Result()
	if err != nil {
		logrus.Error(err)
	}
	return isMember
}

func (c *BotConfig) IsMarsOCRWhitelist(username string) bool {
	if !config.Conf.MarsOCR.EnableWhitelist {
		return true
	}
	marsOCRWhitelistKey := util.StrBuilder(marsOCRWhitelistDir, util.NumToStr(c.bot.Self.ID))
	isMember, err := db.RDB.SIsMember(c.ctx, marsOCRWhitelistKey, username).Result()
	if err != nil {
		logrus.Error(err)
	}
	return isMember
}

func (c *BotConfig) IsChatGuardWhitelist(username string) bool {
	if !config.Conf.ChatGuard.EnableWhitelist {
		return true
	}
	chatGuardWhitelistKey := util.StrBuilder(chatGuardWhitelistDir, util.NumToStr(c.bot.Self.ID))
	isMember, err := db.RDB.SIsMember(c.ctx, chatGuardWhitelistKey, username).Result()
	if err != nil {
		logrus.Error(err)
	}
	return isMember
}

func (c *BotConfig) IsChatbotWhitelist(username string) bool {
	if !config.Conf.ChatBot.EnableWhitelist {
		return true
	}
	chatbotWhitelistKey := util.StrBuilder(chatBotWhitelistDir, util.NumToStr(c.bot.Self.ID))
	isMember, err := db.RDB.SIsMember(c.ctx, chatbotWhitelistKey, username).Result()
	if err != nil {
		logrus.Error(err)
	}
	return isMember
}

func (c *BotConfig) IsCommand() bool {
	util.LogMarshal(c.update)
	if !c.update.Message.IsCommand() {
		return false
	}
	if c.update.Message.Command() != c.update.Message.CommandWithAt() {
		if c.update.Message.CommandWithAt() != util.StrBuilder(c.update.Message.Command(), "@", c.bot.Self.UserName) {
			return false
		}
	}
	if c.update.Message.ViaBot != nil {
		if c.update.Message.ViaBot.UserName != c.bot.Self.UserName {
			return false
		}
	}
	return true
}

func (c *BotConfig) GetUpdate() tgbotapi.Update {
	return c.update
}
