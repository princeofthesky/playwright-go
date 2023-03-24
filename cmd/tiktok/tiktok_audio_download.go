package main

import (
	"encoding/json"
	"errors"

	"flag"
	"fmt"
	"github.com/pelletier/go-toml"
	"github.com/playwright-community/playwright-go"
	"github.com/playwright-community/playwright-go/config"
	"github.com/playwright-community/playwright-go/db"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	conf           = flag.String("conf", "./tiktok_audio_decoder.toml", "config run file *.toml")
	updateRegions  = flag.Int("update_region", 0, " 1 if update ,0 if not - defaults")
	c              = config.CrawlConfig{}
	audioDir       = flag.String("audio_dir", "/tiktok_audios/", "video meme direction")
	coverDir       = flag.String("cover_dir", "/tiktok_cover_audios/", "cover meme direction")
	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"
	count          = 0
	maxPage        = 2
	allDataCrawl   = []*DataCrawl{}
	mu             sync.Mutex
	errClickOption error = errors.New("Error when click option mood , theme , genre")
)

type DataCrawl struct {
	Region, Theme, Mood, Genre string
}

func GetADataCrawl() *DataCrawl {
	mu.Lock()
	defer mu.Unlock()
	if count >= len(allDataCrawl) {
		return nil
	}
	count = count + 1
	return allDataCrawl[count-1]
}

func CreateNewPage(browser playwright.Browser) playwright.Page {
	page, _ := browser.NewPage(playwright.BrowserNewContextOptions{
		UserAgent: &userAgent,
	})
	fmt.Println("NewPage ")
	page.Goto("https://ads.tiktok.com/business/creativecenter/music/mobile/en")
	time.Sleep(30 * time.Second)
	return page
}
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
	if c.MaxPage > 0 {
		maxPage = c.MaxPage
	}
	err = db.Init(c.Postgres)

	playwright.Install(&playwright.RunOptions{Verbose: true, DriverDirectory: "/home/tamnb/.cache/"})
	pw, err := playwright.Run()
	browser, err := pw.Chromium.Launch()

	if *updateRegions == 1 {
		page, _ := browser.NewPage(playwright.BrowserNewContextOptions{
			UserAgent: &userAgent,
		})
		fmt.Println("NewPage")
		page.Goto("https://ads.tiktok.com/business/creativecenter/music/mobile/en")
		time.Sleep(30 * time.Second)
		UpdateRegions(page)
		//UpdateThemes(page)
		//UpdateGenres(page)
		//UpdateMoods(page)
	} else {
		regions, _ := db.GetAllRegions()
		genres, _ := db.GetAllGenres()
		moods, _ := db.GetAllMoods()
		themes, _ := db.GetAllThemes()

		for _, region := range regions {
			for _, genre := range genres {
				for _, mood := range moods {
					for _, theme := range themes {
						allDataCrawl = append(allDataCrawl, &DataCrawl{Region: region.Title, Genre: genre.Title, Theme: theme.Title, Mood: mood.Title})
					}
				}
			}
		}
		thread := c.MaxThread
		if thread < 2 {
			thread = 2
		}
		if thread > 19 {
			thread = 20
		}
		var wg sync.WaitGroup

		wg.Add(thread)

		for i := 0; i < thread; i++ {
			go func(i int) {
				page := CreateNewPage(browser)
				defer page.Close()
				defer wg.Done()
				for true {
					dataCrawl := GetADataCrawl()
					if dataCrawl == nil {
						break
					}
					err = SelectOptionlLabel(i, page, dataCrawl.Region, dataCrawl.Theme, dataCrawl.Mood, dataCrawl.Genre, browser)
					if err == errClickOption {
						page.Close()
						page = CreateNewPage(browser)
						println("SelectOptionlLabel", dataCrawl.Region, dataCrawl.Theme, dataCrawl.Mood, dataCrawl.Genre)
						SelectOptionlLabel(i, page, dataCrawl.Region, dataCrawl.Theme, dataCrawl.Mood, dataCrawl.Genre, browser)
					}
				}

			}(i)
		}
		wg.Wait()
	}

	if err = browser.Close(); err != nil {
		log.Fatalf("could not close browser: %v", err)
	}
	if err = pw.Stop(); err != nil {
		log.Fatalf("could not stop Playwright: %v", err)
	}
}

