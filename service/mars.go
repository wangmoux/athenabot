package service

import (
	"athenabot/client"
	"athenabot/config"
	"athenabot/db"
	"athenabot/model"
	"athenabot/util"
	"bytes"
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"time"
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
	c.messageConfig.Text = util.StrBuilder("这条愚蠢的消息已经火星", util.NumToStr(c.currentMars.Count), "次了！！!")
	c.messageConfig.Entities = []tgbotapi.MessageEntity{{
		Type:   "text_link",
		URL:    LatestMarsMessage,
		Offset: 9,
		Length: 2,
	},
	}
	c.sendCommandMessage()
	logrus.Infof("mars_user:%v mars_id:%v", c.update.Message.From.ID, c.marsID)
	t := newTopConfig(c.BotConfig)
	t.setTop(marsTopKeyDir, c.update.Message.From.ID, 1)

}

func (c *MarsConfig) HandlePhoto() {
	fileUrl, err := c.bot.GetFileDirectURL(c.update.Message.Photo[len(c.update.Message.Photo)-1].FileID)
	if err != nil {
		logrus.Error(err)
		return
	}
	fileResponse, err := util.GetFileResponse(fileUrl)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer fileResponse.Body.Close()
	fileByte, _ := ioutil.ReadAll(fileResponse.Body)
	pHash, err := util.GetFilePHash(bytes.NewBuffer(fileByte))
	if err != nil {
		logrus.Error(err)
		return
	}
	c.marsID = util.NumToStr(pHash)
	logrus.Infof("mars_id:%v", c.marsID)

	if c.isMarsExists() {
		c.handleMars()
	} else {
		if config.Conf.MarsOCR.EnableOCR && c.IsEnableChatService("chat_mars_ocr") {
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

func (c *MarsConfig) HandleVideo() {
	c.marsID = c.update.Message.Video.FileUniqueID
	if c.isMarsExists() {
		c.handleMars()
	} else {
		c.setMars()
	}
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
