package client

import (
	"athenabot/model"
	"athenabot/util"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"strings"
	"time"
)

type MysqlClient struct {
	*gorm.DB
}

func newMysqlClient(url string) (ImageDocClient, error) {
	db, err := gorm.Open(mysql.Open(url), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Second * 600)
	return &MysqlClient{DB: db}, err
}

func (m *MysqlClient) addImageDoc(doc *model.ImageDoc) error {
	_doc := &model.MysqlImageDoc{
		ChatID:     doc.ChatID,
		MarsID:     doc.MarsID,
		CreateTime: doc.CreateTime.Format("2006-01-02 15:04:05"),
	}
	for _, i := range doc.ImagePhrases {
		_doc.ImagePhrases += util.StrBuilder(i, "\n")
	}
	if err := m.Create(_doc).Error; err != nil {
		return err
	}
	return nil
}

func (m *MysqlClient) searchImageDoc(chatID int64, phrase string) ([]*model.ImageDoc, error) {
	var _imageDocs []model.MysqlImageDoc
	var imageDocs []*model.ImageDoc
	defer func() {
		for _, i := range imageDocs {
			model.ImageDocPool.Put(i)
		}
	}()
	sql := util.StrBuilder("MATCH (image_phrases) AGAINST (", "'\"", phrase, "\"'", " IN BOOLEAN MODE) AND chat_id = ?")
	err := m.Limit(searchLimit).Where(sql, chatID).Find(&_imageDocs).Error
	if err != nil {
		return imageDocs, err
	}
	for _, item := range _imageDocs {
		t, _ := time.Parse("2006-01-02 15:04:05", item.CreateTime)
		imageDoc := model.ImageDocPool.Get().(*model.ImageDoc)
		imageDoc.ImagePhrases = strings.Split(strings.TrimSpace(item.ImagePhrases), "\n")
		imageDoc.MarsID = item.MarsID
		imageDoc.ChatID = item.ChatID
		imageDoc.CreateTime = t
		imageDocs = append(imageDocs, imageDoc)
	}
	return imageDocs, nil
}
