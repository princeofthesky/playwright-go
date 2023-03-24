package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/playwright-community/playwright-go/model"
	"github.com/playwright-community/playwright-go/tiktok_audio_decoder"
	"io"
	"net/http"

	"flag"
	"fmt"
	"github.com/pelletier/go-toml"
	"github.com/playwright-community/playwright-go"
	"github.com/playwright-community/playwright-go/config"
	"github.com/playwright-community/playwright-go/db"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"
)

var (
	conf           = flag.String("conf", "./tiktok_audio.toml", "config run file *.toml")
	updateRegions  = flag.Int("update_region", 1, " 1 if update ,0 if not - defaults")
	c              = config.CrawlConfig{}
	audioDir       = flag.String("audio_dir", "/tiktok_audios/", "video meme direction")
	coverDir       = flag.String("cover_dir", "/tiktok_cover_audios/", "cover meme direction")
	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"
	count          = 0
	maxPage        = 2
	allDataCrawl   = []*DataCrawl{}
	mu             sync.Mutex
	errClickOption error = errors.New("Error when click option mood , theme , genre")
	client               = http.DefaultClient
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

	if *updateRegions == 1 {
		UpdateRegions()
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
						allDataCrawl = append(allDataCrawl, &DataCrawl{Region: region.Code, Genre: genre.Title, Theme: theme.Title, Mood: mood.Title})
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
				defer wg.Done()
				for true {
					dataCrawl := GetADataCrawl()
					if dataCrawl == nil {
						break
					}
					err = ParseAudioInfo(i, dataCrawl)
				}
			}(i)
		}
		wg.Wait()
	}
}
func GetAudioDataFromTiktok(audioRequest model.TiktokPostAudioRequest) (model.TiktokRequestResponse, error) {
	body, _ := json.Marshal(audioRequest)
	req, _ := http.NewRequest("POST", "https://ads.tiktok.com/creative_radar_api/v1/audio_lib/music/list", bytes.NewReader([]byte(body)))
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36")
	req.Header.Set("content-type", "application/json")
	res, err := client.Do(req)
	time.Sleep(100 * time.Millisecond)
	var dataResponse model.TiktokRequestResponse
	if err != nil {
		return dataResponse, err
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return dataResponse, err
	}
	err = json.Unmarshal(data, &dataResponse)
	if err != nil {
		return dataResponse, err
	}
	if dataResponse.Code != 0 {
		return dataResponse, errors.New("Error when get audio info from tik tok " + dataResponse.Msg)
	}
	return dataResponse, err
}

func ParseAudioInfo(threadId int, dataCrawl *DataCrawl) error {
	audioRequest := model.TiktokPostAudioRequest{
		Page:       1,
		Limit:      20,
		Region:     dataCrawl.Region,
		Genres:     []string{dataCrawl.Genre},
		Moods:      []string{dataCrawl.Mood},
		Themes:     []string{dataCrawl.Theme},
		Singer:     "",
		MusicName:  "",
		Scenarios:  0,
		Placements: []string{},
	}
	fmt.Println(threadId, dataCrawl.Region, dataCrawl.Theme, dataCrawl.Mood, dataCrawl.Genre)
	data, err := GetAudioDataFromTiktok(audioRequest)
	if err != nil {
		fmt.Println("error when GetAudioDataFromTiktok ", err, data.Msg)
		return err
	}
	totalPage := data.Data.Pagination.TotalCount / 20
	if totalPage > maxPage {
		totalPage = maxPage
	}
	region, _ := db.GetRegionByTitle(dataCrawl.Region)
	theme, _ := db.GetThemeByTitle(dataCrawl.Theme)
	mood, _ := db.GetMoodByTitle(dataCrawl.Mood)
	genre, _ := db.GetGenreByTitle(dataCrawl.Genre)
	for i := totalPage; i > 0; i-- {
		audioRequest.Page = i
		data, err = GetAudioDataFromTiktok(audioRequest)
		if err != nil {
			fmt.Println("error when GetAudioDataFromTiktok ", err, data.Msg)
			continue
		}
		for j := len(data.Data.List) - 1; j >= 0; j-- {
			tiktokAudioResponse := data.Data.List[j]
			tiktokAudioId := tiktokAudioResponse.MusicId
			audioInfo, _ := db.GetAudioByTikTokId(tiktokAudioId)
			if audioInfo.Id == 0 {
				audioInfo = db.Audio{}
				audioInfo.Title = tiktokAudioResponse.Title
				audioInfo.Artist = tiktokAudioResponse.Singer
				audioInfo.Cover = tiktokAudioResponse.PosterUrl
				hashTags, _ := json.Marshal(tiktokAudioResponse.PlacementAllowed)
				audioInfo.HashTags = string(hashTags)
				audioInfo.Duration = tiktokAudioResponse.Duration
				audioInfo.TiktokUrl = tiktok_audio_decoder.GetAudioLinkFromDetail(tiktokAudioResponse.Detail)
				audioInfo.TiktokId = tiktokAudioId
				audioInfo.Url = audioInfo.TiktokUrl
				audioInfo.CrawledTime = time.Now().Unix()
				audioInfo, err = db.InsertAudioInfo(audioInfo)
				if err != nil {
					println("error when insert audio ", err.Error())
					continue
				}
			}
			if region.Id > 0 {
				err = db.InsertAudioToListNew(region.Id, db.RegionTrendingAudio{
					AudioId:     audioInfo.Id,
					UpdatedTime: audioInfo.CrawledTime,
					ThemeId:     theme.Id,
					Duration:    audioInfo.Duration,
					GenreId:     genre.Id,
					MoodId:      mood.Id,
				})
				if err != nil {
					println("error when insert to list region audio ", err.Error())
				}
			}

		}
	}
	return nil
}

