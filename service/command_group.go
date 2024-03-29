package service

import (
	"athenabot/config"
	"athenabot/db"
	"athenabot/util"
	"crypto/rand"
	"github.com/go-redis/redis/v8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"math/big"
	"strconv"
	"strings"
	"time"
)

func (c *CommandConfig) marsTopCommand() {
	t := newTopConfig(c.BotConfig)
	t.getTop(marsTopKeyDir, "火星过", "次")
}

func (c *CommandConfig) studyTopCommand() {
	t := newTopConfig(c.BotConfig)
	t.getTop(studyTopKeyDir, "总课时", "分钟")
}

func (c *CommandConfig) studyCommand() {
	c.canHandleSelf = true
	if !c.isApproveCommandRule() {
		return
	}

	if c.isLimitCommand(3) {
		c.messageConfig.Text = "你学的太多了休息一下"
		c.sendCommandMessage()
		return
	}

	studyTopKey := util.StrBuilder(studyTopKeyDir, util.NumToStr(c.chatID))
	users, err := db.RDB.ZRange(c.ctx, studyTopKey, -3, -1).Result()
	if err != nil {
		logrus.Error(err)
	}
	for _, userID := range users {
		if userID == util.NumToStr(c.update.Message.From.ID) {
			if c.isLimitCommand(1) {
				c.messageConfig.Text = "你成绩太优秀了休息一下"
				c.sendCommandMessage()
				return
			}
		}
	}

	nowTimestamp := time.Now().Unix()
	randTime, _ := rand.Int(rand.Reader, big.NewInt(60))
	var rtTime int64
	arg, err := strconv.Atoi(c.commandArg)
	if err == nil {
		rtTime = randTime.Int64() + int64(uint8(arg))
	} else {
		rtTime = randTime.Int64() + 1
	}
	req, err := c.bot.Request(tgbotapi.RestrictChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: c.chatID,
			UserID: c.handleUserID,
		},
		UntilDate: rtTime*60 + nowTimestamp,
	})
	if req.Ok {
		logrus.Infof("handle_user:%v rt_time:%v", c.handleUserID, rtTime)
		c.messageConfig.Entities = []tgbotapi.MessageEntity{{
			Type:   "text_mention",
			Offset: 4,
			Length: util.TGNameWidth(c.handleUserName),
			User:   &tgbotapi.User{ID: c.handleUserID},
		}}
		c.messageConfig.Text = util.StrBuilder("好学生 ", c.handleUserName, " 恭喜获得学习时间", util.NumToStr(rtTime), "分钟")
		c.sendCommandMessage()
		t := newTopConfig(c.BotConfig)
		t.setTop(studyTopKeyDir, c.handleUserID, float64(rtTime))
		c.commandLimitAdd(1)
	} else {
		logrus.Errorln(req.ErrorCode, err)
	}
}

func (c *CommandConfig) banCommand() {
	c.mustAdmin = true
	c.mustReply = true
	c.mustAdminCanRestrictMembers = true
	if !c.isApproveCommandRule() {
		return
	}

	req, err := c.bot.Request(tgbotapi.BanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: c.chatID,
			UserID: c.handleUserID,
		},
	})
	if req.Ok {
		logrus.Infof("handle_user:%v", c.handleUserID)
		c.messageConfig.Entities = []tgbotapi.MessageEntity{{
			Type:   "text_mention",
			Offset: 0,
			Length: util.TGNameWidth(c.handleUserName),
			User:   &tgbotapi.User{ID: c.handleUserID},
		}}
		c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 已经被干掉了")
		c.sendCommandMessage()
	} else {
		logrus.Errorln(req.ErrorCode, err)
	}
}