func SelectOptionlLabel(threadIndex int, page playwright.Page, region, theme, mood, genre string, browser playwright.Browser) error {
	fmt.Println("SelectOptionlLabel", threadIndex, region, theme, mood, genre)
	//select region
	regionElements, err := page.QuerySelectorAll("div[class*=byted-select-popover-panel-search] div[class*=byted-select-popover-panel-inner] div[data-option-id]  div[class*=byted-list-item-container]")
	if err != nil {
		log.Fatalf("could not find region Element : %v", err)
	}
	labelSelect, err := page.QuerySelector("div[class=topPart--mCYAM] label")
	if labelSelect == nil || err != nil {
		log.Fatalf("error when find labelSelect click : %v", err, labelSelect)
		return errClickOption
	}
	err = labelSelect.Click()
	if err != nil {
		log.Fatalf("error when labelSelect click : %v", err)
		return errClickOption
	}
	for i := 0; i < len(regionElements); i++ {
		regionText, err := regionElements[i].TextContent()
		if err != nil {
			log.Fatalf("could not find region Element : %v", err)
		}
		regionText = strings.TrimSpace(regionText)
		if strings.Compare(regionText, region) == 0 {
			err = regionElements[i].Click()
			if err != nil {
				log.Fatalf("could not find region Element : %v", err)
			}
			break
		}
	}
	// select themes
	options, err := page.QuerySelectorAll("div[class=sideBar--bzFCZ] div[class*=byted-submenu-light]")
	if err != nil {
		log.Fatalf("could not find  option element : %v", err)
	}
	for i := 0; i < len(options); i++ {
		fieldElement, err := options[i].QuerySelector("span[class=byted-menu-line-title]")
		if err != nil || fieldElement == nil {
			println("could not find  option element ", err, fieldElement)
			continue
		}
		fieldText, _ := fieldElement.TextContent()
		fieldText = strings.TrimSpace(fieldText)
		valueElements, err := options[i].QuerySelectorAll("div[class=radioSingle--U4mpE] label[class*=byted-checkbox]")
		if err != nil || valueElements == nil {
			println("could not find  valueElements :", err, valueElements)
			continue
		}
		for j := 0; j < len(valueElements); j++ {
			valueText, err := valueElements[j].TextContent()
			if err != nil {
				println("could not find  valueText : ", err)
				continue
			}
			valueText = strings.TrimSpace(valueText)
			if strings.Compare(valueText, "All") == 0 {
				continue
			}
			switch fieldText {
			case "Mood":
				if strings.Compare(valueText, mood) == 0 {
					err := valueElements[j].Click()
					if err != nil {
						println("error when MoodElement Click : ", err)
						break
					}
				}
				break
			case "Themes":
				if strings.Compare(valueText, theme) == 0 {
					err := valueElements[j].Click()
					if err != nil {
						println("error when Themes Element Click ", err)
						break
					}
				}
				break
			case "Genre":
				if strings.Compare(valueText, genre) == 0 {
					err := valueElements[j].Click()
					if err != nil {
						println("error when Genre Element Click : ", err)
						break
					}
				}
				break
			}
		}
	}
	return ParseAudioInfo(region, theme, mood, genre, page)
}

