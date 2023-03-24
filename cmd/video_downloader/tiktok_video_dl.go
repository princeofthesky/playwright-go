package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/playwright-community/playwright-go"
	"github.com/playwright-community/playwright-go/model"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	httpPort    = flag.String("http_port", "9090", "http_port listen")
	videosCache = cache.New(2*time.Hour, 0)
	datr        = "-3P0YynNybl-r4JDs6iZSvVZ"
)

const Path_Service_Name = "tiktok_videos"

func LoadCookies() {
	err := playwright.Install(&playwright.RunOptions{Verbose: true})
	if err != nil {
		log.Fatalf("could not install driver %v", err)
	}
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start playwright: %v", err)
	}
	browser, err := pw.Chromium.Launch()
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	usergant := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"
	page, err := browser.NewPage(playwright.BrowserNewContextOptions{
		UserAgent: &usergant,
	})
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}
	if _, err = page.Goto("https://www.facebook.com/watch/?v=173654149476708"); err != nil {
		log.Fatalf("could not goto: %v", err)
	}
	time.Sleep(30 * time.Second)
	cookies, _ := page.Context().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "datr" {
			datr = cookie.Value
			fmt.Println("reload Cookie", datr, int(cookie.Expires))
		}
	}
}
func main() {
	LoadCookies()
	go func() {
		for _ = range time.Tick(2 * time.Hour) {
			LoadCookies()
		}
	}()
	flag.Parse()
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET(Path_Service_Name+"/videos", GetTiktokVideoInfoByURL)
	r.GET(Path_Service_Name+"/videos/fb", GetFbVideoInfoByURL)
	r.Run(":" + *httpPort)
}

func GetTiktokVideoId(url string) string {
	texts := strings.Split(url, "?")
	texts = strings.Split(texts[0], "/")
	return texts[len(texts)-1]
}

func GetTiktokVideoInfo(videoId string) (map[string]string, error) {
	res, err := http.Get("https://api16-normal-c-useast1a.tiktokv.com/aweme/v1/feed/?aweme_id=" + videoId)
	if err != nil {
		return nil, errors.New("error making http request: " + err.Error())
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, errors.New("error when Unmarshal body json")
	}
	if _, exit := data["aweme_list"]; !exit {
		return nil, errors.New("error not found aweme_list")
	}
	awemeList := data["aweme_list"].([]interface{})
	response := map[string]string{}
	for _, awesome := range awemeList {
		info := awesome.(map[string]interface{})
		//
		if _, exit := info["aweme_id"]; !exit {
			return nil, errors.New("error not found aweme_id")
		}
		awemeId := info["aweme_id"].(string)
		if awemeId != videoId {
			continue
		}
		//
		if _, exit := info["video"]; !exit {
			return nil, errors.New("error not found video")
		}
		video := info["video"].(map[string]interface{})
		//
		if _, exit := video["play_addr"]; !exit {
			fmt.Println(info)
			return nil, errors.New("error not found play_addr")
		}
		playAddr := video["play_addr"].(map[string]interface{})
		//
		if _, exit := playAddr["url_list"]; !exit {
			return nil, errors.New("error not found url_list")
		}
		urlList := playAddr["url_list"].([]interface{})
		//
		response["video"] = urlList[len(urlList)-1].(string)
		//
		response["desc"] = ""
		if _, exit := info["desc"]; exit {
			response["desc"] = info["desc"].(string)
		}
		//
		if _, exit := video["origin_cover"]; !exit {
			return nil, errors.New("error not found origin_cover")
		}
		originCover := video["origin_cover"].(map[string]interface{})
		//
		if _, exit := originCover["url_list"]; !exit {
			return nil, errors.New("error not found originCover url_list")
		}
		coverUrlList := originCover["url_list"].([]interface{})
		//
		response["cover"] = coverUrlList[len(coverUrlList)-1].(string)
		return response, nil
	}
	return nil, errors.New("not found video info")
}