func (c *CommandConfig) dbanCommand() {
	c.mustAdmin = true
	c.mustReply = true
	c.mustAdminCanRestrictMembers = true
	if !c.isApproveCommandRule() {
		return
	}
	req, err := c.bot.Request(tgbotapi.BanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: c.chatID,
			UserID: c.handleUserID,
		},
		RevokeMessages: true,
	})
	if c.IsEnableChatService("clear_my_48h_message") {
		chat48hMessageKey := util.StrBuilder(chat48hMessageDir, util.NumToStr(c.chatID), ":", util.NumToStr(c.handleUserID))
		messageIDs, err := db.RDB.HGetAll(c.ctx, chat48hMessageKey).Result()
		if err != nil {
			logrus.Error(err)
			return
		}
		for k := range messageIDs {
			messageID, err := strconv.Atoi(k)
			if err != nil {
				continue
			}
			c.addDeleteMessageQueue(0, messageID)
		}
	}
	if req.Ok {
		logrus.Infof("handle_user:%v", c.handleUserID)
		c.messageConfig.Entities = []tgbotapi.MessageEntity{{
			Type:   "text_mention",
			Offset: 0,
			Length: util.TGNameWidth(c.handleUserName),
			User:   &tgbotapi.User{ID: c.handleUserID},
		}}
		c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 已经消失的无影无踪")
		c.sendCommandMessage()
		req, err := c.bot.Request(tgbotapi.DeleteMessageConfig{
			ChatID:    c.chatID,
			MessageID: c.update.Message.ReplyToMessage.MessageID,
		})
		if !req.Ok {
			logrus.Errorln(req.ErrorCode, err)
		}
	} else {
		logrus.Errorln(req.ErrorCode, err)
	}
}

func (c *CommandConfig) unBanCommand() {
	c.mustAdmin = true
	c.mustReply = true
	c.mustAdminCanRestrictMembers = true
	if !c.isApproveCommandRule() {
		return
	}

	req, err := c.bot.Request(tgbotapi.UnbanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: c.chatID,
			UserID: c.handleUserID,
		},
		OnlyIfBanned: true,
	})
	if req.Ok {
		logrus.Infof("handle_user:%v", c.handleUserID)
		c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 获得救赎")
		c.sendCommandMessage()
	} else {
		logrus.Errorln(req.ErrorCode, err)
	}
}

func (c *CommandConfig) rtCommand() {
	c.mustAdmin = true
	c.mustReply = true
	c.mustAdminCanRestrictMembers = true
	if !c.isApproveCommandRule() {
		return
	}

	nowTimestamp := time.Now().Unix()
	var rtTime int64
	arg, err := strconv.Atoi(c.commandArg)
	if err == nil {
		rtTime = int64(arg)
	} else {
		rtTime = 500000
	}
	req, err := c.bot.Request(tgbotapi.RestrictChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: c.chatID,
			UserID: c.handleUserID,
		},
		UntilDate: rtTime*60 + nowTimestamp,
	})
	if req.Ok {
		logrus.Infof("handle_user:%v rt_time:%v", c.handleUserID, rtTime)
		c.messageConfig.Entities = []tgbotapi.MessageEntity{{
			Type:   "text_mention",
			Offset: 0,
			Length: util.TGNameWidth(c.handleUserName),
			User:   &tgbotapi.User{ID: c.handleUserID},
		}}
		c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 你需要休息", util.NumToStr(rtTime), "分钟")
		c.sendCommandMessage()
	} else {
		logrus.Errorln(req.ErrorCode, err)
	}
}

func (c *CommandConfig) unRtCommand() {
	c.mustAdmin = true
	c.mustReply = true
	c.mustAdminCanRestrictMembers = true
	if !c.isApproveCommandRule() {
		return
	}
	req, err := c.bot.Request(tgbotapi.RestrictChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: c.chatID,
			UserID: c.handleUserID,
		},
		Permissions: &tgbotapi.ChatPermissions{
			CanSendMessages:       true,
			CanSendMediaMessages:  true,
			CanSendPolls:          true,
			CanSendOtherMessages:  true,
			CanAddWebPagePreviews: true,
			CanChangeInfo:         false,
			CanInviteUsers:        true,
			CanPinMessages:        false,
		},
	})
	if req.Ok {
		logrus.Infof("handle_user:%v", c.handleUserID)
		c.messageConfig.Entities = []tgbotapi.MessageEntity{{
			Type:   "text_mention",
			Offset: 0,
			Length: util.TGNameWidth(c.handleUserName),
			User:   &tgbotapi.User{ID: c.handleUserID},
		}}
		c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 可以说话了")
		c.sendCommandMessage()
	} else {
		logrus.Errorln(req.ErrorCode, err)
	}
}