func ParseAudioInfo(region, theme, mood, genre string, page playwright.Page) error {
	regionElement, err := page.QuerySelector("div[class=topPart--mCYAM] input[class*=byted-input-size-md]")
	if err != nil {
		println("could not find region Element : %v", err)
	}
	if regionElement == nil {
		println("could not find region Element ==nil ")
	}
	regionText, err := regionElement.GetAttribute("value")
	if err != nil {
		println("could not parse region Text : %v", err)
	}
	themeText := ""
	moodText := ""
	genreText := ""
	options, err := page.QuerySelectorAll("div[class=sideBar--bzFCZ] div[class*=byted-submenu-light]")
	for i := 0; i < len(options); i++ {
		fieldElement, err := options[i].QuerySelector("span[class=byted-menu-line-title]")
		if err != nil {
			println("could not find  option element : %v", err)
			continue
		}
		fieldText, _ := fieldElement.TextContent()
		valueElements, err := options[i].QuerySelectorAll("div[class=radioSingle--U4mpE] label[class*=byted-checkbox]")
		if err != nil {
			println("could not find  valueElements : %v", err)
			continue
		}
		for j := 0; j < len(valueElements); j++ {
			checkAttribute, err := valueElements[j].QuerySelectorAll("span[class*=byted-checkbox-icon-checked]")
			if err != nil {
				println("could not find  checkAttribute : %v", err)
				continue
			}
			valueText, err := valueElements[j].TextContent()
			if err != nil {
				println("could not find  valueText : %v", err)
				continue
			}
			if len(checkAttribute) > 0 {
				switch fieldText {
				case "Mood":
					moodText = valueText
					break
				case "Themes":
					themeText = valueText
					break
				case "Genre":
					genreText = valueText
					break
				}
			}
		}
	}
	if strings.Compare(regionText, region) != 0 || strings.Compare(themeText, theme) != 0 ||
		strings.Compare(moodText, mood) != 0 || strings.Compare(genreText, genre) != 0 {
		fmt.Println("error when check option again", regionText, themeText, moodText, genreText)
		return errClickOption
	}
	regionInfo, _ := db.GetRegionByTitle(regionText)
	themeInfo, _ := db.GetThemeByTitle(themeText)
	moodInfo, _ := db.GetMoodByTitle(moodText)
	genreInfo, _ := db.GetGenreByTitle(genreText)

	fmt.Println("ParseAudioInfo", regionText, themeText, moodText, genreText)
	//count, _ := db.CountAudios()
	//oldPage := 0 / 20

	maxPagElement, err := page.QuerySelector("div[class=wrap--gNm34] span[class=byted-pager-record]")
	if err != nil || maxPagElement == nil {
		println("error when get maxPage page new audio ", maxPagElement)
		println("error when get maxPage page new audio ", err)
		return err
	}
	totalAudioText, _ := maxPagElement.TextContent()
	totalText := strings.TrimSpace(totalAudioText)
	reg := regexp.MustCompile(`[^0-9]`)
	totalText = reg.ReplaceAllString(totalText, "")
	totalAudio, _ := strconv.Atoi(totalText)
	totalPage := totalAudio / 20
	if totalAudio%20 != 0 {
		totalPage = totalPage + 1
	}
	if totalPage > maxPage {
		totalPage = maxPage
	}
	for currentPage := totalPage; currentPage > 0; currentPage-- {
		if totalPage>1 {
			inputElement, err := page.QuerySelector("input[class*=byted-input-size-sm]")
			if err != nil {
				println("error find  inputElement nextPage ", err.Error())
				break
			}
			if inputElement == nil {
				println("error find  inputElement nextPage Fill = nil")
				break
			}
			err = inputElement.Fill(strconv.Itoa(currentPage))
			if err != nil {
				println("error when Fill click", err.Error())
				break
			}
			nextPage, err := page.QuerySelector("span[class=byted-pager-jump]")
			if err != nil || nextPage == nil {
				println("error when nextPage click", err, nextPage)
				break
			}
			err = nextPage.Click()
			if err != nil {
				println("error when nextPage click", err.Error())
				break
			}
		}
		entries, err := page.QuerySelectorAll("div[class*=singleItem]")
		if len(entries) == 0 {
			fmt.Println("Not found list audios ")
			continue
		}
		println(currentPage, totalPage, len(entries), totalAudio, totalAudioText)
		for i := len(entries) - 1; i > 0; i-- {
			tiktokIdElement, err := entries[i].QuerySelector("div[class*=titleWrapper] a[class*=tool][href]")
			if tiktokIdElement==nil {

			}
			tiktokAudioSrc, _ := tiktokIdElement.GetAttribute("href")
			tiktokAudioText := strings.Split(tiktokAudioSrc, "&id=")
			tiktokAudioId := tiktokAudioText[len(tiktokAudioText)-1]
			tiktokAudioText = strings.Split(tiktokAudioId, "&")
			tiktokAudioId = tiktokAudioText[0]
			fmt.Println(tiktokAudioId)
			audioInfo, _ := db.GetAudioByTikTokId(tiktokAudioId)
			if audioInfo.Id == 0 {
				imageElement, _ := entries[i].QuerySelector("div[class*=imgWrap] img")
				image, _ := imageElement.GetAttribute("src")
				titleElement, _ := entries[i].QuerySelector("div[class*=titleWrapper] span[class*=title]")
				title, _ := titleElement.TextContent()
				artistElement, _ := entries[i].QuerySelector("div[class*=artist]")
				artist, _ := artistElement.TextContent()
				durationElement, _ := entries[i].QuerySelector("div[class*=duration]")
				durationText, _ := durationElement.TextContent()
				durations := strings.Split(durationText, ":")
				minuteTime, _ := strconv.Atoi(durations[0])
				secondTime, _ := strconv.Atoi(durations[1])
				tagsElement, _ := entries[i].QuerySelectorAll("div[class*=tags] span[class*=tag]")
				tags := []string{}
				for _, tagElement := range tagsElement {
					tag, _ := tagElement.TextContent()
					tags = append(tags, tag)
				}
				audioClick, err := entries[i].QuerySelector("div[class*=pauseIconWrap]")
				if err != nil || audioClick == nil {
					continue
				}
				err = audioClick.Click()
				if err != nil {
					continue
				}
				audioElement, err := page.QuerySelector("audio[src]")
				if err != nil || audioElement == nil {
					continue
				}
				audioMp3, err := audioElement.GetAttribute("src")
				if err != nil {
					continue
				}
				audioMp3 = strings.TrimSpace(audioMp3)
				if len(audioMp3) == 0 {
					continue
				}
				audioInfo = db.Audio{}
				audioInfo.Title = title
				audioInfo.Artist = artist
				audioInfo.Cover = image
				hashTags, _ := json.Marshal(tags)
				audioInfo.HashTags = string(hashTags)
				audioInfo.Duration = minuteTime*60 + secondTime
				audioInfo.TiktokUrl = audioMp3
				audioInfo.TiktokId = tiktokAudioId
				audioInfo.Url = audioMp3
				audioInfo.CrawledTime = time.Now().Unix()
				audioInfo, err = db.InsertAudioInfo(audioInfo)
				if err != nil {
					//println("error when insert audio ", err.Error())
					continue
				}
			}
			err = db.InsertAudioToListNew(regionInfo.Id, db.RegionTrendingAudio{
				AudioId:     audioInfo.Id,
				UpdatedTime: audioInfo.CrawledTime,
				ThemeId:     themeInfo.Id,
				Duration:    audioInfo.Duration,
				GenreId:     genreInfo.Id,
				MoodId:      moodInfo.Id,
			})
			if err != nil {
				//println("error when insert new audio ", err.Error())
			}
		}
	}
	return nil
}

