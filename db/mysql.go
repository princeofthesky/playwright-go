package db

import (
	"context"
	"github.com/go-pg/pg/v10"
	"github.com/playwright-community/playwright-go/config"
	"strconv"
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
	err := mysqlDb.Model(&info).Where("tiktok_url=?", info.TiktokUrl).Select()
	return info, err
}

func GetAudioByTikTokId(tiktokId string) (Audio, error) {
	info := Audio{TiktokId: tiktokId}
	err := mysqlDb.Model(&info).Where("tiktok_id=?", info.TiktokId).Select()
	return info, err
}

func InsertAudioInfo(info Audio) (Audio, error) {
	_, err := mysqlDb.Model(&info).Insert()
	return info, err
}

func InsertRegionInfo(info Region) (Region, error) {
	_, err := mysqlDb.Model(&info).Insert()
	if err != nil {
		return info, err
	}
	_, err = mysqlDb.Exec("create table if not exists trending_audios_regions_" + strconv.Itoa(info.Id) +
		"( audio_id int not null primary key, " +
		"updated_time int not null, " +
		"genre_id int not null, " +
		"theme_id int not null, " +
		"duration int not null, " +
		"mood_id int not null );")
	return info, err
}

func GetAllRegions() ([]Region, error) {
	data := []Region{}
	err := mysqlDb.Model((*Region)(nil)).ForEach(
		func(c *Region) error {
			data = append(data, *c)
			return nil
		})
	return data, err
}

func GetRegionByCode(code string) (Region, error) {
	info := Region{Code: code}
	err := mysqlDb.Model(&info).Where("code=?", info.Code).Select()
	return info, err
}

func InsertMoodInfo(info Mood) (Mood, error) {
	_, err := mysqlDb.Model(&info).Insert()
	return info, err
}
func GetAllMoods() ([]Mood, error) {
	data := []Mood{}
	err := mysqlDb.Model((*Mood)(nil)).ForEach(
		func(c *Mood) error {
			data = append(data, *c)
			return nil
		})
	return data, err
}
func GetMoodByTitle(title string) (Mood, error) {
	info := Mood{Title: title}
	err := mysqlDb.Model(&info).Where("title=?", info.Title).Select()
	return info, err
}
func InsertThemeInfo(info Theme) (Theme, error) {
	_, err := mysqlDb.Model(&info).Insert()
	return info, err
}
func GetAllThemes() ([]Theme, error) {
	data := []Theme{}
	err := mysqlDb.Model((*Theme)(nil)).ForEach(
		func(c *Theme) error {
			data = append(data, *c)
			return nil
		})
	return data, err
}
func GetThemeByTitle(title string) (Theme, error) {
	info := Theme{Title: title}
	err := mysqlDb.Model(&info).Where("title=?", info.Title).Select()
	return info, err
}

func InsertGenreInfo(info Genre) (Genre, error) {
	_, err := mysqlDb.Model(&info).Insert()
	return info, err
}
func GetAllGenres() ([]Genre, error) {
	data := []Genre{}
	err := mysqlDb.Model((*Genre)(nil)).ForEach(
		func(c *Genre) error {
			data = append(data, *c)
			return nil
		})
	return data, err
}
func GetGenreByTitle(title string) (Genre, error) {
	info := Genre{Title: title}
	err := mysqlDb.Model(&info).Where("title=?", info.Title).Select()
	return info, err
}
func InsertAudioToListNew(regionId int ,data RegionTrendingAudio) error {
	_, err := mysqlDb.Exec("INSERT INTO trending_audios_regions_" +strconv.Itoa(regionId)+
		" (audio_id, updated_time, theme_id, genre_id, mood_id, duration) VALUES (?, ?, ?, ?, ?, ?);",
		data.AudioId,data.UpdatedTime,data.ThemeId,data.GenreId,data.MoodId,data.Duration)
	return err
}

func GetListNewAudioId(themes, moods, genres []int, region, minDuration, maxDuration, offset, length int) ([]RegionTrendingAudio, error) {
	WhereCondition := "updated_time < " + strconv.Itoa(offset) + " AND region_id = " + strconv.Itoa(region) +
		" AND duration >= " + strconv.Itoa(minDuration) + " AND duration <= " + strconv.Itoa(maxDuration)

	if len(themes) > 0 {
		addedQuery := " AND ( theme_id = " + strconv.Itoa(themes[0])
		for i := 1; i < len(themes); i++ {
			addedQuery = addedQuery + " OR theme_id = " + strconv.Itoa(themes[i])
		}

		WhereCondition = WhereCondition + addedQuery + " ) "
	}

	if len(moods) > 0 {
		addedQuery := " AND ( mood_id = " + strconv.Itoa(moods[0])
		for i := 1; i < len(moods); i++ {
			addedQuery = addedQuery + " OR mood_id = " + strconv.Itoa(moods[i])
		}

		WhereCondition = WhereCondition + addedQuery + " ) "
	}

	if len(genres) > 0 {
		addedQuery := " AND ( genre_id = " + strconv.Itoa(genres[0])
		for i := 1; i < len(genres); i++ {
			addedQuery = addedQuery + " OR genre_id = " + strconv.Itoa(genres[i])
		}

		WhereCondition = WhereCondition + addedQuery + " ) "
	}
	data := []RegionTrendingAudio{}
	err := mysqlDb.Model((*RegionTrendingAudio)(nil)).Column("audio_id", "updated_time").Where(WhereCondition).Limit(length).Order("updated_time DESC").ForEach(
		func(c *RegionTrendingAudio) error {
			data = append(data, *c)
			return nil
		})
	return data, err
}