func GetTiktokVideoInFormCache(videoId string) map[string]string {
	videoInfo, ok := videosCache.Get(videoId)
	if videoInfo != nil && ok {
		return videoInfo.(map[string]string)
	}
	videoInfo, err := GetTiktokVideoInfo(videoId)
	if err != nil {
		fmt.Println("err", err)
	}
	if err == nil && videoInfo != nil {
		videosCache.Add(videoId, videoInfo, cache.DefaultExpiration)
	}
	return videoInfo.(map[string]string)
}

func GetTiktokVideoInfoByURL(c *gin.Context) {
	url, exit := c.GetQuery("url")
	if !exit {
		url = "0"
	}
	videoId := GetTiktokVideoId(url)
	response := GetTiktokVideoInFormCache(videoId)
	responseData, _ := json.Marshal(model.HttpResponse{
		Code: model.HttpSuccess,
		Msg:  "",
		Data: response,
	})
	c.Data(200, "text/html; charset=UTF-8", responseData)
}

func GetFbVideoId(url string) string {
	if strings.Contains(url, "facebook.com/watch/?") {
		texts := strings.Split(url, "?")
		if len(texts) < 2 {
			return ""
		}
		texts = strings.Split(texts[1], "v=")
		if len(texts) < 2 {
			return ""
		}
		texts = strings.Split(texts[1], "&")
		return texts[0]
	} else if strings.Contains(url, "/videos/") {
		texts := strings.Split(url, "?")
		texts = strings.Split(texts[0], "/")
		return texts[len(texts)-1]
	}
	return ""
}