func UpdateRegions(page playwright.Page) {
	regionElements, err := page.QuerySelectorAll("div[class*=byted-select-popover-panel-search] div[class*=byted-select-popover-panel-inner] div[class*=byted-list-item-inner-wrapper]")
	if err != nil {
		log.Fatalf("could not find region Element : %v", err)
	}
	for i := 0; i < len(regionElements); i++ {
		regionText, _ := regionElements[i].TextContent()
		regionText = strings.TrimSpace(regionText)
		db.InsertRegionInfo(db.Region{Title: regionText})
	}
}

func UpdateThemes(page playwright.Page) {
	options, err := page.QuerySelectorAll("div[class=sideBar--bzFCZ] div[class*=byted-submenu-light]")
	if err != nil {
		log.Fatalf("could not find  option element : %v", err)
	}
	for i := 0; i < len(options); i++ {
		fieldElement, err := options[i].QuerySelector("span[class=byted-menu-line-title]")
		if err != nil {
			log.Fatalf("could not find  option element : %v", err)
		}
		fieldText, _ := fieldElement.TextContent()
		fieldText = strings.TrimSpace(fieldText)
		if strings.Compare(fieldText, "Themes") != 0 {
			continue
		}
		valueElements, err := options[i].QuerySelectorAll("div[class=radioSingle--U4mpE] label[class*=byted-checkbox] span[class*=byted-checkbox-label]")
		if err != nil {
			log.Fatalf("could not find  valueElements : %v", err)
		}
		for j := 0; j < len(valueElements); j++ {
			themeText, err := valueElements[j].TextContent()
			if err != nil {
				log.Fatalf("could not find  themeText : %v", err)
			}
			themeText = strings.TrimSpace(themeText)
			if strings.Compare(themeText, "All") == 0 {
				continue
			}
			db.InsertThemeInfo(db.Theme{Title: themeText})
		}
	}
}

