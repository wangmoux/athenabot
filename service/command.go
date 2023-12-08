package service

import (
	"athenabot/config"
	"athenabot/db"
	"athenabot/util"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type CommandConfig struct {
	*BotConfig
	command                      string
	commandArg                   string
	handleUserName               string
	handleUserID                 int64
	mustReply                    bool
	mustAdmin                    bool
	canHandleAdmin               bool
	userIsAdmin                  bool
	replyUserIsAdmin             bool
	canHandleSelf                bool
	commandMessageCleanCountdown int
	canHandleNoAdminReply        bool
	canHandleAdminReply          bool
	mustAdminCanRestrictMembers  bool
	userIsRestrictAdmin          bool
}

func NewCommandConfig(botConfig *BotConfig) (commandConfig *CommandConfig) {
	commandConfig = &CommandConfig{
		BotConfig:  botConfig,
		command:    botConfig.update.Message.Command(),
		commandArg: botConfig.update.Message.CommandArguments(),
	}
	return commandConfig
}

func (c *CommandConfig) InCommands() {
	if _, ok := config.DisableCommandsMap[c.command]; !ok {
		if _, _ok := commandsGroupFunc[c.command]; !_ok {
			logrus.Warnf("command not registered:%v", c.command)
			return
		}
		c.botMessageCleanCountdown = 300
		c.commandMessageCleanCountdown = 300
		defer func() {
			if c.commandMessageCleanCountdown > 0 {
				go c.addDeleteMessageQueue(c.commandMessageCleanCountdown, c.update.Message.MessageID)
			}
			if c.botMessageID > 0 && c.botMessageCleanCountdown > 0 {
				go c.addDeleteMessageQueue(c.botMessageCleanCountdown, c.botMessageID)
			}
		}()
		if c.IsEnableChatService(c.command) || c.update.Message.From.ID == config.Conf.OwnerID {
			logrus.Infof("command_user:%v command:%s command_arg:%s", c.update.Message.From.ID, c.command, c.commandArg)
			commandsGroupFunc[c.command](c)
		} else {
			logrus.Warnf("command disabled:%v", c.command)
		}
	}
}

func (c *CommandConfig) InPrivateCommands() {
	if _, ok := config.DisablePrivateCommandsMap[c.command]; !ok {
		if _, _ok := commandsPrivateFunc[c.command]; !_ok {
			logrus.Warnf("command not registered:%v", c.command)
			return
		}
		logrus.Infof("command_user:%v command:%s command_arg:%s", c.update.Message.From.ID, c.command, c.commandArg)
		commandsPrivateFunc[c.command](c)
	} else {
		logrus.Warnf("command disabled:%v", c.command)
	}
}

func (c *CommandConfig) commandLimitAdd(addCount int) {
	commandLimitKey := util.StrBuilder(commandLimitKeyDir, util.NumToStr(c.chatID), ":"+c.command, "_", util.NumToStr(c.update.Message.From.ID))
	res, err := db.RDB.Exists(c.ctx, commandLimitKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	var count int
	if res > 0 {
		resCount, err := db.RDB.Get(c.ctx, commandLimitKey).Result()
		if err != nil {
			logrus.Error(err)
		}
		count, err = strconv.Atoi(resCount)
		if err != nil {
			logrus.Error(err)
		}
		count += addCount
	} else {
		count = addCount
	}
	now := time.Now().UTC()
	expiration := ((24 - now.Hour()) * 3600) - (now.Minute() * 60) - now.Second()
	err = db.RDB.Set(c.ctx, commandLimitKey, count, time.Second*time.Duration(expiration)).Err()
	if err != nil {
		logrus.Error(err)
	}
}

func (c *CommandConfig) isLimitCommand(limit int) bool {
	if config.Conf.LogLevel > 3 {
		logrus.Warnln("ignore command limit log_level>3")
		return false
	}
	commandLimitKey := util.StrBuilder(commandLimitKeyDir, util.NumToStr(c.chatID), ":"+c.command, "_", util.NumToStr(c.update.Message.From.ID))
	res, err := db.RDB.Exists(c.ctx, commandLimitKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	if res > 0 {
		resCount, err := db.RDB.Get(c.ctx, commandLimitKey).Result()
		if err != nil {
			logrus.Error(err)
		}
		count, err := strconv.Atoi(resCount)
		if err != nil {
			logrus.Error(err)
		}
		if count >= limit {
			return true
		}
	}
	return false
}

func (c *CommandConfig) isApproveCommandRule() bool {
	if c.isAdmin(c.update.Message.From.ID) {
		c.userIsAdmin = true
	}
	if c.isAdminCanRestrictMembers(c.update.Message.From.ID) {
		c.userIsRestrictAdmin = true
	}
	if c.update.Message.ReplyToMessage != nil {
		if c.isAdmin(c.update.Message.ReplyToMessage.From.ID) {
			c.replyUserIsAdmin = true
		}
	}
	// c.mustReply = true 必须处理指定消息
	if c.mustReply && c.update.Message.ReplyToMessage == nil {
		c.messageConfig.Text = "搞空气！"
		c.sendCommandMessage()
		return false
	}
	// c.mustAdmin = true 管理员才能使用的命令
	if c.mustAdmin && !c.userIsAdmin {
		c.messageConfig.Text = "你不行！"
		c.sendCommandMessage()
		return false
	}
	// c.mustAdminCanRestrictMembers = true 有封禁权限的管理员才能使用的命令
	if c.mustAdminCanRestrictMembers && !c.userIsRestrictAdmin {
		c.messageConfig.Text = "你不中！"
		c.sendCommandMessage()
		return false
	}
	if c.update.Message.ReplyToMessage != nil {
		// 不能处理管理员的消息
		if c.replyUserIsAdmin && !c.canHandleAdminReply {
			c.messageConfig.Text = "你太弱！"
			c.sendCommandMessage()
			return false
		}
		// c.canHandleSelf = true 允许处理自己的消息
		if c.canHandleSelf && c.update.Message.From.ID == c.update.Message.ReplyToMessage.From.ID {
			if c.userIsAdmin && !c.canHandleNoAdminReply {
				c.messageConfig.Text = "搞不了！"
				c.sendCommandMessage()
				return false
			}
		} else {
			// 处理的不是自己的消息则必须是管理员 (c.canHandleNoAdminReply = true 除外)
			if !c.userIsAdmin && !c.canHandleNoAdminReply {
				c.messageConfig.Text = "搞不成！"
				c.sendCommandMessage()
				return false
			}
		}
		c.handleUserName = c.update.Message.ReplyToMessage.From.FirstName
		c.handleUserID = c.update.Message.ReplyToMessage.From.ID
	} else {
		// c.canHandleAdmin = true 允许管理员发送单独指令
		if !c.canHandleAdmin && c.userIsAdmin {
			c.messageConfig.Text = "不受理！"
			c.sendCommandMessage()
			return false
		}
		c.handleUserName = c.update.Message.From.FirstName
		c.handleUserID = c.update.Message.From.ID
	}
	return true
}

func init() {
	defer func() {
		for i := range commandsPrivateFunc {
			logrus.Infof("registr_private_command:%v", i)
		}
		for i := range commandsGroupFunc {
			logrus.Infof("registr_group_command:%v", i)
		}
	}()
	commandsGroupFunc["studytop"] = func(c *CommandConfig) {
		c.studyTopCommand()
	}
	commandsGroupFunc["marstop"] = func(c *CommandConfig) {
		c.marsTopCommand()
	}
	commandsGroupFunc["study"] = func(c *CommandConfig) {
		c.studyCommand()
	}
	commandsGroupFunc["ban"] = func(c *CommandConfig) {
		c.banCommand()
	}
	commandsGroupFunc["dban"] = func(c *CommandConfig) {
		c.dbanCommand()
	}
	commandsGroupFunc["unban"] = func(c *CommandConfig) {
		c.unBanCommand()
	}
	commandsGroupFunc["rt"] = func(c *CommandConfig) {
		c.rtCommand()
	}
	commandsGroupFunc["unrt"] = func(c *CommandConfig) {
		c.unRtCommand()
	}
	commandsGroupFunc["warn"] = func(c *CommandConfig) {
		c.warnCommand()
	}
	commandsGroupFunc["unwarn"] = func(c *CommandConfig) {
		c.unWarnCommand()
	}
	commandsPrivateFunc["start"] = func(c *CommandConfig) {
		c.startCommand()
	}
	commandsGroupFunc["enable"] = func(c *CommandConfig) {
		c.enableCommand()
	}
	commandsGroupFunc["disable"] = func(c *CommandConfig) {
		c.disableCommand()
	}
	commandsGroupFunc["doudou"] = func(c *CommandConfig) {
		c.doudouCommand()
	}
	commandsGroupFunc["doudoutop"] = func(c *CommandConfig) {
		c.doudouTopCommand()
	}
	commandsGroupFunc["clear_my_48h_message"] = func(c *CommandConfig) {
		c.clearMy48hMessageCommand()
	}
	commandsGroupFunc["honortop"] = func(c *CommandConfig) {
		c.honorTopCommand()
	}
	commandsGroupFunc["chat_blacklist"] = func(c *CommandConfig) {
		c.chatBlacklistCommand()
	}
	commandsGroupFunc["chat_user_activity"] = func(c *CommandConfig) {
		c.chatUserActivityCommand()
	}
	commandsGroupFunc["bot_shareholders"] = func(c *CommandConfig) {
		c.botShareholdersCommand()
	}
	commandsGroupFunc["bot_be_shareholder"] = func(c *CommandConfig) {
		c.botBeShareholderCommand()
	}
	commandsGroupFunc["all"] = func(c *CommandConfig) {}
	commandsGroupFunc["chat_mars"] = func(c *CommandConfig) {}
	commandsGroupFunc["chat_member_verify"] = func(c *CommandConfig) {}
	commandsGroupFunc["chat_userprofile_watch"] = func(c *CommandConfig) {}
	commandsGroupFunc["chat_limit"] = func(c *CommandConfig) {}
	commandsGroupFunc["hei_wu_lei"] = func(c *CommandConfig) {
		c.heiWuLeiCommand()
	}
	commandsGroupFunc["ping_fan"] = func(c *CommandConfig) {
		c.pingFanCommand()
	}
	commandsGroupFunc["power"] = func(c *CommandConfig) {
		c.powerCommand()
	}
}
