package service

import (
	"athenabot/config"
	"athenabot/db"
	"athenabot/util"
	"context"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

type CommandConfig struct {
	*BotConfig
	ctx              context.Context
	command          string
	commandArg       string
	handleUserName   string
	handleUserID     int64
	mustReply        bool
	mustAdmin        bool
	canHandleAdmin   bool
	userIsAdmin      bool
	replyUserIsAdmin bool
	canHandleSelf    bool
}

func NewCommandConfig(ctx context.Context, botConfig *BotConfig) (commandConfig *CommandConfig) {
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
		ctx:        ctx,
		BotConfig:  botConfig,
		command:    commandFull[0],
		commandArg: arg,
	}
	return commandConfig
}

func (c *CommandConfig) InCommands() {
	if _, ok := config.CommandsMap[c.command]; ok {
		commandSwitchKey := util.StrBuilder(commandSwitchKeyDir, util.NumToStr(c.update.Message.Chat.ID), ":disable_")
		res, err := db.RDB.Exists(c.ctx, commandSwitchKey+c.command).Result()
		if err != nil {
			logrus.Error(err)
			return
		}
		if res == 0 {
			logrus.Infof("command_user=%v command=%s command_arg=%s", c.update.Message.From.ID, c.command, c.commandArg)
			commandsFunc[c.command](c)
		} else {
			c.messageConfig.Text = "该命令已禁用"
			c.sendMessage()
		}
	}
}

func (c *CommandConfig) InPrivateCommands() {
	if _, ok := config.PrivateCommandsMap[c.command]; ok {
		logrus.Infof("command_user=%v command=%s command_arg=%s", c.update.Message.From.ID, c.command, c.commandArg)
		commandsFunc[c.command](c)
	}
}

func (c *CommandConfig) commandLimitAdd(addCount int) bool {
	var count int
	commandLimitKey := util.StrBuilder(commandLimitKeyDir, util.NumToStr(c.update.Message.Chat.ID), ":"+c.command, "_", util.NumToStr(c.handleUserID))
	res, err := db.RDB.Exists(c.ctx, commandLimitKey).Result()
	if err != nil {
		logrus.Error(err)
	}
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
	err = db.RDB.Set(c.ctx, commandLimitKey, count, 86400*time.Second).Err()
	if err != nil {
		logrus.Error(err)
	}
	return false
}

func (c *CommandConfig) isLimitCommand(limit int) bool {
	var count int
	commandLimitKey := util.StrBuilder(commandLimitKeyDir, util.NumToStr(c.update.Message.Chat.ID), ":"+c.command, "_", util.NumToStr(c.handleUserID))
	res, err := db.RDB.Exists(c.ctx, commandLimitKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	if res > 0 {
		resCount, err := db.RDB.Get(c.ctx, commandLimitKey).Result()
		if err != nil {
			logrus.Error(err)
		}
		count, err = strconv.Atoi(resCount)
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
		c.messageConfig.Text = "乐！"
		c.sendMessage()
		return false
	}
	// c.mustAdmin = true 管理员才能使用的命令
	if c.mustAdmin && !c.userIsAdmin {
		c.messageConfig.Text = "仁！"
		c.sendMessage()
		return false
	}
	if c.update.Message.ReplyToMessage != nil {
		// 不能处理管理员的消息
		if c.replyUserIsAdmin {
			c.messageConfig.Text = "义！"
			c.sendMessage()
			return false
		}
		// c.canHandleSelf = true 允许处理自己的消息
		if c.canHandleSelf && c.update.Message.From.ID == c.update.Message.ReplyToMessage.From.ID {
			if c.userIsAdmin {
				c.messageConfig.Text = "礼！"
				c.sendMessage()
				return false
			}
		} else {
			// 处理的不是自己的消息则必须是管理员
			if !c.userIsAdmin {
				c.messageConfig.Text = "智！"
				c.sendMessage()
				return false
			}
		}
		c.handleUserName = c.update.Message.ReplyToMessage.From.FirstName
		c.handleUserID = c.update.Message.ReplyToMessage.From.ID
	} else {
		// c.canHandleAdmin = true 允许管理员发送单独指令
		if !c.canHandleAdmin && c.userIsAdmin {
			c.messageConfig.Text = "信！"
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
			logrus.Infof("registr_command=%v", i)
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
		c.start()
	}
	commandsFunc["enable"] = func(c *CommandConfig) {
		c.enable()
	}
	commandsFunc["disable"] = func(c *CommandConfig) {
		c.disable()
	}
}