func (c *CommandConfig) warnCommand() {
	c.mustAdmin = true
	c.mustReply = true
	c.mustAdminCanRestrictMembers = true
	if !c.isApproveCommandRule() {
		return
	}
	warnKey := util.StrBuilder(warnKeyDir, util.NumToStr(c.chatID), ":", util.NumToStr(c.handleUserID))
	res, err := db.RDB.Exists(c.ctx, warnKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	var count int
	if res > 0 {
		resCount, err := db.RDB.Get(c.ctx, warnKey).Result()
		if err != nil {
			logrus.Error(err)
		}
		count, _ = strconv.Atoi(resCount)
		if count >= 3 {
			err := db.RDB.Del(c.ctx, warnKey).Err()
			if err != nil {
				logrus.Error(err)
			}
			c.banCommand()
			return
		} else {
			count += 1
			err := db.RDB.Set(c.ctx, warnKey, count, time.Second*time.Duration(config.Conf.KeyTTL)).Err()
			if err != nil {
				logrus.Error(err)
			}
		}
	} else {
		count = 1
		err := db.RDB.Set(c.ctx, warnKey, count, time.Second*time.Duration(config.Conf.KeyTTL)).Err()
		if err != nil {
			logrus.Error(err)
		}
	}
	logrus.Infof("handle_user:%v warn_count:%v", c.handleUserID, count)
	c.messageConfig.Entities = []tgbotapi.MessageEntity{{
		Type:   "text_mention",
		Offset: 0,
		Length: util.TGNameWidth(c.handleUserName),
		User:   &tgbotapi.User{ID: c.handleUserID},
	}}
	c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 移除警告 ", strconv.Itoa(count), "/3")
	c.sendCommandMessage()
}

func (c *CommandConfig) unWarnCommand() {
	c.mustAdmin = true
	c.mustReply = true
	c.mustAdminCanRestrictMembers = true
	if !c.isApproveCommandRule() {
		return
	}
	warnKey := util.StrBuilder(warnKeyDir, util.NumToStr(c.chatID), ":", util.NumToStr(c.handleUserID))
	res, err := db.RDB.Exists(c.ctx, warnKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	var count int
	if res > 0 {
		resCount, _ := db.RDB.Get(c.ctx, warnKey).Result()
		count, err = strconv.Atoi(resCount)
		if err != nil {
			logrus.Error(err)
			return
		}
		if count == 1 {
			count = 0
			err := db.RDB.Del(c.ctx, warnKey).Err()
			if err != nil {
				logrus.Error(err)
			}
		} else {
			count -= 1
			err := db.RDB.Set(c.ctx, warnKey, count, time.Second*time.Duration(config.Conf.KeyTTL)).Err()
			if err != nil {
				logrus.Error(err)
			}
		}
	}
	logrus.Infof("handle_user:%v warn_count:%v", c.handleUserID, count)
	c.messageConfig.Entities = []tgbotapi.MessageEntity{{
		Type:   "text_mention",
		Offset: 0,
		Length: util.TGNameWidth(c.handleUserName),
		User:   &tgbotapi.User{ID: c.handleUserID},
	}}
	c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 警告已经移除 ", strconv.Itoa(count), "/3")
	c.sendCommandMessage()
}

func (c *CommandConfig) enableCommand() {
	c.mustAdmin = true
	c.canHandleAdmin = true
	if !c.isApproveCommandRule() {
		return
	}
	commandSwitchKey := util.StrBuilder(serviceSwitchKeyDir, util.NumToStr(c.chatID), ":disable_")
	if _, ok := commandsGroupFunc[c.commandArg]; !ok {
		return
	}
	err := db.RDB.Del(c.ctx, commandSwitchKey+c.commandArg).Err()
	if err != nil {
		logrus.Error(err)
	}
	logrus.Infof("enable_command:%v", c.commandArg)
	c.messageConfig.Text = util.StrBuilder(c.commandArg, "\n命令已启用")
	c.sendCommandMessage()
}

func (c *CommandConfig) disableCommand() {
	c.mustAdmin = true
	c.canHandleAdmin = true
	if !c.isApproveCommandRule() {
		return
	}
	commandSwitchKey := util.StrBuilder(serviceSwitchKeyDir, util.NumToStr(c.chatID), ":disable_")
	if _, ok := commandsGroupFunc[c.commandArg]; !ok {
		return
	}
	err := db.RDB.Set(c.ctx, commandSwitchKey+c.commandArg, 0, 0).Err()
	if err != nil {
		logrus.Error(err)
	}
	logrus.Infof("disable_command:%v", c.commandArg)
	c.messageConfig.Text = util.StrBuilder(c.commandArg, " 命令已禁用")
	c.sendCommandMessage()
}

func (c *CommandConfig) doudouCommand() {
	c.mustReply = true
	c.canHandleSelf = true
	c.canHandleNoAdminReply = true
	c.canHandleAdminReply = true
	if !c.isApproveCommandRule() {
		return
	}

	var isHandleUserToOwner bool
	var isCommandUserFromOwner bool
	if c.handleUserID == config.Conf.OwnerID {
		isHandleUserToOwner = true
	}

	if c.update.Message.From.ID == config.Conf.OwnerID {
		isCommandUserFromOwner = true
	}

	rtTimestamp := time.Now().Unix()
	chatMember, err := c.getChatMember(c.handleUserID)
	if err != nil {
		logrus.Error(err)
		return
	}
	if !chatMember.CanSendMessages && chatMember.Status == "restricted" {
		if chatMember.UntilDate > rtTimestamp {
			rtTimestamp += chatMember.UntilDate - rtTimestamp
		} else {
			c.messageConfig.Entities = []tgbotapi.MessageEntity{{
				Type:   "text_mention",
				Offset: 0,
				Length: util.TGNameWidth(c.handleUserName),
				User:   &tgbotapi.User{ID: c.handleUserID},
			}}
			c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 已经被打倒不用在斗了")
			c.sendCommandMessage()
			return
		}
	}
	limit := 3
	if c.userIsAdmin {
		limit = 5
	}
	heiWuLeiKey := util.StrBuilder(heiWuLeiDir, util.NumToStr(c.chatID))
	isHandleUserToHwl, err := db.RDB.SIsMember(c.ctx, heiWuLeiKey, c.handleUserID).Result()
	if err != nil {
		logrus.Error(err)
	}
	isCommandUserFromHwl, err := db.RDB.SIsMember(c.ctx, heiWuLeiKey, c.update.Message.From.ID).Result()
	if err != nil {
		logrus.Error(err)
	}
	if isHandleUserToHwl {
		limit = 10
	}

	c.messageConfig.Entities = []tgbotapi.MessageEntity{{
		Type:   "text_mention",
		Offset: 0,
		Length: util.TGNameWidth(c.update.Message.From.FirstName),
		User:   &tgbotapi.User{ID: c.update.Message.From.ID},
	}}

	if c.isLimitCommand(limit) && !isCommandUserFromOwner {
		if c.userIsAdmin {
			c.messageConfig.Text = util.StrBuilder(c.update.Message.From.FirstName, " 斗斗扩大化 检讨3分钟（实际不执行）")
			c.sendCommandMessage()
		} else {
			rtTimestamp += 180
			req, err := c.bot.Request(tgbotapi.RestrictChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: c.chatID,
					UserID: c.update.Message.From.ID,
				},
				UntilDate: rtTimestamp,
			})
			if req.Ok {
				logrus.Infof("handle_user:%v rt_time:%v", c.update.Message.From.ID, 3)
				c.messageConfig.Text = util.StrBuilder(c.update.Message.From.FirstName, " 斗斗扩大化 检讨3分钟")
				c.sendCommandMessage()
			} else {
				logrus.Errorln(req.ErrorCode, err)
			}
		}
		return
	}

	doudouOdds, _ := rand.Int(rand.Reader, big.NewInt(99))
	if isCommandUserFromHwl {
		doudouOdds, _ = rand.Int(rand.Reader, big.NewInt(10))
	}
	if doudouOdds.Int64() < 5 && !c.userIsAdmin && !isCommandUserFromOwner && !isHandleUserToHwl {
		rtTimestamp += 180
		req, err := c.bot.Request(tgbotapi.RestrictChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{
				ChatID: c.chatID,
				UserID: c.update.Message.From.ID,
			},
			UntilDate: rtTimestamp,
		})
		if req.Ok {
			logrus.Infof("handle_user:%v rt_time:%v", c.update.Message.From.ID, 3)
			c.messageConfig.Text = util.StrBuilder(c.update.Message.From.FirstName, " 诬陷群友 检讨3分钟")
			c.sendCommandMessage()
		} else {
			logrus.Errorln(req.ErrorCode, err)
		}
		return
	}

	c.messageConfig.Entities = []tgbotapi.MessageEntity{{
		Type:   "text_mention",
		Offset: 0,
		Length: util.TGNameWidth(c.handleUserName),
		User:   &tgbotapi.User{ID: c.handleUserID},
	}}

	if c.replyUserIsAdmin {
		c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 属于群内部矛盾 诫勉谈话一次")
		c.sendCommandMessage()
	} else {
		randTime, _ := rand.Int(rand.Reader, big.NewInt(3))
		rtMin := randTime.Int64() + 1
		if isHandleUserToHwl {
			rtMin = 3
		}
		rtTimestamp += rtMin * 60
		logrus.Infof("handle_user:%v rt_time:%v", c.handleUserID, rtMin)
		if isHandleUserToOwner {
			c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 对群不忠诚 检讨", util.NumToStr(rtMin), "分钟")
			c.sendCommandMessage()
		} else {
			req, err := c.bot.Request(tgbotapi.RestrictChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: c.chatID,
					UserID: c.handleUserID,
				},
				UntilDate: rtTimestamp,
			})
			if req.Ok {
				c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 对群不忠诚 检讨", util.NumToStr(rtMin), "分钟")
				c.sendCommandMessage()
			} else {
				logrus.Errorln(req.ErrorCode, err)
			}
		}
	}
	c.commandLimitAdd(1)
	t := newTopConfig(c.BotConfig)
	t.setTop(doudouTopKeyDir, c.handleUserID, float64(1))
}

