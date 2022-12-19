package service

import (
	"athenabot/client"
	"athenabot/config"
	"athenabot/util"
	"github.com/bitly/go-simplejson"
	"io"
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
