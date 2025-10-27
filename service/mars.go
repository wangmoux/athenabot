package service

import (
	"athenabot/client"
	"athenabot/config"
	"athenabot/db"
	"athenabot/model"
	"athenabot/util"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type mars struct {
	Count int `json:"count,omitempty"`
	MsgID int `json:"msg_id"`
}

type MarsConfig struct {
	*BotConfig
	latestMars  mars
	currentMars mars
	marsID      string
}

func NewMarsConfig(botConfig *BotConfig) *MarsConfig {
	return &MarsConfig{BotConfig: botConfig}
}

func (c *MarsConfig) getMars() {
	marsKey := util.StrBuilder(marsKeyDir, util.NumToStr(c.chatID), ":", c.marsID)
	res, err := db.RDB.Get(c.ctx, marsKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	_ = json.Unmarshal([]byte(res), &c.latestMars)
}

func (c *MarsConfig) setMars() {
	marsKey := util.StrBuilder(marsKeyDir, util.NumToStr(c.chatID), ":", c.marsID)
	c.currentMars.MsgID = c.update.Message.MessageID
	logrus.Infof("msg_id: %v mars_count: %v mars_id: %v", c.currentMars.MsgID, c.currentMars.Count, c.marsID)
	marsJson, _ := json.Marshal(c.currentMars)
	if err := db.RDB.Set(c.ctx, marsKey, marsJson, time.Second*time.Duration(config.Conf.KeyTTL)).Err(); err != nil {
		logrus.Error(err)
	}
}

func (c *MarsConfig) isMarsExists() bool {
	marsKey := util.StrBuilder(marsKeyDir, util.NumToStr(c.chatID), ":", c.marsID)
	res, err := db.RDB.Exists(c.ctx, marsKey).Result()
	if err != nil {
		logrus.Error(err)
	}
	if res > 0 {
		return true
	}
	return false
}

func (c *MarsConfig) handleMars() {
	c.botMessageCleanCountdown = 0
	c.getMars()
	c.currentMars.Count = c.latestMars.Count + 1
	c.setMars()

	humanChatID := c.chatID - c.chatID - c.chatID - 1000000000000
	LatestMarsMessage := util.StrBuilder("https://t.me/c/", util.NumToStr(humanChatID), "/", util.NumToStr(c.latestMars.MsgID))
	c.messageConfig.Text = util.StrBuilder("这条消息火星", util.NumToStr(c.currentMars.Count), "次了哈")
	c.messageConfig.Entities = []tgbotapi.MessageEntity{{
		Type:   "text_link",
		URL:    LatestMarsMessage,
		Offset: 6,
		Length: 2,
	},
	}
	delMarsCallbackData := generateCallbackData("delete-chat-msg-clean", c.update.Message.From.ID, c.update.Message.MessageID)
	getMarsCallbackData := generateCallbackData("get-user-mars", c.update.Message.From.ID, c.update.Message.MessageID)
	delMarsConfirmButton := tgbotapi.NewInlineKeyboardButtonData("悄悄删掉", delMarsCallbackData)
	getMarsConfirmButton := tgbotapi.NewInlineKeyboardButtonData("无动于衷", getMarsCallbackData)
	replyMarkup := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{delMarsConfirmButton, getMarsConfirmButton},
	)
	c.messageConfig.ReplyMarkup = replyMarkup
	c.sendReplyMessage()
	logrus.Infof("mars_user:%v mars_id:%v", c.update.Message.From.ID, c.marsID)
	t := newTopConfig(c.BotConfig)
	t.setTop(marsTopKeyDir, c.update.Message.From.ID, 1)

}

func (c *MarsConfig) HandlePhoto() {
	fileByte, err := util.GetBotFile(c.bot, c.update.Message.Photo[len(c.update.Message.Photo)-1].FileID)
	if err != nil {
		logrus.Error(err)
		return
	}
	pHash, err := util.GetFilePHash(bytes.NewBuffer(fileByte))
	if err != nil {
		logrus.Error(err)
		return
	}
	c.marsID = util.StrBuilder("photo:phash:", util.NumToStr(pHash))

	if c.isMarsExists() {
		c.handleMars()
	} else {
		if config.Conf.MarsOCR.EnableOCR && c.IsEnableChatService("chat_mars_ocr") && c.IsMarsOCRWhitelist(c.update.Message.Chat.UserName) {
			imagePhrases, err := getImageOCR(bytes.NewBuffer(fileByte))
			if err != nil {
				logrus.Error(err)
				return
			}
			c.handleImageDoc(imagePhrases, pHash)
		} else {
			c.setMars()
		}
	}
}

func (c *MarsConfig) handleVideoFileHash() {
	var fileByte []byte
	var err error
	fileSize := c.update.Message.Video.FileSize
	const kb = 64
	if fileSize < 1024*kb+1 {
		fileByte, err = util.GetBotFile(c.bot, c.update.Message.Video.FileID)
	} else {
		fileRange := fmt.Sprintf("bytes=%d-%d", fileSize-1024*kb, fileSize)
		fileByte, err = util.GetBotFile(c.bot, c.update.Message.Video.FileID, fileRange)
	}
	if err != nil {
		logrus.Error(err)
		return
	}
	var s []byte
	s = append(s, util.NumToStr(fileSize)...)
	s = append(s, fileByte...)
	h := sha256.Sum256(s)
	c.marsID = util.StrBuilder("video:end", util.NumToStr(kb), "k_sha256:", hex.EncodeToString(h[:]))
	if c.isMarsExists() {
		c.handleMars()
	} else {
		c.setMars()
	}
}

func (c *MarsConfig) HandleVideo() {
	//c.marsID = util.StrBuilder("video:file_unique_id:", c.update.Message.Video.FileUniqueID)
	//if c.isMarsExists() {
	//	c.handleMars()
	//} else {
	//	c.setMars()
	//}
	c.handleVideoFileHash()
}

func (c *MarsConfig) handleText(s string) {
	h := sha256.Sum256([]byte(s))
	c.marsID = util.StrBuilder("text:sha256:", hex.EncodeToString(h[:]))
	if c.isMarsExists() {
		c.handleMars()
	} else {
		c.setMars()
	}
}

func (c *MarsConfig) HandleText() {
	if c.update.Message.ForwardFromChat == nil {
		return
	}
	if c.update.Message.ForwardFromChat.Type != "channel" {
		return
	}
	if len(c.update.Message.Text) < 100 {
		return
	}
	c.handleText(c.update.Message.Text)
}

func (c *MarsConfig) handleImageDoc(imagePhrases []string, pHash uint64) {
	var noSetMars bool
	defer func() {
		if !noSetMars {
			c.setMars()
		}
	}()

	logrus.Debugf("image_phrases:%v", imagePhrases)

	simpleImagePhrases := generateSimpleImagePhrases(imagePhrases)
	imageDoc := model.ImageDocPool.Get().(*model.ImageDoc)
	imageDoc = &model.ImageDoc{
		MarsID:     util.NumToStr(pHash),
		ChatID:     c.chatID,
		CreateTime: time.Now().UTC(),
	}
	defer model.ImageDocPool.Put(imageDoc)
	imageDoc.ImagePhrases = simpleImagePhrases
	logrus.Debugf("simple_image_phrases:%v", simpleImagePhrases)

	simpleImagePhrasesLen := len(simpleImagePhrases)
	if simpleImagePhrasesLen < config.Conf.MarsOCR.MinPhrase {
		return
	}
	imageDocClient := client.ImageDocProvider[config.Conf.MarsOCR.DocProvider](config.Conf.MarsOCR.DocURL)
	hitPhraseMap := make(map[string]int)

	for _, phrase := range simpleImagePhrases {
		imageDocs, err := client.SearchImageDoc(imageDocClient, imageDoc.ChatID, phrase)
		if err != nil {
			logrus.Error(err)
			continue
		}
		for _, item := range imageDocs {
			lenRatio := float32(simpleImagePhrasesLen) / float32(len(item.ImagePhrases))
			if lenRatio > config.Conf.MarsOCR.MinHitRatio && lenRatio < 1-config.Conf.MarsOCR.MinHitRatio+1 {
				if _, ok := hitPhraseMap[item.MarsID]; ok {
					hitPhraseMap[item.MarsID] += 1
				} else {
					hitPhraseMap[item.MarsID] = 1
				}
			}
		}
	}

	var hitMarsID string
	var maxHit int
	for k, v := range hitPhraseMap {
		if v > maxHit {
			maxHit = v
			hitMarsID = k
		}
	}
	logrus.Infof("hit_phrase_map:%v max_hit:%v simple_image_phrases_len:%v", hitPhraseMap, maxHit, simpleImagePhrasesLen)

	hitRatio := float32(maxHit) / float32(simpleImagePhrasesLen)
	if hitRatio > config.Conf.MarsOCR.MinHitRatio {
		noSetMars = true
		c.marsID = hitMarsID
		if c.isMarsExists() {
			c.handleMars()
		}
	} else {
		if err := client.AddImageDoc(imageDocClient, imageDoc); err != nil {
			logrus.Error(err)
		}
	}
}