func UpdateGenres(page playwright.Page) {
	options, err := page.QuerySelectorAll("div[class=sideBar--bzFCZ] div[class*=byted-submenu-light]")
	if err != nil {
		log.Fatalf("could not find  option element : %v", err)
	}
	for i := 0; i < len(options); i++ {
		fieldElement, err := options[i].QuerySelector("span[class=byted-menu-line-title]")
		if err != nil {
			log.Fatalf("could not find  option element : %v", err)
		}
		fieldText, _ := fieldElement.TextContent()
		fieldText = strings.TrimSpace(fieldText)
		if strings.Compare(fieldText, "Genre") != 0 {
			continue
		}
		valueElements, err := options[i].QuerySelectorAll("div[class=radioSingle--U4mpE] label[class*=byted-checkbox] span[class*=byted-checkbox-label]")
		if err != nil {
			log.Fatalf("could not find  valueElements : %v", err)
		}
		for j := 0; j < len(valueElements); j++ {
			genreText, err := valueElements[j].TextContent()
			if err != nil {
				log.Fatalf("could not find  genreText : %v", err)
			}
			genreText = strings.TrimSpace(genreText)
			if strings.Compare(genreText, "All") == 0 {
				continue
			}
			db.InsertGenreInfo(db.Genre{Title: genreText})
		}
	}
}

func UpdateMoods(page playwright.Page) {
	options, err := page.QuerySelectorAll("div[class=sideBar--bzFCZ] div[class*=byted-submenu-light]")
	if err != nil {
		log.Fatalf("could not find  option element : %v", err)
	}
	for i := 0; i < len(options); i++ {
		fieldElement, err := options[i].QuerySelector("span[class=byted-menu-line-title]")
		if err != nil {
			log.Fatalf("could not find  option element : %v", err)
		}
		fieldText, _ := fieldElement.TextContent()
		fieldText = strings.TrimSpace(fieldText)
		if strings.Compare(fieldText, "Mood") != 0 {
			continue
		}
		valueElements, err := options[i].QuerySelectorAll("div[class=radioSingle--U4mpE] label[class*=byted-checkbox] span[class*=byted-checkbox-label]")
		if err != nil {
			log.Fatalf("could not find  valueElements : %v", err)
		}
		for j := 0; j < len(valueElements); j++ {
			moodText, err := valueElements[j].TextContent()
			if err != nil {
				log.Fatalf("could not find  moodText : %v", err)
			}
			moodText = strings.TrimSpace(moodText)
			if strings.Compare(moodText, "All") == 0 {
				continue
			}
			db.InsertMoodInfo(db.Mood{Title: moodText})
		}
	}
}