func (c *CommandConfig) doudouTopCommand() {
	t := newTopConfig(c.BotConfig)
	t.getTop(doudouTopKeyDir, "被斗过", "次")
}

func (c *CommandConfig) clearMy48hMessageCommand() {
	// c.commandMessageCleanCountdown = 5
	c.canHandleSelf = true
	c.canHandleAdmin = true
	if !c.isApproveCommandRule() {
		return
	}
	if c.handleUserID != c.update.Message.From.ID {
		logrus.Warn("ignore: handle_user_id != from_id")
		return
	}
	c.messageConfig.Entities = []tgbotapi.MessageEntity{{
		Type:   "text_mention",
		Offset: 0,
		Length: util.TGNameWidth(c.handleUserName),
		User:   &tgbotapi.User{ID: c.handleUserID},
	}}

	if c.isLimitCommand(2) {
		c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 本来无一物 何处惹尘埃")
		c.sendCommandMessage()
		return
	}

	logrus.Infof("handle_user:%v", c.handleUserID)
	chat48hMessageDeleteCrontabKey := util.StrBuilder(chat48hMessageDeleteCrontabDir, util.NumToStr(c.chatID))
	switch c.commandArg {
	case "enable_cron":
		err := db.RDB.SAdd(c.ctx, chat48hMessageDeleteCrontabKey, c.handleUserID).Err()
		if err != nil {
			logrus.Error(err)
		}
		c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 图图计划已开启，你在本群的消息将会自动图图")
		c.sendCommandMessage()
		return
	case "disable_cron":
		err := db.RDB.SRem(c.ctx, chat48hMessageDeleteCrontabKey, c.handleUserID).Err()
		if err != nil {
			logrus.Warn(err)
		}
		c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 图图计划已禁用")
		c.sendCommandMessage()
		return
	default:
		c.commandLimitAdd(1)
		c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 图图中，如想完整图图请咨询管理员\nFBI WARNING 请勿随意模仿")
		callbackData := generateCallbackData("clear-msg", c.handleUserID, c.update.Message.MessageID)
		confirmButton := tgbotapi.NewInlineKeyboardButtonData("确定图图？", callbackData)
		replyMarkup := tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{confirmButton},
		)
		c.messageConfig.ReplyMarkup = replyMarkup
		c.sendCommandMessage()
	}
}

