package db

type Audio struct {
	Id          int    `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Duration    int    `json:"duration"`
	HashTags    string `json:"hash_tags"`
	Cover       string `json:"cover"`
	TiktokUrl   string `json:"tiktok_url"`
	TiktokId    string `json:"tiktok_id"`
	Url         string `json:"url"`
	CrawledTime int64  `json:"crawled_time"`
}

type Theme struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
}

type Genre struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
}

type Mood struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
}

type Region struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
	Code  string `json:"code"`
}

type RegionTrendingAudio struct {
	AudioId     int   `json:"audio_id"`
	UpdatedTime int64 `json:"updated_time"`
	ThemeId     int   `json:"theme_id"`
	GenreId     int   `json:"genre_id"`
	MoodId      int   `json:"mood_id"`
	Duration    int   `json:"duration"`
}

type User struct {
	Id int
	Name string
	Avatar string
	Gender string
	Birthday string
	Username string
	Password string
	Provider string
}

type UserAndDevice struct {
	UserId string
	DeviceId string
	LastAccess int64
}

type MatchResult struct {
	Id           int     `json:"id"`
	UserId       int     `json:"user_id"`
	AudioId      int     `json:"audio_id"`
	Video        string  `json:"video"`
	Cover        string  `json:"cover"`
	VideoMd5     string  `json:"video_md5"`
	CoverMd5     string  `json:"cover_md5"`
	Score        int     `json:"score"`
	Accuracy     float32 `json:"accuracy"`
	PoseTime     float32 `json:"pose_time"`
	PlayInfo     string  `json:"play_info"`
	ReceivedTime int64   `json:"received_time"`
}

type ListMatch struct {
	MatchId int
	UserId int
	AudioId int
	Score int
	Time int
}


type MatchAndYoutube struct {
	MatchId   int    `json:"match_id"`
	YoutubeId string `json:"youtube_id"`
	Thumbnail string `json:"thumbnail"`
}
