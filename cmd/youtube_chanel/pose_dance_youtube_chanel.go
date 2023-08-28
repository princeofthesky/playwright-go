package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/pelletier/go-toml"
	"github.com/playwright-community/playwright-go"
	"github.com/playwright-community/playwright-go/config"
	"github.com/playwright-community/playwright-go/db"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	oldData    = flag.String("old_data", "./video_data.json", "list all old video in channel")
	conf    = flag.String("conf", "./pose_dance_api.toml", "config run file *.toml")
	c       = config.CrawlConfig{}
	channel = "https://www.youtube.com/@ARDanceGameOfficial/shorts"
	//var CONSISTENCY = "AKreu9s6iwhuuJkU3P74hVotFw1Nn_kqtGr5nkRbKcrBcHXEqs6ZKaXZGd9iiXUe2PmzkseJIjJMVQsW2sCKfsE1KnufBWQwvIud69NEaCqSTFuF_ywm-Z8"
	userAgent = "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Mobile Safari/537.36"

)

//var continuation = "4qmFsgL3ARIYVUNjcUliVzFnc3dpNC1wcmZYZzlIMWhBGtoBOGdhZEFScWFBVktYQVFxU0FRcHFRME00VVVGU2IyNXZaMWxyUTJob1ZsRXlUbmhUVjBwWVRWZGtlbVF5YXpCTVdFSjVXbXhvYms5VlozaGhSVVZSUVZKdlEwTkJRV2xCWjJkQlNXaEZTMFI2UlRaTlZGazFUVlJSTWs1VVJYcE5hbXN5VGxOdlRrTm5kRU5aYmxrMVVWUkdhbEV5VmpKaGR4SWtOalV4TkRWbFkyTXRNREF3TUMweU1qZzBMV0U1WVRRdE0yTXlPRFprTkdSbE5URXlHQUUlM0Q%3D"

//func loadCONSISTENCY() error {
//	request, _ := http.NewRequest("GET", channel, nil)
//	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36")
//	res, _ := http.DefaultClient.Do(request)
//	body, _ := io.ReadAll(res.Body)
//	data := string(body)
//	data = strings.TrimSpace(data)
//	splitText := strings.Split(data, "\"consistencyTokenJar\":{\"encryptedTokenJarContents\":\"")
//	if len(splitText) < 2 {
//		return errors.New("len(splitText) < 2")
//	}
//	data = splitText[1]
//	splitText = strings.Split(data, "\",\"")
//	newConsistency := splitText[0]
//	if len(newConsistency) < 20 {
//		return errors.New(" len(newConsistency)  < 20 , " + newConsistency)
//	}
//	CONSISTENCY = newConsistency
//	return nil
//}

