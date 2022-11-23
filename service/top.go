package service

import (
	"athenabot/db"
	"athenabot/util"
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"time"
)

type topConfig struct {
	*BotConfig
	ctx context.Context
}

func newTopConfig(ctx context.Context, botConfig *BotConfig) *topConfig {
	return &topConfig{
		ctx:       ctx,
		BotConfig: botConfig,
	}
}

func (c *topConfig) setTop(topKey string, userID int64, addScore float64) {
	key := util.StrBuilder(topKey + util.NumToStr(c.update.Message.Chat.ID))
	nowTimestampInt := time.Now().UnixMilli()
	nowTimestampFloat, _ := strconv.ParseFloat("0."+util.NumToStr(nowTimestampInt), 64)
	var score float64
	resScore, _ := db.RDB.ZScore(c.ctx, key, util.NumToStr(userID)).Result()
	if resScore > 0 {
		score = addScore + float64(int(resScore)) + nowTimestampFloat
	} else {
		score = addScore + nowTimestampFloat
	}
	if err := db.RDB.ZAdd(c.ctx, key, &redis.Z{
		Score:  score,
		Member: userID,
	}).Err(); err != nil {
		logrus.Error(err)
	}

	memberTotal, err := db.RDB.ZCard(c.ctx, key).Result()
	if err != nil {
		logrus.Error(err)
	}
	if memberTotal > 1000 {
		if err := db.RDB.ZRemRangeByRank(c.ctx, key, 0, memberTotal-1001).Err(); err != nil {
			logrus.Error(err)
		}
	}
}

func (c *topConfig) getTop(topKey string, topPrefix, topSuffix string) {
	key := util.StrBuilder(topKey, util.NumToStr(c.update.Message.Chat.ID))
	var topText string
	resTopUser, err := db.RDB.ZRevRange(c.ctx, key, 0, 9).Result()
	if err != nil {
		logrus.Error(err)
	}
	func() {
		wg := new(sync.WaitGroup)
		for _, userId := range resTopUser {
			id, _ := strconv.ParseInt(userId, 10, 64)
			wg.Add(1)
			go c.getUserNameCache(wg, id)
		}
		wg.Wait()
	}()
	for _, userId := range resTopUser {
		id, _ := strconv.ParseInt(userId, 10, 64)
		score, err := db.RDB.ZScore(c.ctx, key, userId).Result()
		if err != nil {
			logrus.Error(err)
		}
		var firstName string
		if v, ok := userNameCache[id]; ok {
			firstName = v
		} else {
			firstName = "无名氏"
			if _, ok := unknownUserCache[id]; ok {
				err := db.RDB.ZRem(c.ctx, topText, id).Err()
				if err != nil {
					logrus.Error(err)
				}
			}
		}
		topText += util.StrBuilder(firstName, " ", topPrefix, util.NumToStr(score), topSuffix, "\n")
	}
	c.messageConfig.Text = topText
	c.sendMessage()
}
