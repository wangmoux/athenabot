package service

import "sync"

const (
	marsKeyDir                     = "bot:mars_data:"
	marsTopKeyDir                  = "bot:mars_top:"
	studyTopKeyDir                 = "bot:study_top:"
	warnKeyDir                     = "bot:warn_data:"
	chatVerifyKeyDir               = "bot:chat_verify:"
	serviceSwitchKeyDir            = "bot:service_switch:"
	commandLimitKeyDir             = "bot:command_limit:"
	deleteMessageKeyDir            = "bot:delete_message:"
	doudouTopKeyDir                = "bot:doudou_top:"
	chat48hMessageDir              = "bot:chat_48h_message:"
	chat48hMessageDeleteCrontabDir = "bot:chat_48h_message_delete_crontab:"
	administratorsCacheDir         = "bot:administrators_cache:"
	usersCacheDir                  = "bot:users_cache:"
	chatUserprofileWatchDir        = "bot:chat_userprofile_watch:"
	chatBlacklistDir               = "bot:chat_blacklist:"
	chatUserActivityDir            = "bot:chat_user_activity:"
	shareholdersDir                = "bot:shareholders:"
	sudoAdministratorsDir          = "bot:sudo_administrators"
	heiWuLeiDir                    = "bot:hei_wu_lei:"
	groupWhitelistDir              = "bot:group_whitelist:"
	marsWhitelistDir               = "bot:mars_whitelist:"
	marsOCRWhitelistDir            = "bot:mars_ocr_whitelist:"
	chatGuardWhitelistDir          = "bot:chat_guard_whitelist:"
	chatBotWhitelistDir            = "bot:chat_bot_whitelist:"
)

const (
	userIsNotFoundMessage = "user not found"
)

var (
	commandsGroupFunc         = make(map[string]func(c *CommandConfig))
	commandsPrivateFunc       = make(map[string]func(c *CommandConfig))
	groupsChatLimit           = make(map[int64]*chatLimit)
	sudoAdminOnce             sync.Once
	inlineQueryResultArticles []any
)

type CallbackData struct {
	Command string `json:"c"`
	UserID  int64  `json:"u"`
	Data    any    `json:"d,omitempty"`
}

type userActivity struct {
	userID       int64
	fullName     string
	inactiveDays int
}