//func loadVideoNextPage(continuation string, index int) string {
//	postBody := "{\"context\":{\"client\":{\"hl\":\"vi\",\"gl\":\"VN\",\"remoteHost\":\"\",\"deviceMake\":\"Google\",\"deviceModel\":\"Nexus 5\",\"visitorData\":\"CgtnZDlrSGpWUmhjQSis68amBg%3D%3D\",\"userAgent\":\"Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Mobile Safari/537.36,gzip(gfe)\",\"clientName\":\"WEB\",\"clientVersion\":\"2.20230807.01.00\",\"osName\":\"Android\",\"osVersion\":\"6.0\",\"originalUrl\":\"https://www.youtube.com/@ARDanceGameOfficial/shorts\",\"screenPixelDensity\":2,\"platform\":\"MOBILE\",\"clientFormFactor\":\"UNKNOWN_FORM_FACTOR\",\"configInfo\":{\"appInstallData\":\"CKzrxqYGENuvrwUQksuvBRD6vq8FEJrRrwUQzK7-EhCst68FEKXC_hIQ_eeoGBCMy68FEOe6rwUQ8qivBRCQz68FEMzfrgUQieiuBRCSz68FEJbOrwUQgqWvBRC0ya8FEO6irwUQ4LavBRCTz68FEJCjrwUQ6sOvBRC9tq4FENzPrwUQ5LP-EhCpxK8FEOLUrgUQ1KGvBRCPw68FEIbZ_hIQtaavBRCe2_4SELiLrgUQ3ravBRD14P4S\"},\"screenDensityFloat\":2,\"browserName\":\"Chrome Mobile\",\"browserVersion\":\"115.0.0.0\",\"acceptHeader\":\"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7\",\"deviceExperimentId\":\"ChxOekkyTkRjNE56UXlOVE00TnpFM016SXdNZz09EKzrxqYGGKzrxqYG\",\"screenWidthPoints\":981,\"screenHeightPoints\":830,\"utcOffsetMinutes\":420,\"userInterfaceTheme\":\"USER_INTERFACE_THEME_LIGHT\",\"connectionType\":\"CONN_CELLULAR_4G\",\"memoryTotalKbytes\":\"8000000\",\"mainAppWebInfo\":{\"graftUrl\":\"https://www.youtube.com/@ARDanceGameOfficial/shorts\",\"webDisplayMode\":\"WEB_DISPLAY_MODE_BROWSER\",\"isWebNativeShareAvailable\":true},\"timeZone\":\"Asia/Saigon\"},\"user\":{\"lockedSafetyMode\":false},\"request\":{\"useSsl\":true,\"internalExperimentFlags\":[],\"consistencyTokenJars\":[{\"encryptedTokenJarContents\":\"AKreu9s6iwhuuJkU3P74hVotFw1Nn_kqtGr5nkRbKcrBcHXEqs6ZKaXZGd9iiXUe2PmzkseJIjJMVQsW2sCKfsE1KnufBWQwvIud69NEaCqSTFuF_ywm-Z8\",\"expirationSeconds\":\"600\"}]},\"clickTracking\":{\"clickTrackingParams\":\"CBQQ8eIEIhMIp_-Y047MgAMVUyFgCh2EogOg\"},\"adSignalsInfo\":{\"params\":[{\"key\":\"dt\",\"value\":\"1691465131946\"},{\"key\":\"flash\",\"value\":\"0\"},{\"key\":\"frm\",\"value\":\"0\"},{\"key\":\"u_tz\",\"value\":\"420\"},{\"key\":\"u_his\",\"value\":\"2\"},{\"key\":\"u_h\",\"value\":\"824\"},{\"key\":\"u_w\",\"value\":\"973\"},{\"key\":\"u_ah\",\"value\":\"824\"},{\"key\":\"u_aw\",\"value\":\"973\"},{\"key\":\"u_cd\",\"value\":\"24\"},{\"key\":\"bc\",\"value\":\"31\"},{\"key\":\"bih\",\"value\":\"829\"},{\"key\":\"biw\",\"value\":\"980\"},{\"key\":\"brdim\",\"value\":\"0,0,0,0,973,0,973,824,981,830\"},{\"key\":\"vis\",\"value\":\"1\"},{\"key\":\"wgl\",\"value\":\"true\"},{\"key\":\"ca_type\",\"value\":\"image\"}]}},\"continuation\":\"" +
//		continuation +
//		"\"}"
//	request, _ := http.NewRequest("POST", "https://www.youtube.com/youtubei/v1/browse?key=AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8&prettyPrint=false", bytes.NewBufferString(postBody))
//	request.Header.Set("user-agent", userAgent)
//	request.Header.Set("content-type", "application/json")
//	request.AddCookie(&http.Cookie{Name: "CONSISTENCY", Value: CONSISTENCY})
//	//-H 'content-type: application/json' \
//	//-H 'cookie: CONSISTENCY=AKreu9sM2eZdjrcJGtQUUGha-Ryfr4gsJaoyFKcDnKk05GWyJr4T3YQxIc2HqO6keYgLHUNU-_slncoNCC6-nBWGG9G6Hzh4povCWW_ZqQqsZ4xONeYpA6m8eA' \
//	//-H 'user-agent: Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Mobile Safari/537.36' \
//
//	res, _ := http.DefaultClient.Do(request)
//	resBody, _ := io.ReadAll(res.Body)
//	os.WriteFile(strconv.Itoa(index)+".txt", resBody, 644)
//	responseObject := map[string]interface{}{}
//	json.Unmarshal(resBody, &responseObject)
//	onResponseReceivedActions := responseObject["onResponseReceivedActions"].([]interface{})
//	nextPageQuery := ""
//	for _, actionObject := range onResponseReceivedActions {
//		action := actionObject.(map[string]interface{})
//		appendContinuationItemsAction := action["appendContinuationItemsAction"].(map[string]interface{})
//		continuationItems := appendContinuationItemsAction["continuationItems"].([]interface{})
//		for _, continuationItemObject := range continuationItems {
//			continuationItem := continuationItemObject.(map[string]interface{})
//			if continuationItem["continuationItemRenderer"] != nil {
//				continuationItemRenderer := continuationItem["continuationItemRenderer"].(map[string]interface{})
//				continuationEndpoint := continuationItemRenderer["continuationEndpoint"].(map[string]interface{})
//				continuationCommand := continuationEndpoint["continuationCommand"].(map[string]interface{})
//				nextPageQuery = continuationCommand["token"].(string)
//			}
//		}
//	}
//	return nextPageQuery
//}

