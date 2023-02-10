package service

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
	userNameCacheDir               = "bot:user_name_cache:"
	administratorsCacheDir         = "bot:administrators_cache:"
	chatUserprofileWatchDir        = "bot:chat_userprofile_watch:"
)

var (
	commandsFunc    = make(map[string]func(c *CommandConfig))
	groupsChatLimit = make(map[int64]*chatLimit)
)

type userNameCache struct {
	userName map[int64]string
}
