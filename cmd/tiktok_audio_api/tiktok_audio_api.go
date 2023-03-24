package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml"
	"github.com/playwright-community/playwright-go/config"
	"github.com/playwright-community/playwright-go/db"
	"github.com/playwright-community/playwright-go/model"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var (
	httpPort = flag.String("http_port", "9090", "http_port listen")
	conf     = flag.String("conf", "./tiktok_audio_decoder.toml", "config run file *.toml")
	c        = config.CrawlConfig{}
)

func main() {
	flag.Parse()
	configBytes, err := ioutil.ReadFile(*conf)
	if err != nil {
		fmt.Println("err when read config file ", err, "file ", *conf)
	}
	err = toml.Unmarshal(configBytes, &c)
	if err != nil {
		fmt.Println("err when pass toml file ", err)
	}
	text, err := json.Marshal(c)
	fmt.Println("Success read config from toml file ", string(text))
	err = db.Init(c.Postgres)
	if err != nil {
		fmt.Println("err", err)
	}

	defer db.Close()
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/tiktok_audios/list", GetListAudios)
	r.GET("/tiktok_audios/themes", GetAllThemes)
	r.GET("/tiktok_audios/moods", GetAllMood)
	r.GET("/tiktok_audios/genres", GetAllGenres)
	r.GET("/tiktok_audios/regions", GetAllRegions)
	r.Run(":" + *httpPort)
}

func GetAllThemes(c *gin.Context) {
	data, _ := db.GetAllThemes()
	responseData, _ := json.Marshal(model.HttpResponse{
		Code: model.HttpSuccess,
		Msg:  "",
		Data: data,
	})
	c.Data(200, "text/html; charset=UTF-8", responseData)
}

func GetAllMood(c *gin.Context) {
	data, _ := db.GetAllMoods()
	responseData, _ := json.Marshal(model.HttpResponse{
		Code: model.HttpSuccess,
		Msg:  "",
		Data: data,
	})
	c.Data(200, "text/html; charset=UTF-8", responseData)
}

func GetAllRegions(c *gin.Context) {
	data, _ := db.GetAllRegions()
	responseData, _ := json.Marshal(model.HttpResponse{
		Code: model.HttpSuccess,
		Msg:  "",
		Data: data,
	})
	c.Data(200, "text/html; charset=UTF-8", responseData)
}
func GetAllGenres(c *gin.Context) {
	data, _ := db.GetAllGenres()
	responseData, _ := json.Marshal(model.HttpResponse{
		Code: model.HttpSuccess,
		Msg:  "",
		Data: data,
	})
	c.Data(200, "text/html; charset=UTF-8", responseData)
}
func GetIntQuery(c *gin.Context, query string) int {
	valueText, exit := c.GetQuery(query)
	if !exit {
		return 0
	}
	value, err := strconv.Atoi(valueText)
	if err != nil {
		return 0
	}
	return value
}

func GetListAudios(c *gin.Context) {
	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		responseData, _ := json.Marshal(model.HttpResponse{
			Code: model.HttpFail,
			Msg:  "Error when read body data client Post",
			Data: nil,
		})
		c.Data(200, "text/html; charset=UTF-8", responseData)
		return
	}
	audioRequirement := model.AudioRequirement{}
	err = json.Unmarshal(jsonData, &audioRequirement)
	if err != nil {
		responseData, _ := json.Marshal(model.HttpResponse{
			Code: model.HttpFail,
			Msg:  "Error when parse json body data client Post",
			Data: nil,
		})
		c.Data(200, "text/html; charset=UTF-8", responseData)
		return
	}
	if audioRequirement.Region == 0 {
		responseData, _ := json.Marshal(model.HttpResponse{
			Code: model.HttpFail,
			Msg:  "Region not found , not allow 0",
			Data: nil,
		})
		c.Data(200, "text/html; charset=UTF-8", responseData)
		return
	}

	if audioRequirement.Length > 20 {
		audioRequirement.Length = 20
	}
	if audioRequirement.Length < 1 {
		audioRequirement.Length = 1
	}
	offset, err := strconv.Atoi(audioRequirement.Offset)
	if err != nil {
		responseData, _ := json.Marshal(model.HttpResponse{
			Code: model.HttpFail,
			Msg:  "Error when parse offset",
			Data: nil,
		})
		c.Data(200, "text/html; charset=UTF-8", responseData)
		return
	}
	if offset < 1 {
		offset = int(time.Now().Unix())
	}
	if audioRequirement.MinDuration < 0 {
		audioRequirement.MinDuration = 0
	}
	if audioRequirement.MaxDuration < 1 {
		audioRequirement.MaxDuration = 10000
	}
	println(string(jsonData))
	trendingAudios, err := db.GetListNewAudioId(audioRequirement.Themes, audioRequirement.Moods, audioRequirement.Genres,
		audioRequirement.Region, audioRequirement.MinDuration, audioRequirement.MaxDuration,
		offset, audioRequirement.Length)
	if err != nil {
		println("error when get list new video ", err.Error())
	}
	listInfos := model.ListAudiosResponse{}
	for i := 0; i < len(trendingAudios); i++ {
		info, err := db.GetAudioById(trendingAudios[i].AudioId)
		if err != nil {
			continue
			println("error when get info ", err)
		}
		listInfos.Audios = append(listInfos.Audios, info)
	}
	if len(trendingAudios) < audioRequirement.Length {
		listInfos.NextOffset = "-1"
	} else {
		listInfos.NextOffset = strconv.FormatInt(trendingAudios[len(trendingAudios)-1].UpdatedTime-1, 10)
	}
	responseData, _ := json.Marshal(model.HttpResponse{
		Code: model.HttpSuccess,
		Msg:  "",
		Data: listInfos,
	})
	c.Data(200, "text/html; charset=UTF-8", responseData)
}