func loadVideoFirstPage() []db.MatchAndYoutube {
	err := playwright.Install(&playwright.RunOptions{Verbose: true})
	if err != nil {
		log.Fatalf("could not install driver %v", err)
	}
	pw, err := playwright.Run()
	defer pw.Stop()
	if err != nil {
		log.Fatalf("could not start playwright: %v", err)
	}
	browser, err := pw.Chromium.Launch()
	defer browser.Close()
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	page, err := browser.NewPage(playwright.BrowserNewContextOptions{
		UserAgent: &userAgent,
	})
	defer page.Close()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}
	if _, err = page.Goto(channel); err != nil {
		log.Fatalf("could not goto: %v", err)
	}
	time.Sleep(30 * time.Second)

	script := `
        () => {
            return ytInitialData;
        }
    `
	ytInitialData, err := page.Evaluate(script)
	if err != nil {
		log.Fatalf("Could not evaluate script: %v", err)
	}
	results := ytInitialData.(map[string]interface{})["contents"].(map[string]interface{})
	twoColumnBrowseResultsRenderer := results["twoColumnBrowseResultsRenderer"].(map[string]interface{})
	tabs := twoColumnBrowseResultsRenderer["tabs"].([]interface{})
	matchYoutube := []db.MatchAndYoutube{}
	for _, tabObject := range tabs {
		tab := tabObject.(map[string]interface{})
		if tab["tabRenderer"] == nil {
			continue
		}
		tabRenderer := tab["tabRenderer"].(map[string]interface{})
		if tabRenderer["content"] == nil {
			continue
		}
		content := tabRenderer["content"].(map[string]interface{})
		if content["richGridRenderer"] == nil {
			continue
		}
		richGridRenderer := content["richGridRenderer"].(map[string]interface{})
		if richGridRenderer["contents"] == nil {
			continue
		}
		contents := richGridRenderer["contents"].([]interface{})
		for _, contentObject := range contents {
			content = contentObject.(map[string]interface{})
			if content["richItemRenderer"] != nil {
				richItemRenderer := content["richItemRenderer"].(map[string]interface{})
				content = richItemRenderer["content"].(map[string]interface{})
				reelItemRenderer := content["reelItemRenderer"].(map[string]interface{})
				thumbnail := reelItemRenderer["thumbnail"].(map[string]interface{})
				thumbnails := thumbnail["thumbnails"].([]interface{})
				thumbnailUrl := ""
				for _, thumbnailObject := range thumbnails {
					thumbnail = thumbnailObject.(map[string]interface{})
					thumbnailUrl = thumbnail["url"].(string)
					if len(thumbnailUrl) > 0 {
						thumbnailUrl=strings.Split(thumbnailUrl,"?")[0]
						break
					}
				}
				youtubeId := reelItemRenderer["videoId"].(string)
				headline := reelItemRenderer["headline"].(map[string]interface{})
				title := headline["simpleText"].(string)
				matchId, err := strconv.Atoi(title)
				if err != nil || len(youtubeId) == 0 || len(thumbnailUrl) == 0 {
					continue
				}
				matchYoutube = append(matchYoutube, db.MatchAndYoutube{
					MatchId:   matchId,
					YoutubeId: youtubeId,
					Thumbnail: thumbnailUrl,
				})
			}
			//else if content["continuationItemRenderer"] != nil {
			//	continuationItemRenderer := content["continuationItemRenderer"].(map[string]interface{})
			//	continuationEndpoint := continuationItemRenderer["continuationEndpoint"].(map[string]interface{})
			//	continuationCommand := continuationEndpoint["continuationCommand"].(map[string]interface{})
			//	nextPageQuery = continuationCommand["token"].(string)
			//}
		}
	}
	return matchYoutube
}
func loadAllOldData() []db.MatchAndYoutube {
	allOldData:=[]db.MatchAndYoutube{}
	if len(*oldData)<0 {
		return allOldData
	}
	text,_:=os.ReadFile(*oldData)
	allInfo:=[]map[string]interface{}{}
	json.Unmarshal(text,&allInfo)
	for _, info := range allInfo {
		matchId:=info["db_id"].(float64)
		youtubeId:=info["yt_id"].(string)
		thumbnail:=info["thumbnail"].(string)
		allOldData=append(allOldData,db.MatchAndYoutube{
			MatchId: int(matchId),
			YoutubeId: youtubeId,
			Thumbnail: thumbnail,
		})
	}
	return allOldData
}

