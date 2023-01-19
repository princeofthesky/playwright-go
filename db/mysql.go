package db

import (
	"context"
	"github.com/go-pg/pg/v10"
	"github.com/playwright-community/playwright-go/config"
)

var mysqlDb *pg.DB

func Init(postgres config.Postgres) error {
	mysqlDb = pg.Connect(&pg.Options{
		Addr:     postgres.Addr,
		User:     postgres.User,
		Password: postgres.Password,
		Database: postgres.Database,
	})
	return mysqlDb.Ping(context.Background())
}
func Close() {
	mysqlDb.Close()
}
func GetDb() *pg.DB {
	return mysqlDb
}

func GetAudioById(id int) (Audio, error) {
	info := Audio{Id: id}
	err := mysqlDb.Model(&info).Where("\"audio\".\"id\"=?", info.Id).Select()
	return info, err
}

func CountAudios() (int, error) {

	number, err := mysqlDb.Model(&Audio{}).Count()
	return number, err
}
func GetAudioByTikTokUrl(tiktokUrl string) (Audio, error) {
	info := Audio{TiktokUrl: tiktokUrl}
	err := mysqlDb.Model(&info).Where("\"audio\".\"tiktok_url\"=?", info.TiktokUrl).Select()
	return info, err
}

func InsertAudioInfo(info Audio) (Audio, error) {
	_, err := mysqlDb.Model(&info).Insert()
	return info, err
}

func InsertAudioToListNew(data NewAudio) error {
	_, err := mysqlDb.Model(&data).Insert()
	return err
}

func GetListNewAudioId(offset int, length int) ([]NewAudio, error) {
	data := []NewAudio{}
	err := mysqlDb.Model((*NewAudio)(nil)).Column("*").Where("crawled_time < '?'", offset).Limit(length).Order("crawled_time DESC").ForEach(
		func(c *NewAudio) error {
			data = append(data, *c)
			return nil
		})
	return data, err
}
