package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pelletier/go-toml"
	"github.com/playwright-community/playwright-go/config"
	"github.com/playwright-community/playwright-go/db"
	"io/ioutil"
	"time"
)
var (
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
	audioIds,err:=db.GetListNewAudioId([]int{},[]int{},[]int{},1,0,200,int(time.Now().Unix()),20)
	fmt.Println(audioIds)
	fmt.Println(err)
}
