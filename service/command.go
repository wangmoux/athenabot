package service

import (
	"athenabot/config"
	"athenabot/db"
	"athenabot/util"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
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
}

func NewCommandConfig(botConfig *BotConfig) (commandConfig *CommandConfig) {
	commandFull := strings.FieldsFunc(botConfig.update.Message.Text, func(r rune) bool {
		return r == '/' || r == ' ' || r == '@'
	})
	var arg string
	if len(commandFull) == 2 {
		if commandFull[1] != botConfig.bot.Self.UserName {
			arg = commandFull[1]
		}
	}
	if len(commandFull) == 3 {
		if commandFull[1] == botConfig.bot.Self.UserName {
			arg = commandFull[2]
		}
	}
	commandConfig = &CommandConfig{
		BotConfig:  botConfig,
		command:    commandFull[0],
		commandArg: arg,
	}
	return commandConfig
}

func (c *CommandConfig) InCommands() {
	if _, ok := config.CommandsMap[c.command]; ok {
		if _, _ok := commandsFunc[c.command]; !_ok {
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
		if c.IsEnableChatService(c.command) {
			logrus.Infof("command_user:%v command:%s command_arg:%s", c.update.Message.From.ID, c.command, c.commandArg)
			commandsFunc[c.command](c)
		} else {
			logrus.Warnf("command disabled:%v", c.command)
		}
	}
}

func (c *CommandConfig) InPrivateCommands() {
	if _, ok := config.PrivateCommandsMap[c.command]; ok {
		if _, _ok := commandsFunc[c.command]; !_ok {
			logrus.Warnf("command not registered:%v", c.command)
			return
		}
		logrus.Infof("command_user:%v command:%s command_arg:%s", c.update.Message.From.ID, c.command, c.commandArg)
		commandsFunc[c.command](c)
	} else {
		logrus.Warnf("command disabled:%v", c.command)
	}
}

func (c *CommandConfig) commandLimitAdd(addCount int) {
	commandLimitKey := util.StrBuilder(commandLimitKeyDir, util.NumToStr(c.update.Message.Chat.ID), ":"+c.command, "_", util.NumToStr(c.update.Message.From.ID))
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
	commandLimitKey := util.StrBuilder(commandLimitKeyDir, util.NumToStr(c.update.Message.Chat.ID), ":"+c.command, "_", util.NumToStr(c.update.Message.From.ID))
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
	if c.update.Message.ReplyToMessage != nil {
		if c.isAdmin(c.update.Message.ReplyToMessage.From.ID) {
			c.replyUserIsAdmin = true
		}
	}
	// c.mustReply = true 必须处理指定消息
	if c.mustReply && c.update.Message.ReplyToMessage == nil {
		c.messageConfig.Text = "搞空气！"
		c.sendMessage()
		return false
	}
	// c.mustAdmin = true 管理员才能使用的命令
	if c.mustAdmin && !c.userIsAdmin {
		c.messageConfig.Text = "你不行！"
		c.sendMessage()
		return false
	}
	if c.update.Message.ReplyToMessage != nil {
		// 不能处理管理员的消息
		if c.replyUserIsAdmin && !c.canHandleAdminReply {
			c.messageConfig.Text = "你太弱！"
			c.sendMessage()
			return false
		}
		// c.canHandleSelf = true 允许处理自己的消息
		if c.canHandleSelf && c.update.Message.From.ID == c.update.Message.ReplyToMessage.From.ID {
			if c.userIsAdmin && !c.canHandleNoAdminReply {
				c.messageConfig.Text = "搞不了！"
				c.sendMessage()
				return false
			}
		} else {
			// 处理的不是自己的消息则必须是管理员 (c.canHandleNoAdminReply = true 除外)
			if !c.userIsAdmin && !c.canHandleNoAdminReply {
				c.messageConfig.Text = "搞不成！"
				c.sendMessage()
				return false
			}
		}
		c.handleUserName = c.update.Message.ReplyToMessage.From.FirstName
		c.handleUserID = c.update.Message.ReplyToMessage.From.ID
	} else {
		// c.canHandleAdmin = true 允许管理员发送单独指令
		if !c.canHandleAdmin && c.userIsAdmin {
			c.messageConfig.Text = "不受理！"
			c.sendMessage()
			return false
		}
		c.handleUserName = c.update.Message.From.FirstName
		c.handleUserID = c.update.Message.From.ID
	}
	return true
}

func init() {
	defer func() {
		for i := range commandsFunc {
			logrus.Infof("registr_command:%v", i)
		}
	}()
	commandsFunc["studytop"] = func(c *CommandConfig) {
		c.studyTopCommand()
	}
	commandsFunc["marstop"] = func(c *CommandConfig) {
		c.marsTopCommand()
	}
	commandsFunc["study"] = func(c *CommandConfig) {
		c.studyCommand()
	}
	commandsFunc["ban"] = func(c *CommandConfig) {
		c.banCommand()
	}
	commandsFunc["dban"] = func(c *CommandConfig) {
		c.dbanCommand()
	}
	commandsFunc["unban"] = func(c *CommandConfig) {
		c.unBanCommand()
	}
	commandsFunc["rt"] = func(c *CommandConfig) {
		c.rtCommand()
	}
	commandsFunc["unrt"] = func(c *CommandConfig) {
		c.unRtCommand()
	}
	commandsFunc["warn"] = func(c *CommandConfig) {
		c.warnCommand()
	}
	commandsFunc["unwarn"] = func(c *CommandConfig) {
		c.unWarnCommand()
	}
	commandsFunc["start"] = func(c *CommandConfig) {
		c.startCommand()
	}
	commandsFunc["enable"] = func(c *CommandConfig) {
		c.enableCommand()
	}
	commandsFunc["disable"] = func(c *CommandConfig) {
		c.disableCommand()
	}
	commandsFunc["doudou"] = func(c *CommandConfig) {
		c.doudouCommand()
	}
	commandsFunc["doudoutop"] = func(c *CommandConfig) {
		c.doudouTopCommand()
	}
	commandsFunc["clear_my_48h_message"] = func(c *CommandConfig) {
		c.clearMy48hMessageCommand()
	}
	commandsFunc["honortop"] = func(c *CommandConfig) {
		c.honorTopCommand()
	}
	commandsFunc["chat_blacklist"] = func(c *CommandConfig) {
		c.chatBlacklistCommand()
	}
	commandsFunc["chat_user_activity"] = func(c *CommandConfig) {
		c.chatUserActivityCommand()
	}
}
