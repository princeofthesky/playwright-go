package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pelletier/go-toml"
	"github.com/playwright-community/playwright-go"
	"github.com/playwright-community/playwright-go/config"
	"github.com/playwright-community/playwright-go/db"
	"io/ioutil"
	"log"
	"strconv"
	"time"
)

var (
	conf     = flag.String("conf", "./tiktok_audio.toml", "config run file *.toml")
	c        = config.CrawlConfig{}
	audioDir = flag.String("audio_dir", "/tiktok_audios/", "video meme direction")
	coverDir = flag.String("cover_dir", "/tiktok_cover_audios/", "cover meme direction")
)

func main() {
	flag.Parse()
	if (*audioDir)[len(*audioDir)-1:] != "/" {
		*audioDir = *audioDir + "/"
	}
	if (*coverDir)[len(*coverDir)-1:] != "/" {
		*coverDir = *coverDir + "/"
	}
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

	playwright.Install(&playwright.RunOptions{Verbose: true, DriverDirectory: "/home/tamnb/.cache/"})
	pw, err := playwright.Run()
	browser, err := pw.Chromium.Launch()
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"
	page, err := browser.NewPage(playwright.BrowserNewContextOptions{
		UserAgent: &userAgent,
	})
	fmt.Println("NewPage")
	page.Goto("https://ads.tiktok.com/business/creativecenter/music/mobile/en")
	time.Sleep(30 * time.Second)
	count, _ := db.CountAudios()
	oldPage := count / 20
	for true {
		entries, err := page.QuerySelectorAll("div[class*=singleItem]")
		for i := 0; i < len(entries); i++ {
			imageElement, _ := entries[i].QuerySelector("div[class*=imgWrap] img")
			image, _ := imageElement.GetAttribute("src")
			titleElement, _ := entries[i].QuerySelector("div[class*=titleWrapper] span[class*=title]")
			title, _ := titleElement.TextContent()
			artistElement, _ := entries[i].QuerySelector("div[class*=artist]")
			artist, _ := artistElement.TextContent()
			durationElement, _ := entries[i].QuerySelector("div[class*=duration]")
			duration, _ := durationElement.TextContent()
			tagsElement, _ := entries[i].QuerySelectorAll("div[class*=tags] span[class*=tag]")
			tags := []string{}
			for _, tagElement := range tagsElement {
				tag, _ := tagElement.TextContent()
				tags = append(tags, tag)
			}
			audioClick, _ := entries[i].QuerySelector("div[class*=pauseIconWrap]")
			err = audioClick.Click()

			if err != nil {
				println("error when audioClick click", err.Error())
				continue
			}
			time.Sleep(5 * time.Second)
			audioElement, err := page.QuerySelector("audio[src]")
			if err != nil {
				println("error when get audioElement", err.Error(), title)
				continue
			}
			if audioElement == nil {
				println("error when get audioElement = nil", audioElement, title)
				continue
			}
			audioMp3, err := audioElement.GetAttribute("src")
			if err != nil {
				println("error when get audio Mp3", err.Error())
				continue
			}
			if len(audioMp3) == 0 {
				continue
			}
			audioInfo := db.Audio{}
			audioInfo.Title = title
			audioInfo.Artist = artist
			audioInfo.Cover = image
			hashTags, _ := json.Marshal(tags)
			audioInfo.HashTags = string(hashTags)
			audioInfo.Duration = duration
			audioInfo.TiktokUrl = audioMp3
			audioInfo.Url = audioMp3
			audioInfo.CrawledTime = time.Now().Unix()
			println(audioInfo.Title)
			audioInfo, err = db.InsertAudioInfo(audioInfo)
			if err != nil {
				println("error when insert audio ", err.Error())
				continue
			}
			err = db.InsertAudioToListNew(db.NewAudio{
				AudioId:     audioInfo.Id,
				CrawledTime: audioInfo.CrawledTime,
			})
			if err != nil {
				println("error when insert new audio ", err.Error())
			}
		}

		currentPage, _ := page.QuerySelector("li[class*=byted-pager-item][class*=byted-pager-item-checked] span")
		currentText, _ := currentPage.TextContent()
		currentNumber, _ := strconv.Atoi(currentText)
		nextPageNumber := currentNumber + 1
		if oldPage > currentNumber {
			nextPageNumber = oldPage
		}
		println(currentNumber, oldPage, nextPageNumber)
		if currentNumber == 100 {
			break
		}

		inputElement, _ := page.QuerySelector("input[class*=byted-input-size-sm]")
		err = inputElement.Fill(strconv.Itoa(nextPageNumber))
		if err != nil {
			println("error when Fill click", err.Error())
			break
		}
		nextPage, _ := page.QuerySelector("span[class=byted-pager-jump]")
		err = nextPage.Click()
		if err != nil {
			println("error when nextPage click", err.Error())
			break
		}
		time.Sleep(10 * time.Second)
	}
	if err = browser.Close(); err != nil {
		log.Fatalf("could not close browser: %v", err)
	}
	if err = pw.Stop(); err != nil {
		log.Fatalf("could not stop Playwright: %v", err)
	}
}
