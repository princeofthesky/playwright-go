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
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var (
	httpPort = flag.String("http_port", "9090", "http_port listen")
	conf     = flag.String("conf", "./tiktok_audio.toml", "config run file *.toml")
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
	r.GET("/tiktok_audios/list", GetListAudios)
	r.Run(":" + *httpPort)
}

func GetListAudios(c *gin.Context) {
	offSetText, exit := c.GetQuery("offset")
	if !exit {

	}
	lengthText, exit := c.GetQuery("length")
	if !exit {

	}
	offSet, err := strconv.Atoi(offSetText)
	if err != nil {
		// Handle error
	}
	length, err := strconv.Atoi(lengthText)
	if err != nil {
		// Handle error
	}
	if length > 20 {
		length = 20
	}
	if offSet == 0 {
		offSet = int(time.Now().Unix())
	}
	println(offSet, length)
	newVideoIds, err := db.GetListNewAudioId(offSet, length)
	if err != nil {
		println("error when get list new video ", err.Error())
	}
	listInfos := model.ListAudiosResponse{}
	for i := 0; i < len(newVideoIds); i++ {
		info, err := db.GetAudioById(newVideoIds[i].AudioId)
		if err != nil {
			continue
			println("error when get info ", err)
		}
		listInfos.Audios = append(listInfos.Audios, info)
		listInfos.NextOffset = newVideoIds[i].CrawledTime - 1
	}
	if len(newVideoIds) < length {
		listInfos.NextOffset = -1
	}
	reponseData, _ := json.Marshal(model.HttpResponse{
		Code: model.HttpSuccess,
		Msg:  "",
		Data: listInfos,
	})
	c.Data(200, "text/html; charset=UTF-8", reponseData)
}
