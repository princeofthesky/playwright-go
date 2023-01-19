package db

type Audio struct {
	Id          int    `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Duration    string `json:"duration"`
	HashTags    string `json:"hash_tags"`
	Cover       string `json:"cover"`
	TiktokUrl   string `json:"tiktok_url"`
	Url         string `json:"url"`
	CrawledTime int64  `json:"crawled_time"`
}

type NewAudio struct {
	AudioId     int
	CrawledTime int64
}