func updateYoutubeChannel()  {
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
		fmt.Println("err when connect postgres", err)
	}
	defer db.Close()
	allMatchs := loadVideoFirstPage()
	allMatchs=append(allMatchs,loadAllOldData()...)
	mysqlDb := db.GetDb()
	for _, match := range allMatchs {
		//info := db.MatchResult{Id: match.MatchId}
		//err := mysqlDb.Model(&info).WherePK().Select()
		//if err !=nil && err != pg.ErrNoRows {
		//	println("err when find match with id ",match.MatchId,err.Error())
		//}
		//if info.Id > 0 {
		matchYoutube := db.MatchAndYoutube{}
		err = mysqlDb.Model(&matchYoutube).Where("match_id = ? ", match.MatchId).Select()
		if err != nil && err != pg.ErrNoRows {
			println("err when find match with id ", match.MatchId, err.Error())
			continue
		}
		if matchYoutube.MatchId > 0 {
			continue
		}
		matchYoutube = db.MatchAndYoutube{MatchId: match.MatchId, YoutubeId: match.YoutubeId, Thumbnail: match.Thumbnail}
		_, err = mysqlDb.Model(&matchYoutube).Insert()
		if err != nil {
			println("err when insert match with id ", match.MatchId, "youtube", match.YoutubeId, "thumbnail", match.Thumbnail, err.Error())
		}
	}
}

func execute( ctx context.Context,finish chan int)  {
	for {
		select {
		default:
			updateYoutubeChannel()
			finish <- 1
			return
		case <-ctx.Done():
			fmt.Println("halted operation")
			return
		}
	}
}
func main() {
	flag.Parse()
	ctx,cancelFunc:=context.WithTimeout(context.Background(),5 * time.Minute)
	defer cancelFunc()
	finish := make(chan int)
	go execute(ctx,finish)
	select {
	case <-ctx.Done():
		fmt.Println("Program stopped due to function timeout",time.Now().String())
	case <-finish:
		println("Job complete & finish",time.Now().String())
	}
}