func (c *CommandConfig) honorTopCommand() {
	c.botMessageCleanCountdown = 0
	t := newTopConfig(c.BotConfig)
	t.getTop(doudouTopKeyDir, "荣誉值", "升")
}

func (c *CommandConfig) chatBlacklistCommand() {
	key := util.StrBuilder(chatBlacklistDir, util.NumToStr(c.chatID))
	if len(c.commandArg) == 0 {
		keywords, err := db.RDB.ZRevRange(c.ctx, key, 0, -1).Result()
		if err != nil {
			logrus.Error(err)
		}
		var text string
		for _, keyword := range keywords {
			text += util.StrBuilder(keyword, "\n")
		}
		c.messageConfig.Text = text
		c.sendCommandMessage()
	}
	if len(c.commandArg) > 2 {
		c.mustAdmin = true
		c.canHandleAdmin = true
		if !c.isApproveCommandRule() {
			return
		}
		if c.commandArg[:2] == "-a" {
			card, _ := db.RDB.ZCard(c.ctx, key).Result()
			if card >= 100 {
				c.messageConfig.Text = "黑名单不能超过100个"
				c.sendCommandMessage()
				return
			}
			err := db.RDB.ZAdd(c.ctx, key, &redis.Z{
				Score:  float64(time.Now().Unix()),
				Member: c.commandArg[2:],
			}).Err()
			if err != nil {
				logrus.Error(err)
			}
		}
		if c.commandArg[:2] == "-d" {
			err := db.RDB.ZRem(c.ctx, key, c.commandArg[2:]).Err()
			if err != nil {
				logrus.Error(err)
			}
		}
	}
}

