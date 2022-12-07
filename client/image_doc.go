package client

import (
	"athenabot/model"
	"github.com/sirupsen/logrus"
	"sync"
)

type ImageDocClient interface {
	addImageDoc(*model.ImageDoc) error
	searchImageDoc(int64, string) ([]*model.ImageDoc, error)
}

func AddImageDoc(client ImageDocClient, imageDoc *model.ImageDoc) error {
	if err := client.addImageDoc(imageDoc); err != nil {
		return err
	}
	return nil
}

func SearchImageDoc(client ImageDocClient, chatID int64, phrase string) ([]*model.ImageDoc, error) {
	imageDocs, err := client.searchImageDoc(chatID, phrase)
	if err != nil {
		return imageDocs, err
	}
	return imageDocs, nil
}

var ImageDocProvider = make(map[string]func(string) ImageDocClient)

var docClient ImageDocClient
var docClientOnce sync.Once

func init() {
	defer func() {
		for i := range ImageDocProvider {
			logrus.Infof("registr_doc_provider:%v", i)
		}
	}()
	ImageDocProvider["mongo"] = func(url string) ImageDocClient {
		docClientOnce.Do(func() {
			c, err := newMongodbClient(url)
			if err != nil {
				logrus.Panic(err)
			}
			docClient = c
			logrus.Infof("new mongo_client:%+v", c)
		})
		return docClient
	}
	ImageDocProvider["mysql"] = func(url string) ImageDocClient {
		docClientOnce.Do(func() {
			c, err := newMysqlClient(url)
			if err != nil {
				logrus.Panic(err)
			}
			docClient = c
			logrus.Infof("new mysql_client:%+v", c)
		})
		return docClient
	}
	ImageDocProvider["es"] = func(url string) ImageDocClient {
		docClientOnce.Do(func() {
			es, err := newEsClient(url)
			if err != nil {
				logrus.Panic(err)
			}
			docClient = es
			logrus.Infof("new es_client:%+v", es)
		})
		return docClient
	}
}
