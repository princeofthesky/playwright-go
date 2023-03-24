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
	fmt.Println(dataResponse.Data.Pagination.Page, dataResponse.Data.Pagination.TotalCount, dataResponse.Data.Pagination.HasMore, len(dataResponse.Data.List))
	fmt.Println(string(body))
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

	regionsList := "[{\"name\":\"Abkhazia\",\"shortName\":\"AB\",\"area\":\"+7840\"},{\"name\":\"Abkhazia\",\"shortName\":\"AB\",\"area\":\"+7940\"},{\"name\":\"Abkhazia\",\"shortName\":\"AB\",\"area\":\"+99544\"},{\"name\":\"Afghanistan\",\"shortName\":\"AF\",\"area\":\"+93\"},{\"name\":\"Åland Islands\",\"shortName\":\"AX\",\"area\":\"+35818\"},{\"name\":\"Albania\",\"shortName\":\"AL\",\"area\":\"+355\"},{\"name\":\"Algeria\",\"shortName\":\"DZ\",\"area\":\"+213\"},{\"name\":\"American Samoa\",\"shortName\":\"AS\",\"area\":\"+1684\"},{\"name\":\"Andorra\",\"shortName\":\"AD\",\"area\":\"+376\"},{\"name\":\"Angola\",\"shortName\":\"AO\",\"area\":\"+244\"},{\"name\":\"Anguilla\",\"shortName\":\"AI\",\"area\":\"+1264\"},{\"name\":\"Antigua and Barbuda\",\"shortName\":\"AG\",\"area\":\"+1268\"},{\"name\":\"Argentina\",\"shortName\":\"AR\",\"area\":\"+54\"},{\"name\":\"Armenia\",\"shortName\":\"AM\",\"area\":\"+374\"},{\"name\":\"Aruba\",\"shortName\":\"AW\",\"area\":\"+297\"},{\"name\":\"Saint Helena, Ascension and Tristan da Cunha\",\"shortName\":\"SH\",\"area\":\"+247\"},{\"name\":\"Australia\",\"shortName\":\"AU\",\"area\":\"+61\"},{\"name\":\"Australia\",\"shortName\":\"AU\",\"area\":\"+672\"},{\"name\":\"Austria\",\"shortName\":\"AT\",\"area\":\"+43\"},{\"name\":\"Azerbaijan\",\"shortName\":\"AZ\",\"area\":\"+994\"},{\"name\":\"Bahamas\",\"shortName\":\"BS\",\"area\":\"+1242\"},{\"name\":\"Bahrain\",\"shortName\":\"BH\",\"area\":\"+973\"},{\"name\":\"Bangladesh\",\"shortName\":\"BD\",\"area\":\"+880\"},{\"name\":\"Barbados\",\"shortName\":\"BB\",\"area\":\"+1246\"},{\"name\":\"Antigua and Barbuda\",\"shortName\":\"AG\",\"area\":\"+1268\"},{\"name\":\"Belarus\",\"shortName\":\"BY\",\"area\":\"+375\"},{\"name\":\"Belgium\",\"shortName\":\"BE\",\"area\":\"+32\"},{\"name\":\"Belize\",\"shortName\":\"BZ\",\"area\":\"+501\"},{\"name\":\"Benin\",\"shortName\":\"BJ\",\"area\":\"+229\"},{\"name\":\"Bermuda\",\"shortName\":\"BM\",\"area\":\"+1441\"},{\"name\":\"Bhutan\",\"shortName\":\"BT\",\"area\":\"+975\"},{\"name\":\"Bolivia, Plurinational State of\",\"shortName\":\"BO\",\"area\":\"+591\"},{\"name\":\"Bosnia and Herzegovina\",\"shortName\":\"BA\",\"area\":\"+387\"},{\"name\":\"Botswana\",\"shortName\":\"BW\",\"area\":\"+267\"},{\"name\":\"Brazil\",\"shortName\":\"BR\",\"area\":\"+55\"},{\"name\":\"British Indian Ocean Territory\",\"shortName\":\"IO\",\"area\":\"+246\"},{\"name\":\"Virgin Islands, British\",\"shortName\":\"VG\",\"area\":\"+1284\"},{\"name\":\"Brunei Darussalam\",\"shortName\":\"BN\",\"area\":\"+673\"},{\"name\":\"Bulgaria\",\"shortName\":\"BG\",\"area\":\"+359\"},{\"name\":\"Burkina Faso\",\"shortName\":\"BF\",\"area\":\"+226\"},{\"name\":\"Burundi\",\"shortName\":\"BI\",\"area\":\"+257\"},{\"name\":\"Cambodia\",\"shortName\":\"KH\",\"area\":\"+855\"},{\"name\":\"Cameroon\",\"shortName\":\"CM\",\"area\":\"+237\"},{\"name\":\"Canada\",\"shortName\":\"CA\",\"area\":\"+1\"},{\"name\":\"Cape Verde\",\"shortName\":\"CV\",\"area\":\"+238\"},{\"name\":\"Bonaire, Sint Eustatius and Saba\",\"shortName\":\"BQ\",\"area\":\"+5997\"},{\"name\":\"Cayman Islands\",\"shortName\":\"KY\",\"area\":\"+1345\"},{\"name\":\"Central African Republic\",\"shortName\":\"CF\",\"area\":\"+236\"},{\"name\":\"Chad\",\"shortName\":\"TD\",\"area\":\"+235\"},{\"name\":\"Chile\",\"shortName\":\"CL\",\"area\":\"+56\"},{\"name\":\"China\",\"shortName\":\"CN\",\"area\":\"+86\"},{\"name\":\"Christmas Island\",\"shortName\":\"CX\",\"area\":\"+61\"},{\"name\":\"Cocos (Keeling) Islands\",\"shortName\":\"CC\",\"area\":\"+61\"},{\"name\":\"Colombia\",\"shortName\":\"CO\",\"area\":\"+57\"},{\"name\":\"Comoros\",\"shortName\":\"KM\",\"area\":\"+269\"},{\"name\":\"Congo\",\"shortName\":\"CG\",\"area\":\"+242\"},{\"name\":\"Cook Islands\",\"shortName\":\"CK\",\"area\":\"+682\"},{\"name\":\"Costa Rica\",\"shortName\":\"CR\",\"area\":\"+506\"},{\"name\":\"Croatia\",\"shortName\":\"HR\",\"area\":\"+385\"},{\"name\":\"Curaçao\",\"shortName\":\"CW\",\"area\":\"+5999\"},{\"name\":\"Cyprus\",\"shortName\":\"CY\",\"area\":\"+357\"},{\"name\":\"Czech Republic\",\"shortName\":\"CZ\",\"area\":\"+420\"},{\"name\":\"Denmark\",\"shortName\":\"DK\",\"area\":\"+45\"},{\"name\":\"Diego Garcia\",\"shortName\":\"DG\",\"area\":\"+246\"},{\"name\":\"Djibouti\",\"shortName\":\"DJ\",\"area\":\"+253\"},{\"name\":\"Dominica\",\"shortName\":\"DM\",\"area\":\"+1767\"},{\"name\":\"Dominican Republic\",\"shortName\":\"DO\",\"area\":\"+1809\"},{\"name\":\"Dominican Republic\",\"shortName\":\"DO\",\"area\":\"+1829\"},{\"name\":\"Dominican Republic\",\"shortName\":\"DO\",\"area\":\"+1849\"},{\"name\":\"Russian Federation\",\"shortName\":\"RU\",\"area\":\"+7\"},{\"name\":\"Ecuador\",\"shortName\":\"EC\",\"area\":\"+593\"},{\"name\":\"Egypt\",\"shortName\":\"EG\",\"area\":\"+20\"},{\"name\":\"El Salvador\",\"shortName\":\"SV\",\"area\":\"+503\"},{\"name\":\"Equatorial Guinea\",\"shortName\":\"GQ\",\"area\":\"+240\"},{\"name\":\"Eritrea\",\"shortName\":\"ER\",\"area\":\"+291\"},{\"name\":\"Estonia\",\"shortName\":\"EE\",\"area\":\"+372\"},{\"name\":\"Ethiopia\",\"shortName\":\"ET\",\"area\":\"+251\"},{\"name\":\"Falkland Islands (Malvinas)\",\"shortName\":\"FK\",\"area\":\"+500\"},{\"name\":\"Faroe Islands\",\"shortName\":\"FO\",\"area\":\"+298\"},{\"name\":\"Fiji\",\"shortName\":\"FJ\",\"area\":\"+679\"},{\"name\":\"Finland\",\"shortName\":\"FI\",\"area\":\"+358\"},{\"name\":\"France\",\"shortName\":\"FR\",\"area\":\"+33\"},{\"name\":\"French Guiana\",\"shortName\":\"GF\",\"area\":\"+594\"},{\"name\":\"French Polynesia\",\"shortName\":\"PF\",\"area\":\"+689\"},{\"name\":\"Gabon\",\"shortName\":\"GA\",\"area\":\"+241\"},{\"name\":\"Gambia\",\"shortName\":\"GM\",\"area\":\"+220\"},{\"name\":\"Georgia\",\"shortName\":\"GE\",\"area\":\"+995\"},{\"name\":\"Germany\",\"shortName\":\"DE\",\"area\":\"+49\"},{\"name\":\"Ghana\",\"shortName\":\"GH\",\"area\":\"+233\"},{\"name\":\"Gibraltar\",\"shortName\":\"GI\",\"area\":\"+350\"},{\"name\":\"Greece\",\"shortName\":\"GR\",\"area\":\"+30\"},{\"name\":\"Greenland\",\"shortName\":\"GL\",\"area\":\"+299\"},{\"name\":\"Grenada\",\"shortName\":\"GD\",\"area\":\"+1473\"},{\"name\":\"Guadeloupe\",\"shortName\":\"GP\",\"area\":\"+590\"},{\"name\":\"Guam\",\"shortName\":\"GU\",\"area\":\"+1671\"},{\"name\":\"Guatemala\",\"shortName\":\"GT\",\"area\":\"+502\"},{\"name\":\"Guernsey\",\"shortName\":\"GG\",\"area\":\"+44\"},{\"name\":\"Guinea\",\"shortName\":\"GN\",\"area\":\"+224\"},{\"name\":\"Guinea-Bissau\",\"shortName\":\"GW\",\"area\":\"+245\"},{\"name\":\"Guyana\",\"shortName\":\"GY\",\"area\":\"+592\"},{\"name\":\"Haiti\",\"shortName\":\"HT\",\"area\":\"+509\"},{\"name\":\"Honduras\",\"shortName\":\"HN\",\"area\":\"+504\"},{\"name\":\"Hong Kong\",\"shortName\":\"HK\",\"area\":\"+852\"},{\"name\":\"Hungary\",\"shortName\":\"HU\",\"area\":\"+36\"},{\"name\":\"Iceland\",\"shortName\":\"IS\",\"area\":\"+354\"},{\"name\":\"India\",\"shortName\":\"IN\",\"area\":\"+91\"},{\"name\":\"Indonesia\",\"shortName\":\"ID\",\"area\":\"+62\"},{\"name\":\"Iraq\",\"shortName\":\"IQ\",\"area\":\"+964\"},{\"name\":\"Ireland\",\"shortName\":\"IE\",\"area\":\"+353\"},{\"name\":\"Israel\",\"shortName\":\"IL\",\"area\":\"+972\"},{\"name\":\"Italy\",\"shortName\":\"IT\",\"area\":\"+39\"},{\"name\":\"Jamaica\",\"shortName\":\"JM\",\"area\":\"+1876\"},{\"name\":\"Japan\",\"shortName\":\"JP\",\"area\":\"+81\"},{\"name\":\"Jersey\",\"shortName\":\"JE\",\"area\":\"+44\"},{\"name\":\"Jordan\",\"shortName\":\"JO\",\"area\":\"+962\"},{\"name\":\"Kazakhstan\",\"shortName\":\"KZ\",\"area\":\"+76\"},{\"name\":\"Kazakhstan\",\"shortName\":\"KZ\",\"area\":\"+77\"},{\"name\":\"Kenya\",\"shortName\":\"KE\",\"area\":\"+254\"},{\"name\":\"Kiribati\",\"shortName\":\"KI\",\"area\":\"+686\"},{\"name\":\"Kuwait\",\"shortName\":\"KW\",\"area\":\"+965\"},{\"name\":\"Kyrgyzstan\",\"shortName\":\"KG\",\"area\":\"+996\"},{\"name\":\"Lao People's Democratic Republic\",\"shortName\":\"LA\",\"area\":\"+856\"},{\"name\":\"Latvia\",\"shortName\":\"LV\",\"area\":\"+371\"},{\"name\":\"Lebanon\",\"shortName\":\"LB\",\"area\":\"+961\"},{\"name\":\"Lesotho\",\"shortName\":\"LS\",\"area\":\"+266\"},{\"name\":\"Libya\",\"shortName\":\"LY\",\"area\":\"+218\"},{\"name\":\"Liechtenstein\",\"shortName\":\"LI\",\"area\":\"+423\"},{\"name\":\"Lithuania\",\"shortName\":\"LT\",\"area\":\"+370\"},{\"name\":\"Luxembourg\",\"shortName\":\"LU\",\"area\":\"+352\"},{\"name\":\"Macao\",\"shortName\":\"MO\",\"area\":\"+853\"},{\"name\":\"Macedonia, the Former Yugoslav Republic of\",\"shortName\":\"MK\",\"area\":\"+389\"},{\"name\":\"Madagascar\",\"shortName\":\"MG\",\"area\":\"+261\"},{\"name\":\"Malawi\",\"shortName\":\"MW\",\"area\":\"+265\"},{\"name\":\"Malaysia\",\"shortName\":\"MY\",\"area\":\"+60\"},{\"name\":\"Maldives\",\"shortName\":\"MV\",\"area\":\"+960\"},{\"name\":\"Mali\",\"shortName\":\"ML\",\"area\":\"+223\"},{\"name\":\"Malta\",\"shortName\":\"MT\",\"area\":\"+356\"},{\"name\":\"Marshall Islands\",\"shortName\":\"MH\",\"area\":\"+692\"},{\"name\":\"Martinique\",\"shortName\":\"MQ\",\"area\":\"+596\"},{\"name\":\"Mauritania\",\"shortName\":\"MR\",\"area\":\"+222\"},{\"name\":\"Mauritius\",\"shortName\":\"MU\",\"area\":\"+230\"},{\"name\":\"Mayotte\",\"shortName\":\"YT\",\"area\":\"+262\"},{\"name\":\"Mexico\",\"shortName\":\"MX\",\"area\":\"+52\"},{\"name\":\"Micronesia, Federated States of\",\"shortName\":\"FM\",\"area\":\"+691\"},{\"name\":\"Moldova, Republic of\",\"shortName\":\"MD\",\"area\":\"+373\"},{\"name\":\"Monaco\",\"shortName\":\"MC\",\"area\":\"+377\"},{\"name\":\"Mongolia\",\"shortName\":\"MN\",\"area\":\"+976\"},{\"name\":\"Montenegro\",\"shortName\":\"ME\",\"area\":\"+382\"},{\"name\":\"Montserrat\",\"shortName\":\"MS\",\"area\":\"+1664\"},{\"name\":\"Morocco\",\"shortName\":\"MA\",\"area\":\"+212\"},{\"name\":\"Mozambique\",\"shortName\":\"MZ\",\"area\":\"+258\"},{\"name\":\"Namibia\",\"shortName\":\"NA\",\"area\":\"+264\"},{\"name\":\"Nauru\",\"shortName\":\"NR\",\"area\":\"+674\"},{\"name\":\"Nepal\",\"shortName\":\"NP\",\"area\":\"+977\"},{\"name\":\"Netherlands\",\"shortName\":\"NL\",\"area\":\"+31\"},{\"name\":\"New Caledonia\",\"shortName\":\"NC\",\"area\":\"+687\"},{\"name\":\"New Zealand\",\"shortName\":\"NZ\",\"area\":\"+64\"},{\"name\":\"Nicaragua\",\"shortName\":\"NI\",\"area\":\"+505\"},{\"name\":\"Niger\",\"shortName\":\"NE\",\"area\":\"+227\"},{\"name\":\"Nigeria\",\"shortName\":\"NG\",\"area\":\"+234\"},{\"name\":\"Niue\",\"shortName\":\"NU\",\"area\":\"+683\"},{\"name\":\"Norfolk Island\",\"shortName\":\"NF\",\"area\":\"+672\"},{\"name\":\"Northern Mariana Islands\",\"shortName\":\"MP\",\"area\":\"+1670\"},{\"name\":\"Norway\",\"shortName\":\"NO\",\"area\":\"+47\"},{\"name\":\"Oman\",\"shortName\":\"OM\",\"area\":\"+968\"},{\"name\":\"Pakistan\",\"shortName\":\"PK\",\"area\":\"+92\"},{\"name\":\"Palau\",\"shortName\":\"PW\",\"area\":\"+680\"},{\"name\":\"Palestine, State of\",\"shortName\":\"PS\",\"area\":\"+970\"},{\"name\":\"Panama\",\"shortName\":\"PA\",\"area\":\"+507\"},{\"name\":\"Papua New Guinea\",\"shortName\":\"PG\",\"area\":\"+675\"},{\"name\":\"Paraguay\",\"shortName\":\"PY\",\"area\":\"+595\"},{\"name\":\"Peru\",\"shortName\":\"PE\",\"area\":\"+51\"},{\"name\":\"Philippines\",\"shortName\":\"PH\",\"area\":\"+63\"},{\"name\":\"Pitcairn\",\"shortName\":\"PN\",\"area\":\"+64\"},{\"name\":\"Poland\",\"shortName\":\"PL\",\"area\":\"+48\"},{\"name\":\"Portugal\",\"shortName\":\"PT\",\"area\":\"+351\"},{\"name\":\"Puerto Rico\",\"shortName\":\"PR\",\"area\":\"+1787\"},{\"name\":\"Puerto Rico\",\"shortName\":\"PR\",\"area\":\"+1939\"},{\"name\":\"Qatar\",\"shortName\":\"QA\",\"area\":\"+974\"},{\"name\":\"Romania\",\"shortName\":\"RO\",\"area\":\"+40\"},{\"name\":\"Russian Federation\",\"shortName\":\"RU\",\"area\":\"+7\"},{\"name\":\"Rwanda\",\"shortName\":\"RW\",\"area\":\"+250\"},{\"name\":\"Reunion Island\",\"shortName\":\"SURVEY\",\"area\":\"+262\"},{\"name\":\"Samoa\",\"shortName\":\"WS\",\"area\":\"+685\"},{\"name\":\"San Marino\",\"shortName\":\"SM\",\"area\":\"+378\"},{\"name\":\"Saudi Arabia\",\"shortName\":\"SA\",\"area\":\"+966\"},{\"name\":\"Senegal\",\"shortName\":\"SN\",\"area\":\"+221\"},{\"name\":\"Serbia\",\"shortName\":\"RS\",\"area\":\"+381\"},{\"name\":\"Seychelles\",\"shortName\":\"SC\",\"area\":\"+248\"},{\"name\":\"Sierra Leone\",\"shortName\":\"SL\",\"area\":\"+232\"},{\"name\":\"Singapore\",\"shortName\":\"SG\",\"area\":\"+65\"},{\"name\":\"Bonaire, Sint Eustatius and Saba\",\"shortName\":\"BQ\",\"area\":\"+5993\"},{\"name\":\"Sint Maarten (Dutch part)\",\"shortName\":\"SX\",\"area\":\"+1721\"},{\"name\":\"Slovakia\",\"shortName\":\"SK\",\"area\":\"+421\"},{\"name\":\"Slovenia\",\"shortName\":\"SI\",\"area\":\"+386\"},{\"name\":\"Solomon Islands\",\"shortName\":\"SB\",\"area\":\"+677\"},{\"name\":\"Somalia\",\"shortName\":\"SO\",\"area\":\"+252\"},{\"name\":\"South Africa\",\"shortName\":\"ZA\",\"area\":\"+27\"},{\"name\":\"South Georgia and the South Sandwich Islands\",\"shortName\":\"GS\",\"area\":\"+500\"},{\"name\":\"Korea, Republic of\",\"shortName\":\"KR\",\"area\":\"+82\"},{\"name\":\"Singapore\",\"shortName\":\"SG\",\"area\":\"+99534\"},{\"name\":\"South Sudan\",\"shortName\":\"SS\",\"area\":\"+211\"},{\"name\":\"Spain\",\"shortName\":\"ES\",\"area\":\"+34\"},{\"name\":\"Sri Lanka\",\"shortName\":\"LK\",\"area\":\"+94\"},{\"name\":\"Saint Barthélemy\",\"shortName\":\"BL\",\"area\":\"+590\"},{\"name\":\"Saint Helena, Ascension and Tristan da Cunha\",\"shortName\":\"SH\",\"area\":\"+290\"},{\"name\":\"Saint Kitts and Nevis\",\"shortName\":\"KN\",\"area\":\"+1869\"},{\"name\":\"Saint Lucia\",\"shortName\":\"LC\",\"area\":\"+1758\"},{\"name\":\"Saint Martin (French part)\",\"shortName\":\"MF\",\"area\":\"+590\"},{\"name\":\"Saint Pierre and Miquelon\",\"shortName\":\"PM\",\"area\":\"+508\"},{\"name\":\"Saint Vincent and the Grenadines\",\"shortName\":\"VC\",\"area\":\"+1784\"},{\"name\":\"Suriname\",\"shortName\":\"SR\",\"area\":\"+597\"},{\"name\":\"Svalbard and Jan Mayen\",\"shortName\":\"SJ\",\"area\":\"+4779\"},{\"name\":\"Svalbard and Jan Mayen\",\"shortName\":\"SJ\",\"area\":\"+4779\"},{\"name\":\"Swaziland\",\"shortName\":\"SZ\",\"area\":\"+268\"},{\"name\":\"Sweden\",\"shortName\":\"SE\",\"area\":\"+46\"},{\"name\":\"Switzerland\",\"shortName\":\"CH\",\"area\":\"+41\"},{\"name\":\"Sao Tome and Principe\",\"shortName\":\"ST\",\"area\":\"+239\"},{\"name\":\"Taiwan\",\"shortName\":\"TW\",\"area\":\"+886\"},{\"name\":\"Tajikistan\",\"shortName\":\"TJ\",\"area\":\"+992\"},{\"name\":\"Tanzania, United Republic of\",\"shortName\":\"TZ\",\"area\":\"+255\"},{\"name\":\"Thailand\",\"shortName\":\"TH\",\"area\":\"+66\"},{\"name\":\"Timor-Leste\",\"shortName\":\"TL\",\"area\":\"+670\"},{\"name\":\"Togo\",\"shortName\":\"TG\",\"area\":\"+228\"},{\"name\":\"Tokelau\",\"shortName\":\"TK\",\"area\":\"+690\"},{\"name\":\"Tonga\",\"shortName\":\"TO\",\"area\":\"+676\"},{\"name\":\"Trinidad and Tobago\",\"shortName\":\"TT\",\"area\":\"+1868\"},{\"name\":\"Tunisia\",\"shortName\":\"TN\",\"area\":\"+216\"},{\"name\":\"Turkey\",\"shortName\":\"TR\",\"area\":\"+90\"},{\"name\":\"Turkmenistan\",\"shortName\":\"TM\",\"area\":\"+993\"},{\"name\":\"Turks and Caicos Islands\",\"shortName\":\"TC\",\"area\":\"+1649\"},{\"name\":\"Tuvalu\",\"shortName\":\"TV\",\"area\":\"+688\"},{\"name\":\"Virgin Islands, U.S.\",\"shortName\":\"VI\",\"area\":\"+1340\"},{\"name\":\"Uganda\",\"shortName\":\"UG\",\"area\":\"+256\"},{\"name\":\"Ukraine\",\"shortName\":\"UA\",\"area\":\"+380\"},{\"name\":\"United Arab Emirates\",\"shortName\":\"AE\",\"area\":\"+971\"},{\"name\":\"United Kingdom\",\"shortName\":\"UK\",\"area\":\"+44\"},{\"name\":\"United States\",\"shortName\":\"US\",\"area\":\"+1\"},{\"name\":\"Uruguay\",\"shortName\":\"UY\",\"area\":\"+598\"},{\"name\":\"Uzbekistan\",\"shortName\":\"UZ\",\"area\":\"+998\"},{\"name\":\"Vanuatu\",\"shortName\":\"VU\",\"area\":\"+678\"},{\"name\":\"Holy See (Vatican City State)\",\"shortName\":\"VA\",\"area\":\"+3906698\"},{\"name\":\"Holy See (Vatican City State)\",\"shortName\":\"VA\",\"area\":\"+379\"},{\"name\":\"Venezuela, Bolivarian Republic of\",\"shortName\":\"VE\",\"area\":\"+58\"},{\"name\":\"Viet Nam\",\"shortName\":\"VN\",\"area\":\"+84\"},{\"name\":\"Wallis and Futuna\",\"shortName\":\"WF\",\"area\":\"+681\"},{\"name\":\"Yemen\",\"shortName\":\"YE\",\"area\":\"+967\"},{\"name\":\"Zambia\",\"shortName\":\"ZM\",\"area\":\"+260\"},{\"name\":\"Zanzibar\",\"shortName\":\"TZ\",\"area\":\"+255\"}]"
	listData := []interface{}{}
	mapCode:=map[string]string{}
	err := json.Unmarshal([]byte(regionsList), &listData)
	if err != nil {
		fmt.Println("UpdateRegions json err", err)
	}
	for i := 0; i < len(listData); i++ {
		mapData := listData[i].(map[string]interface{})
		title := mapData["name"].(string)
		title = strings.TrimSpace(title)
		code := mapData["shortName"].(string)
		code = strings.TrimSpace(code)
		mapCode[title]=code
	}

	playwright.Install(&playwright.RunOptions{Verbose: true, DriverDirectory: "/home/tamnb/.cache/"})
	pw, err := playwright.Run()
	browser, err := pw.Chromium.Launch()
	page, _ := browser.NewPage(playwright.BrowserNewContextOptions{
		UserAgent: &userAgent,
	})
	fmt.Println("NewPage")
	page.Goto("https://ads.tiktok.com/business/creativecenter/music/mobile/en")
	time.Sleep(30 * time.Second)

	regionElements, err := page.QuerySelectorAll("div[class*=byted-select-popover-panel-search] div[class*=byted-select-popover-panel-inner] div[class*=byted-list-item-inner-wrapper]")
	if err != nil {
		log.Fatalf("could not find region Element : %v", err)
	}
	fmt.Println(len(mapCode),len(regionElements))
	for i := 0; i < len(regionElements); i++ {
		title, _ := regionElements[i].TextContent()
		title = strings.TrimSpace(title)
		code,exit:=mapCode[title]
		if !exit {
			fmt.Println("region not found code ", title, code)
			continue
		}
		//data,err:=GetAudioDataFromTiktok(model.TiktokPostAudioRequest{
		//	Page: 1,
		//	Limit: 20,
		//	Region: code,
		//})
		//if err!=nil {
		//	fmt.Println("error when get data from region ", title, code ,err)
		//	continue
		//}
		//if len(data.Data.List)==0 {
		//	fmt.Println("error: Not found data from region ", title, code , " data = 0")
		//	continue
		//}
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