func GetFbVideoInfoFromDocumentJson(videoId, document string) (map[string]string, error) {
	fmt.Println(document)
	videoInfo := map[string]string{}
	var doc map[string]interface{}
	json.Unmarshal([]byte(document), &doc)
	requires := doc["require"].([]interface{})
	for _, requireInterface := range requires {
		if require, ok := requireInterface.([]interface{}); ok {
			for _, xI := range require {
				if x, ok := xI.([]interface{}); ok {
					for _, y := range x {
						if y.(map[string]interface{})["__bbox"] == nil {
							continue
						}
						bbox := y.(map[string]interface{})["__bbox"].(map[string]interface{})
						require1s := bbox["require"].([]interface{})
						for _, require1Interface := range require1s {
							if require1, ok := require1Interface.([]interface{}); ok {
								for _, require2Interface := range require1 {
									if require2, ok := require2Interface.([]interface{}); ok {
										for _, require3Interface := range require2 {
											if require3, ok := require3Interface.(map[string]interface{}); ok {
												bbox1 := require3["__bbox"].(map[string]interface{})
												result := bbox1["result"].(map[string]interface{})
												data := result["data"].(map[string]interface{})
												video := data["video"].(map[string]interface{})
												id := video["id"].(string)
												if id != videoId {
													continue
												}
												story := video["story"].(map[string]interface{})
												attachments := story["attachments"].([]interface{})
												for _, attachmentInterface := range attachments {
													attachment := attachmentInterface.(map[string]interface{})
													media := attachment["media"].(map[string]interface{})
													preferred_thumbnail := media["preferred_thumbnail"].(map[string]interface{})
													image := preferred_thumbnail["image"].(map[string]interface{})
													if uri, ok := image["uri"].(string); ok {
														videoInfo["cover"] = uri
													} else {
														videoInfo["cover"] = ""
													}
													if videoSd, ok := media["playable_url"].(string); ok {
														videoInfo["video_sd"] = videoSd
													} else {
														videoInfo["video_sd"] = ""
													}
													if videoHd, ok := media["playable_url_quality_hd"].(string); ok {
														videoInfo["video_hd"] = videoHd
													} else {
														videoInfo["video_hd"] = ""
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	if (len(videoInfo["video_sd"]) > 0) || (len(videoInfo["video_hd"]) > 0) {
		return videoInfo, nil
	} else {
		return videoInfo, nil
	}
}
func GetFieldFromDocument(document, field string) string {
	texts := strings.Split(document, field)
	if len(texts) < 2 {
		return ""
	}
	texts = strings.Split(texts[1], "\"")
	text := texts[0]
	jsonText := "{\"a\":\"" + text + "\"}"
	var mapValue map[string]string
	json.Unmarshal([]byte(jsonText), &mapValue)
	return mapValue["a"]
}

func GetFbVideoInfoFromDocument(document string) (map[string]string, error) {
	videoInfo := map[string]string{}
	videoInfo["cover"] = GetFieldFromDocument(document, "\"preferred_thumbnail\":{\"image\":{\"uri\":\"")
	videoInfo["video_sd"] = GetFieldFromDocument(document, "\"playable_url\":\"")
	videoInfo["video_hd"] = GetFieldFromDocument(document, "\"playable_url_quality_hd\":\"")

	if (len(videoInfo["video_sd"]) > 0) || (len(videoInfo["video_hd"]) > 0) {
		return videoInfo, nil
	} else {
		return videoInfo, nil
	}
}
func GetFbVideoInfo(videoId string) (map[string]string, error) {
	client := http.DefaultClient
	req, err := http.NewRequest("GET", "https://www.facebook.com/aaa/videos/"+videoId, nil)
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36")
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	//req.AddCookie(&http.Cookie{	Name: "sb",	Value: "D9azY5k9BdyX-MJiCPbF9v65",	})
	//req.AddCookie(&http.Cookie{	Name: "datr",	Value: "D9azYwrr-uCzESQ4havN2Kun",	})

	req.AddCookie(&http.Cookie{Name: "datr", Value: datr})
	//req.AddCookie(&http.Cookie{	Name: "fr",	Value: "kew3vGS27j94CXNU.AWVPJmNyboUbL7FDhrhTJ_Q1qvg.Bj8xwI.rG.AAA.0.0.Bj8xwI.AWXVDIzM1ks",	})
	//req.AddCookie(&http.Cookie{	Name: "presence",	Value: "C%7B%22t3%22%3A%5B%5D%2C%22utc3%22%3A1676876814402%2C%22v%22%3A1%7D",	})
	//req.AddCookie(&http.Cookie{	Name: "wd",	Value: "1536x775",	})
	reponse, err := client.Do(req)
	doc, err := goquery.NewDocumentFromReader(reponse.Body)
	if err != nil {
		log.Fatal(err)
	}
	info := map[string]string{}
	// Find the review items
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the title
		ret := s.Text()
		if strings.Contains(ret, ".mp4") && strings.Contains(ret, videoId) {
			videoInfo, err := GetFbVideoInfoFromDocument(ret)
			if err == nil {
				videoInfo["title"]=videoId
				info = videoInfo
			}
		}
	})
	return info, err
}

func GetFbVideoInFormCache(videoId string) map[string]string {
	videoInfo, ok := videosCache.Get(videoId)
	if videoInfo != nil && ok {
		return videoInfo.(map[string]string)
	}
	videoInfo, err := GetFbVideoInfo(videoId)
	if err != nil {
		fmt.Println("err", err)
	}
	if err == nil && videoInfo != nil {
		videosCache.Add(videoId, videoInfo, cache.DefaultExpiration)
	}
	return videoInfo.(map[string]string)
}

func GetFbVideoInfoByURL(c *gin.Context) {
	url, exit := c.GetQuery("url")
	if !exit {
		responseData, _ := json.Marshal(model.HttpResponse{
			Code: model.HttpFail,
			Msg:  "url not found",
			Data: nil,
		})
		c.Data(200, "text/html; charset=UTF-8", responseData)
		return
	}
	videoId := GetFbVideoId(url)
	if len(videoId) < 2 {
		responseData, _ := json.Marshal(model.HttpResponse{
			Code: model.HttpFail,
			Msg:  "video not found : " + url,
			Data: nil,
		})
		c.Data(200, "text/html; charset=UTF-8", responseData)
		return
	}
	response := GetFbVideoInFormCache(videoId)
	responseData, _ := json.Marshal(model.HttpResponse{
		Code: model.HttpSuccess,
		Msg:  "",
		Data: response,
	})
	c.Data(200, "text/html; charset=UTF-8", responseData)
}