func (c *CommandConfig) chatUserActivityCommand() {
	c.mustAdmin = true
	c.canHandleAdmin = true
	if !c.isApproveCommandRule() {
		return
	}

	if len(c.commandArg) > 0 {
		if c.isLimitCommand(5) {
			return
		}

		inactiveDays, err := strconv.Atoi(c.commandArg)
		if err != nil {
			return
		}
		callbackData := generateCallbackData("clear-users", c.handleUserID, inactiveDays)
		confirmButton := tgbotapi.NewInlineKeyboardButtonData("确定图图？", callbackData)
		replyMarkup := tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{confirmButton},
		)
		c.messageConfig.ReplyMarkup = replyMarkup
		c.messageConfig.Text = util.StrBuilder(util.NumToStr(inactiveDays), " 天不活跃群友将会被图图\n")
		c.sendCommandMessage()
		c.commandLimitAdd(1)
		return
	}

	chatUserActivityData, err := c.generateUserActivityData()
	if err != nil {
		logrus.Error(err)
		return
	}

	var msg string
	for _, data := range chatUserActivityData {
		if data.inactiveDays == 0 {
			msg += util.StrBuilder(data.fullName, " ", "活跃中\n")
			continue
		}
		msg += util.StrBuilder(data.fullName, " ", util.NumToStr(data.inactiveDays), "天 未活跃\n")
	}

	c.messageConfig.Text = msg
	c.sendCommandMessage()
}

