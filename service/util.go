package service

import (
	"athenabot/client"
	"athenabot/config"
	"athenabot/db"
	"athenabot/util"
	"encoding/json"
	"errors"
	"github.com/bitly/go-simplejson"
	"io"
	"strconv"
	"strings"
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

func generateCallbackData(command string, userID int64, data any) string {
	callbackData := CallbackData{Command: command, UserID: userID, Data: data}
	d, _ := json.Marshal(callbackData)
	str := string(d)
	str = strings.ReplaceAll(str, " ", "")
	return str
}

func ParseCallbackData(data string) (*CallbackData, error) {
	callbackData := &CallbackData{}
	err := json.Unmarshal([]byte(data), callbackData)
	if err != nil {
		return nil, err
	}
	if len(callbackData.Command) == 0 {
		return nil, errors.New("callback data command is nil")
	}
	return callbackData, nil
}

func (c *BotConfig) generateUserActivityData() ([]userActivity, error) {
	key := util.StrBuilder(chatUserActivityDir, util.NumToStr(c.chatID))
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
		day := float64(nowTime-lastTime) / 86400

		userID, err := strconv.ParseInt(userID, 10, 64)
		if err != nil {
			continue
		}

		userActivityData = append(userActivityData, userActivity{
			userID:       userID,
			fullName:     c.getUserCache(userID).User.FirstName,
			inactiveDays: int(day),
		})

	}

	return userActivityData, nil
}
