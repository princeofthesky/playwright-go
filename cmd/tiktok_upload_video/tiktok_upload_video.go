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
	page.Goto("https://www.tiktok.com/login/phone-or-email/email")

	userInputElement, _ := page.QuerySelector("input[name=username]")
	userInputElement.Fill("nguyenbatam90@gmail.com")

	passInputElement, _ := page.QuerySelector("input[type=password]")
	passInputElement.Fill("@1A2b3C4d5E")

	time.Sleep(5 * time.Second)

	nextPage, _ := page.QuerySelector("button[data-e2e=login-button]")
	err = nextPage.Click()
	if err != nil {
		println("error when nextPage click", err.Error())
	}
	time.Sleep(30 * time.Second)

	page.Goto("https://www.tiktok.com/login/phone-or-email/email")

	if err = browser.Close(); err != nil {
		log.Fatalf("could not close browser: %v", err)
	}
	if err = pw.Stop(); err != nil {
		log.Fatalf("could not stop Playwright: %v", err)
	}
}
