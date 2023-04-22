package service

import (
	"athenabot/client"
	"athenabot/config"
	"athenabot/db"
	"athenabot/util"
	"encoding/json"
	"github.com/bitly/go-simplejson"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

func getImageOCR(image io.Reader) ([]string, error) {
	ocrReq := client.NewRequestFD(config.Conf.MarsOCR.OcrURL, "POST")
	ocrReq.TimeOut = time.Second * 30
	ocrReq.File = &client.File{
		FileName: "file",
		File:     image,
	}
	ocrReq.Head = make(map[string]string)
	ocrReq.Head["ocr-type"] = config.Conf.MarsOCR.OcrProvider
	imagePhrasesRes, err := ocrReq.Do()
	if err != nil {
		return nil, err
	}
	resJson := &simplejson.Json{}
	resJson, err = simplejson.NewJson(imagePhrasesRes)
	if err != nil {
		return nil, err
	}
	return resJson.Get("image_phrases").MustStringArray(), nil
}

func generateSimpleImagePhrases(strArray []string) []string {
	const (
		minLen = 15
		tag    = "___tag___"
	)
	var res []string
	for _, str := range strArray {
		if len(str) < minLen {
			if len(res) == 0 {
				res = append(res, str)
			} else {
				res[len(res)-1] = util.StrBuilder(res[len(res)-1], tag, str)
			}
		} else {
			if len(res) == 0 || len(res[len(res)-1]) > minLen {
				res = append(res, str)
			} else {
				res[len(res)-1] = util.StrBuilder(res[len(res)-1], tag, str)
			}
		}
	}
	return res
}

func generateCallbackData(command string, userID int64, messageID int) string {
	callbackData := CallbackData{Command: command, UserID: userID, MsgID: messageID}
	data, _ := json.Marshal(callbackData)
	str := string(data)
	str = strings.ReplaceAll(str, " ", "")

	return str
}

func ParseCallbackData(data string) (*CallbackData, error) {
	callbackData := &CallbackData{}
	err := json.Unmarshal([]byte(data), callbackData)
	if err != nil {
		return nil, err
	}
	return callbackData, nil
}

func (c *BotConfig) generateUserActivityData() ([]userActivity, error) {
	key := util.StrBuilder(chatUserActivityDir, util.NumToStr(c.update.Message.Chat.ID))
	userActivityRes, err := db.RDB.ZRange(c.ctx, key, 0, 10).Result()
	if err != nil {
		return nil, err
	}
	var userActivityData []userActivity

	for _, userID := range userActivityRes {
		lastTimeRes, err := db.RDB.ZScore(c.ctx, key, userID).Result()
		if err != nil {
			continue
		}
		nowTime := time.Now().Unix()
		lastTime := int64(lastTimeRes)
		if nowTime-lastTime > 86400*30 {
			day := float64(nowTime-lastTime) / 86400

			userID, err := strconv.ParseInt(userID, 10, 64)
			if err != nil {
				continue
			}

			userNameCache := &userNameCache{
				userName: make(map[int64]string),
			}
			wg := new(sync.WaitGroup)
			wg.Add(1)
			c.getUserNameCache(wg, userID, userNameCache)

			var fullName string
			if name, ok := userNameCache.userName[userID]; ok {
				fullName = name
			} else {
				fullName = "unknown"
			}

			userActivityData = append(userActivityData, userActivity{
				userID:       userID,
				fullName:     fullName,
				inactiveDays: int(day),
			})
		}

	}

	return userActivityData, nil
}