func UpdateRegions() {

	regionsList := "[{\"value\":\"AD\",\"label\":\"Andorra\"},{\"value\":\"AE\",\"label\":\"United Arab Emirates\"},{\"value\":\"AF\",\"label\":\"Afghanistan\"},{\"value\":\"AG\",\"label\":\"Antigua and Barbuda\"},{\"value\":\"AI\",\"label\":\"Anguilla\"},{\"value\":\"AL\",\"label\":\"Albania\"},{\"value\":\"AM\",\"label\":\"Armenia\"},{\"value\":\"AO\",\"label\":\"Angola\"},{\"value\":\"AR\",\"label\":\"Argentina\"},{\"value\":\"AS\",\"label\":\"American Samoa\"},{\"value\":\"AT\",\"label\":\"Austria\"},{\"value\":\"AU\",\"label\":\"Australia\"},{\"value\":\"AW\",\"label\":\"Aruba\"},{\"value\":\"AX\",\"label\":\"Åland\"},{\"value\":\"AZ\",\"label\":\"Azerbaijan\"},{\"value\":\"BA\",\"label\":\"Bosnia and Herzegovina\"},{\"value\":\"BB\",\"label\":\"Barbados\"},{\"value\":\"BD\",\"label\":\"Bangladesh\"},{\"value\":\"BE\",\"label\":\"Belgium\"},{\"value\":\"BF\",\"label\":\"Burkina Faso\"},{\"value\":\"BG\",\"label\":\"Bulgaria\"},{\"value\":\"BH\",\"label\":\"Bahrain\"},{\"value\":\"BI\",\"label\":\"Burundi\"},{\"value\":\"BJ\",\"label\":\"Benin\"},{\"value\":\"BL\",\"label\":\"Saint-Barthélemy\"},{\"value\":\"BM\",\"label\":\"Bermuda\"},{\"value\":\"BN\",\"label\":\"Brunei\"},{\"value\":\"BO\",\"label\":\"Bolivia\"},{\"value\":\"BQ\",\"label\":\"Bonaire, Sint Eustatius, and Saba\"},{\"value\":\"BR\",\"label\":\"Brazil\"},{\"value\":\"BS\",\"label\":\"Bahamas\"},{\"value\":\"BT\",\"label\":\"Bhutan\"},{\"value\":\"BV\",\"label\":\"Bouvet Island\"},{\"value\":\"BW\",\"label\":\"Botswana\"},{\"value\":\"BY\",\"label\":\"Belarus\"},{\"value\":\"BZ\",\"label\":\"Belize\"},{\"value\":\"CA\",\"label\":\"Canada\"},{\"value\":\"CC\",\"label\":\"Cocos [Keeling] Islands\"},{\"value\":\"CD\",\"label\":\"Congo\"},{\"value\":\"CF\",\"label\":\"Central African Republic\"},{\"value\":\"CG\",\"label\":\"Republic of the Congo\"},{\"value\":\"CH\",\"label\":\"Switzerland\"},{\"value\":\"CI\",\"label\":\"Ivory Coast\"},{\"value\":\"CK\",\"label\":\"Cook Islands\"},{\"value\":\"CL\",\"label\":\"Chile\"},{\"value\":\"CM\",\"label\":\"Cameroon\"},{\"value\":\"CN\",\"label\":\"China\"},{\"value\":\"CO\",\"label\":\"Colombia\"},{\"value\":\"CR\",\"label\":\"Costa Rica\"},{\"value\":\"CU\",\"label\":\"Cuba\"},{\"value\":\"CV\",\"label\":\"Cabo Verde\"},{\"value\":\"CW\",\"label\":\"Curaçao\"},{\"value\":\"CX\",\"label\":\"Christmas Island\"},{\"value\":\"CY\",\"label\":\"Cyprus\"},{\"value\":\"CZ\",\"label\":\"Czechia\"},{\"value\":\"DE\",\"label\":\"Germany\"},{\"value\":\"DJ\",\"label\":\"Djibouti\"},{\"value\":\"DK\",\"label\":\"Denmark\"},{\"value\":\"DM\",\"label\":\"Dominica\"},{\"value\":\"DO\",\"label\":\"Dominican Republic\"},{\"value\":\"DZ\",\"label\":\"Algeria\"},{\"value\":\"EC\",\"label\":\"Ecuador\"},{\"value\":\"EE\",\"label\":\"Estonia\"},{\"value\":\"EG\",\"label\":\"Egypt\"},{\"value\":\"EH\",\"label\":\"Western Sahara\"},{\"value\":\"ER\",\"label\":\"Eritrea\"},{\"value\":\"ES\",\"label\":\"Spain\"},{\"value\":\"ET\",\"label\":\"Ethiopia\"},{\"value\":\"FI\",\"label\":\"Finland\"},{\"value\":\"FJ\",\"label\":\"Fiji\"},{\"value\":\"FK\",\"label\":\"Falkland Islands\"},{\"value\":\"FM\",\"label\":\"Federated States of Micronesia\"},{\"value\":\"FO\",\"label\":\"Faroe Islands\"},{\"value\":\"FR\",\"label\":\"France\"},{\"value\":\"GA\",\"label\":\"Gabon\"},{\"value\":\"GB\",\"label\":\"United Kingdom\"},{\"value\":\"GD\",\"label\":\"Grenada\"},{\"value\":\"GE\",\"label\":\"Georgia\"},{\"value\":\"GF\",\"label\":\"French Guiana\"},{\"value\":\"GG\",\"label\":\"Guernsey\"},{\"value\":\"GH\",\"label\":\"Ghana\"},{\"value\":\"GI\",\"label\":\"Gibraltar\"},{\"value\":\"GL\",\"label\":\"Greenland\"},{\"value\":\"GM\",\"label\":\"Gambia\"},{\"value\":\"GN\",\"label\":\"Guinea\"},{\"value\":\"GP\",\"label\":\"Guadeloupe\"},{\"value\":\"GQ\",\"label\":\"Equatorial Guinea\"},{\"value\":\"GR\",\"label\":\"Greece\"},{\"value\":\"GS\",\"label\":\"South Georgia and the South Sandwich Islands\"},{\"value\":\"GT\",\"label\":\"Guatemala\"},{\"value\":\"GU\",\"label\":\"Guam\"},{\"value\":\"GW\",\"label\":\"Guinea-Bissau\"},{\"value\":\"GY\",\"label\":\"Guyana\"},{\"value\":\"HK\",\"label\":\"Hong Kong\"},{\"value\":\"HM\",\"label\":\"Heard Island and McDonald Islands\"},{\"value\":\"HN\",\"label\":\"Honduras\"},{\"value\":\"HR\",\"label\":\"Croatia\"},{\"value\":\"HT\",\"label\":\"Haiti\"},{\"value\":\"HU\",\"label\":\"Hungary\"},{\"value\":\"ID\",\"label\":\"Indonesia\"},{\"value\":\"IE\",\"label\":\"Ireland\"},{\"value\":\"IL\",\"label\":\"Israel\"},{\"value\":\"IM\",\"label\":\"Isle of Man\"},{\"value\":\"IN\",\"label\":\"India\"},{\"value\":\"IO\",\"label\":\"British Indian Ocean Territory\"},{\"value\":\"IQ\",\"label\":\"Iraq\"},{\"value\":\"IR\",\"label\":\"Iran\"},{\"value\":\"IS\",\"label\":\"Iceland\"},{\"value\":\"IT\",\"label\":\"Italy\"},{\"value\":\"JE\",\"label\":\"Jersey\"},{\"value\":\"JM\",\"label\":\"Jamaica\"},{\"value\":\"JO\",\"label\":\"Hashemite Kingdom of Jordan\"},{\"value\":\"JP\",\"label\":\"Japan\"},{\"value\":\"KE\",\"label\":\"Kenya\"},{\"value\":\"KG\",\"label\":\"Kyrgyzstan\"},{\"value\":\"KH\",\"label\":\"Cambodia\"},{\"value\":\"KI\",\"label\":\"Kiribati\"},{\"value\":\"KM\",\"label\":\"Comoros\"},{\"value\":\"KN\",\"label\":\"St Kitts and Nevis\"},{\"value\":\"KP\",\"label\":\"North Korea\"},{\"value\":\"KR\",\"label\":\"Republic of Korea\"},{\"value\":\"KW\",\"label\":\"Kuwait\"},{\"value\":\"KY\",\"label\":\"Cayman Islands\"},{\"value\":\"KZ\",\"label\":\"Kazakhstan\"},{\"value\":\"LA\",\"label\":\"Laos\"},{\"value\":\"LB\",\"label\":\"Lebanon\"},{\"value\":\"LC\",\"label\":\"Saint Lucia\"},{\"value\":\"LI\",\"label\":\"Liechtenstein\"},{\"value\":\"LK\",\"label\":\"Sri Lanka\"},{\"value\":\"LR\",\"label\":\"Liberia\"},{\"value\":\"LS\",\"label\":\"Lesotho\"},{\"value\":\"LT\",\"label\":\"Republic of Lithuania\"},{\"value\":\"LU\",\"label\":\"Luxembourg\"},{\"value\":\"LV\",\"label\":\"Latvia\"},{\"value\":\"LY\",\"label\":\"Libya\"},{\"value\":\"MA\",\"label\":\"Morocco\"},{\"value\":\"MC\",\"label\":\"Monaco\"},{\"value\":\"MD\",\"label\":\"Republic of Moldova\"},{\"value\":\"ME\",\"label\":\"Montenegro\"},{\"value\":\"MF\",\"label\":\"Saint Martin\"},{\"value\":\"MG\",\"label\":\"Madagascar\"},{\"value\":\"MH\",\"label\":\"Marshall Islands\"},{\"value\":\"MK\",\"label\":\"Macedonia\"},{\"value\":\"ML\",\"label\":\"Mali\"},{\"value\":\"MM\",\"label\":\"Myanmar [Burma]\"},{\"value\":\"MN\",\"label\":\"Mongolia\"},{\"value\":\"MO\",\"label\":\"Macao\"},{\"value\":\"MP\",\"label\":\"Northern Mariana Islands\"},{\"value\":\"MQ\",\"label\":\"Martinique\"},{\"value\":\"MR\",\"label\":\"Mauritania\"},{\"value\":\"MS\",\"label\":\"Montserrat\"},{\"value\":\"MT\",\"label\":\"Malta\"},{\"value\":\"MU\",\"label\":\"Mauritius\"},{\"value\":\"MV\",\"label\":\"Maldives\"},{\"value\":\"MW\",\"label\":\"Malawi\"},{\"value\":\"MX\",\"label\":\"Mexico\"},{\"value\":\"MY\",\"label\":\"Malaysia\"},{\"value\":\"MZ\",\"label\":\"Mozambique\"},{\"value\":\"NA\",\"label\":\"Namibia\"},{\"value\":\"NC\",\"label\":\"New Caledonia\"},{\"value\":\"NE\",\"label\":\"Niger\"},{\"value\":\"NF\",\"label\":\"Norfolk Island\"},{\"value\":\"NG\",\"label\":\"Nigeria\"},{\"value\":\"NI\",\"label\":\"Nicaragua\"},{\"value\":\"NL\",\"label\":\"Netherlands\"},{\"value\":\"NO\",\"label\":\"Norway\"},{\"value\":\"NP\",\"label\":\"Nepal\"},{\"value\":\"NR\",\"label\":\"Nauru\"},{\"value\":\"NU\",\"label\":\"Niue\"},{\"value\":\"NZ\",\"label\":\"New Zealand\"},{\"value\":\"OM\",\"label\":\"Oman\"},{\"value\":\"PA\",\"label\":\"Panama\"},{\"value\":\"PE\",\"label\":\"Peru\"},{\"value\":\"PF\",\"label\":\"French Polynesia\"},{\"value\":\"PG\",\"label\":\"Papua New Guinea\"},{\"value\":\"PH\",\"label\":\"Philippines\"},{\"value\":\"PK\",\"label\":\"Pakistan\"},{\"value\":\"PL\",\"label\":\"Poland\"},{\"value\":\"PM\",\"label\":\"Saint Pierre and Miquelon\"},{\"value\":\"PN\",\"label\":\"Pitcairn Islands\"},{\"value\":\"PR\",\"label\":\"Puerto Rico\"},{\"value\":\"PS\",\"label\":\"Palestine\"},{\"value\":\"PT\",\"label\":\"Portugal\"},{\"value\":\"PW\",\"label\":\"Palau\"},{\"value\":\"PY\",\"label\":\"Paraguay\"},{\"value\":\"QA\",\"label\":\"Qatar\"},{\"value\":\"RE\",\"label\":\"Réunion\"},{\"value\":\"RO\",\"label\":\"Romania\"},{\"value\":\"RS\",\"label\":\"Serbia\"},{\"value\":\"RU\",\"label\":\"Russia\"},{\"value\":\"RW\",\"label\":\"Rwanda\"},{\"value\":\"SA\",\"label\":\"Saudi Arabia\"},{\"value\":\"SB\",\"label\":\"Solomon Islands\"},{\"value\":\"SC\",\"label\":\"Seychelles\"},{\"value\":\"SD\",\"label\":\"Sudan\"},{\"value\":\"SE\",\"label\":\"Sweden\"},{\"value\":\"SG\",\"label\":\"Singapore\"},{\"value\":\"SH\",\"label\":\"Saint Helena\"},{\"value\":\"SI\",\"label\":\"Slovenia\"},{\"value\":\"SJ\",\"label\":\"Svalbard and Jan Mayen\"},{\"value\":\"SK\",\"label\":\"Slovakia\"},{\"value\":\"SL\",\"label\":\"Sierra Leone\"},{\"value\":\"SM\",\"label\":\"San Marino\"},{\"value\":\"SN\",\"label\":\"Senegal\"},{\"value\":\"SO\",\"label\":\"Somalia\"},{\"value\":\"SR\",\"label\":\"Suriname\"},{\"value\":\"SS\",\"label\":\"South Sudan\"},{\"value\":\"ST\",\"label\":\"São Tomé and Príncipe\"},{\"value\":\"SV\",\"label\":\"El Salvador\"},{\"value\":\"SY\",\"label\":\"Syria\"},{\"value\":\"SZ\",\"label\":\"Swaziland\"},{\"value\":\"TC\",\"label\":\"Turks and Caicos Islands\"},{\"value\":\"TD\",\"label\":\"Chad\"},{\"value\":\"TF\",\"label\":\"French Southern Territories\"},{\"value\":\"TG\",\"label\":\"Togo\"},{\"value\":\"TH\",\"label\":\"Thailand\"},{\"value\":\"TJ\",\"label\":\"Tajikistan\"},{\"value\":\"TK\",\"label\":\"Tokelau\"},{\"value\":\"TL\",\"label\":\"East Timor\"},{\"value\":\"TM\",\"label\":\"Turkmenistan\"},{\"value\":\"TN\",\"label\":\"Tunisia\"},{\"value\":\"TO\",\"label\":\"Tonga\"},{\"value\":\"TR\",\"label\":\"Turkey\"},{\"value\":\"TT\",\"label\":\"Trinidad and Tobago\"},{\"value\":\"TV\",\"label\":\"Tuvalu\"},{\"value\":\"TW\",\"label\":\"Taiwan\"},{\"value\":\"TZ\",\"label\":\"Tanzania\"},{\"value\":\"UA\",\"label\":\"Ukraine\"},{\"value\":\"UG\",\"label\":\"Uganda\"},{\"value\":\"UM\",\"label\":\"U.S. Minor Outlying Islands\"},{\"value\":\"US\",\"label\":\"United States\"},{\"value\":\"UY\",\"label\":\"Uruguay\"},{\"value\":\"UZ\",\"label\":\"Uzbekistan\"},{\"value\":\"VA\",\"label\":\"Vatican City\"},{\"value\":\"VC\",\"label\":\"Saint Vincent and the Grenadines\"},{\"value\":\"VE\",\"label\":\"Venezuela\"},{\"value\":\"VG\",\"label\":\"British Virgin Islands\"},{\"value\":\"VI\",\"label\":\"U.S. Virgin Islands\"},{\"value\":\"VN\",\"label\":\"Vietnam\"},{\"value\":\"VU\",\"label\":\"Vanuatu\"},{\"value\":\"WF\",\"label\":\"Wallis and Futuna\"},{\"value\":\"WS\",\"label\":\"Samoa\"},{\"value\":\"YE\",\"label\":\"Yemen\"},{\"value\":\"YT\",\"label\":\"Mayotte\"},{\"value\":\"ZA\",\"label\":\"South Africa\"},{\"value\":\"ZM\",\"label\":\"Zambia\"},{\"value\":\"ZW\",\"label\":\"Zimbabwe\"}]"
	listData := []interface{}{}
	mapCode:=map[string]string{}
	err := json.Unmarshal([]byte(regionsList), &listData)
	if err != nil {
		fmt.Println("UpdateRegions json err", err)
	}
	for i := 0; i < len(listData); i++ {
		mapData := listData[i].(map[string]interface{})
		title := mapData["label"].(string)
		title = strings.TrimSpace(title)
		code := mapData["value"].(string)
		code = strings.TrimSpace(code)
		mapCode[title]=code
	//}

	//playwright.Install(&playwright.RunOptions{Verbose: true, DriverDirectory: "/home/tamnb/.cache/"})
	//pw, err := playwright.Run()
	//browser, err := pw.Chromium.Launch()
	//page, _ := browser.NewPage(playwright.BrowserNewContextOptions{
	//	UserAgent: &userAgent,
	//})
	//fmt.Println("NewPage")
	//page.Goto("https://ads.tiktok.com/business/creativecenter/music/mobile/en")
	//time.Sleep(30 * time.Second)

	//regionElements, err := page.QuerySelectorAll("div[class*=byted-select-popover-panel-search] div[class*=byted-select-popover-panel-inner] div[class*=byted-list-item-inner-wrapper]")
	//if err != nil {
	//	log.Fatalf("could not find region Element : %v", err)
	//}
	//fmt.Println(len(mapCode),len(regionElements))
	//for i := 0; i < len(regionElements); i++ {
	//	title, _ := regionElements[i].TextContent()
	//	title = strings.TrimSpace(title)
	//	code,exit:=mapCode[title]
	//	if !exit {
	//		fmt.Println("region not found code ", title, code)
	//		continue
	//	}
		data,err:=GetAudioDataFromTiktok(model.TiktokPostAudioRequest{
			Page: 1,
			Limit: 20,
			Region: code,
		})
		if err!=nil {
			fmt.Println("error when get data from region ", title, code ,err)
			continue
		}
		if len(data.Data.List)==0 {
			fmt.Println("error: Not found data from region ", title, code , " data = 0")
			continue
		}
		//fmt.Println("data from region ", title, code , " count " ,data.Data.Pagination.TotalCount)
		//_, err = db.InsertRegionInfo(db.Region{Title: title, Code: code})
		//if err != nil {
		//	fmt.Println("UpdateRegions InsertRegionInfo err", err, title, code)
		//}
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
