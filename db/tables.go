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