func (c *CommandConfig) botShareholdersCommand() {
	shareholdersKey := util.StrBuilder(shareholdersDir, util.NumToStr(c.bot.Self.ID))
	shareholdersData, err := db.RDB.HGetAll(c.ctx, shareholdersKey).Result()
	if err != nil {
		logrus.Error(err)
		return
	}
	c.messageConfig.Text = "本BOT股东成员（排名不分先后）\n"
	for id, title := range shareholdersData {
		userID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			continue
		}
		c.messageConfig.Text += util.StrBuilder(c.getUserCache(userID).User.FirstName, " ", title, "\n")
	}
	c.sendCommandMessage()
}

func (c *CommandConfig) botBeShareholderCommand() {
	shareholdersKey := util.StrBuilder(shareholdersDir, util.NumToStr(c.bot.Self.ID))
	c.mustReply = true
	c.canHandleSelf = true
	c.canHandleNoAdminReply = true
	c.canHandleAdminReply = true
	if !c.isApproveCommandRule() {
		return
	}
	if c.update.Message.From.ID != config.Conf.OwnerID {
		c.messageConfig.Text = "你不是董事长，无权邀请董事会成员"
		c.sendCommandMessage()
		return
	}
	err := db.RDB.HSet(c.ctx, shareholdersKey, c.handleUserID, c.commandArg).Err()
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Infof("handle_user:%v command_arg:%v", c.handleUserID, c.commandArg)
	c.messageConfig.Entities = []tgbotapi.MessageEntity{{
		Type:   "text_mention",
		Offset: 0,
		Length: util.TGNameWidth(c.handleUserName),
		User:   &tgbotapi.User{ID: c.handleUserID},
	}}
	c.botMessageCleanCountdown = 0
	c.commandMessageCleanCountdown = 0
	c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 加入董事会，职位为", c.commandArg)
	c.sendCommandMessage()
}

func (c *CommandConfig) heiWuLeiCommand() {
	c.mustReply = true
	c.mustAdmin = true
	c.mustAdminCanRestrictMembers = true
	if !c.isApproveCommandRule() {
		return
	}
	heiWuLeiKey := util.StrBuilder(heiWuLeiDir, util.NumToStr(c.chatID))
	c.messageConfig.Entities = []tgbotapi.MessageEntity{{
		Type:   "text_mention",
		Offset: 0,
		Length: util.TGNameWidth(c.handleUserName),
		User:   &tgbotapi.User{ID: c.handleUserID},
	}}
	err := db.RDB.SAdd(c.ctx, heiWuLeiKey, c.handleUserID).Err()
	if err != nil {
		logrus.Error(err)
	}
	c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 踏上千万只脚")
	c.sendCommandMessage()
}

func (c *CommandConfig) pingFanCommand() {
	c.mustReply = true
	c.mustAdmin = true
	c.mustAdminCanRestrictMembers = true
	if !c.isApproveCommandRule() {
		return
	}
	heiWuLeiKey := util.StrBuilder(heiWuLeiDir, util.NumToStr(c.chatID))
	c.messageConfig.Entities = []tgbotapi.MessageEntity{{
		Type:   "text_mention",
		Offset: 0,
		Length: util.TGNameWidth(c.handleUserName),
		User:   &tgbotapi.User{ID: c.handleUserID},
	}}
	isHwl, err := db.RDB.SIsMember(c.ctx, heiWuLeiKey, c.handleUserID).Result()
	if err != nil {
		logrus.Error(err)
	}
	if !isHwl {
		return
	}
	err = db.RDB.SRem(c.ctx, heiWuLeiKey, c.handleUserID).Err()
	if err != nil {
		logrus.Error(err)
	}
	c.messageConfig.Text = util.StrBuilder(c.handleUserName, " 已被平反，你受委屈了，有困难跟群里提")
	c.sendCommandMessage()
}

func (c *CommandConfig) powerCommand() {
	arg := c.commandArg
	if c.update.Message.From.ID != config.Conf.OwnerID {
		return
	}
	c.userIsAdmin = true
	c.userIsRestrictAdmin = true
	if len(arg) > 0 {
		args := strings.Split(arg, ",")
		if len(args) == 2 {
			c.commandArg = args[1]
		} else {
			c.commandArg = ""
		}
		if f, ok := commandsGroupFunc[args[0]]; ok {
			f(c)
		}
	}
}