//curl 'https://www.youtube.com/youtubei/v1/browse?key=AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8&prettyPrint=false' \
//-H 'content-type: application/json' \
//-H 'cookie: CONSISTENCY=AKreu9tmKVlSJwfbvSVrjJqFgNeFgSmLmq5LcbJswOyYKp1WpvdZGq-ikhkjRziTcP7i3XRIP_SXtyGUCv306BfIIutcphVocE57_0_oa66X58LyNVVLhf0' \
//-H 'user-agent: Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Mobile Safari/537.36' \
//--data-raw '{"context":{"client":{"hl":"vi","gl":"VN","remoteHost":"","deviceMake":"","deviceModel":"","visitorData":"Cgtmenc3WF9hUlBtbyjBvMKmBg%3D%3D","userAgent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36,gzip(gfe)","clientName":"WEB","clientVersion":"2.20230804.01.00","osName":"Windows","osVersion":"10.0","originalUrl":"https://www.youtube.com/@ARDanceGameOfficial/shorts","platform":"DESKTOP","clientFormFactor":"UNKNOWN_FORM_FACTOR","configInfo":{"appInstallData":"CMG8wqYGEN62rwUQ57qvBRCPo68FEJHPrwUQqcSvBRCst68FEPq-rwUQls6vBRDks_4SENuvrwUQ4LavBRCe2_4SEMyu_hIQ7qKvBRCXz68FEInorgUQpcL-EhDrk64FEJLLrwUQ6sOvBRCCpa8FEMzfrgUQvbauBRDUoa8FEP3nqBgQj8OvBRDyqK8FEJrRrwUQ4tSuBRDcz68FELWmrwUQuIuuBRCG2f4SELTJrwUQjMuvBRCi3v4S"},"browserName":"Chrome","browserVersion":"115.0.0.0","acceptHeader":"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7","deviceExperimentId":"ChxOekkyTkRRNE1ESXdNamcwTXpFM09ESTFNZz09EMG8wqYGGMG8wqYG","screenWidthPoints":981,"screenHeightPoints":830,"screenPixelDensity":2,"screenDensityFloat":2,"utcOffsetMinutes":420,"userInterfaceTheme":"USER_INTERFACE_THEME_LIGHT","connectionType":"CONN_CELLULAR_4G","memoryTotalKbytes":"8000000","mainAppWebInfo":{"graftUrl":"https://www.youtube.com/@ARDanceGameOfficial/shorts","pwaInstallabilityStatus":"PWA_INSTALLABILITY_STATUS_UNKNOWN","webDisplayMode":"WEB_DISPLAY_MODE_BROWSER","isWebNativeShareAvailable":true},"timeZone":"Asia/Saigon"},"user":{"lockedSafetyMode":false},"request":{"useSsl":true,"internalExperimentFlags":[],"consistencyTokenJars":[{"encryptedTokenJarContents":"AKreu9vV4JV9LJXDnnzAOeZbQDpGPPTpH78ZfIMCXzcONqKUyFvKqquYOVIY0jCSUYZp2-zKcIAHJ9J4Yl0SZXhlGY6BvelCZO2q5so-qct3QsZl1_Wt87s","expirationSeconds":"600"}]},"clickTracking":{"clickTrackingParams":"CBgQ8eIEIhMIl8iRloTKgAMVtEAPAh3qdwAX"},"adSignalsInfo":{"params":[{"key":"dt","value":"1691393600562"},{"key":"flash","value":"0"},{"key":"frm","value":"0"},{"key":"u_tz","value":"420"},{"key":"u_his","value":"2"},{"key":"u_h","value":"824"},{"key":"u_w","value":"973"},{"key":"u_ah","value":"824"},{"key":"u_aw","value":"973"},{"key":"u_cd","value":"24"},{"key":"bc","value":"31"},{"key":"bih","value":"829"},{"key":"biw","value":"980"},{"key":"brdim","value":"0,0,0,0,973,0,973,824,981,830"},{"key":"vis","value":"1"},{"key":"wgl","value":"true"},{"key":"ca_type","value":"image"}]}},"continuation":"%3D"}' \
//--compressed
