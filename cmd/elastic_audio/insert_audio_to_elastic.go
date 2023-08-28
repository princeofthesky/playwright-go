package main

import (
	"flag"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/pelletier/go-toml"
	"github.com/playwright-community/playwright-go/config"
	"github.com/playwright-community/playwright-go/db"
	"io"
	"log"
	"os"
	"strconv"
	// Import the Elasticsearch library packages
	"github.com/elastic/go-elasticsearch/v8"
)

var (
	conf = flag.String("conf", "./tiktok_audio.toml", "config run file *.toml")
	c    = config.CrawlConfig{}
)

func main() {
	flag.Parse()
	configBytes, err := os.ReadFile(*conf)
	if err != nil {
		fmt.Println("err when read config file ", err, "file ", *conf)
	}
	err = toml.Unmarshal(configBytes, &c)
	// Declare an Elasticsearch configuration
	cfg := elasticsearch.Config{
		Addresses: []string{
			c.Elasticsearch.Addr,
		},
		Username: c.Elasticsearch.User,
		Password: c.Elasticsearch.Password,
	}
	// Instantiate a new Elasticsearch client object instance
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		println(err.Error())
	}
	// Have the client instance return a response
	res, err := client.Info()
	if err != nil {
		println("error when connect with elasticsearch", err.Error())
	}
	stat,_:=io.ReadAll(res.Body)
	println(string(stat))
	db.Init(c.Postgres)
	audios, err := db.GetAllAudio()
	if err != nil {
		log.Fatal("error when get all audios", err.Error())
	}
	for _, audio := range audios {
		audioInfo:=map[string]string{}
		audioInfo["title"]=audio.Title
		audioInfo["artist"]=audio.Artist
		audioInfo["id"]=strconv.Itoa(audio.Id)
		_, err := client.Index("tiktok_audios", esutil.NewJSONReader(&audioInfo))
		if err != nil {
			println("error when insert audios", err.Error())
		}
	}
}
