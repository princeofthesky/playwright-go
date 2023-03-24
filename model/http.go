package model

import "github.com/playwright-community/playwright-go/db"

type EUploadStatus int64

const (
	Exist     EUploadStatus = 0
	Uploading               = 1
)

type HttpResponseCode int64

const (
	HttpSuccess HttpResponseCode = 0
	HttpFail                     = 1
)

type HttpResponse struct {
	Msg  string           `json:"msg"`
	Code HttpResponseCode `json:"code"`
	Data interface{}      `json:"data"`
}

type MetaDataVideo struct {
	Md5Video    string   `json:"md5_video"`
	Md5Cover    string   `json:"md5_cover"`
	Size        int      `json:"size"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	HashTags    []string `json:"hash_tags"`
	Topics      []string `json:"hash_tags"`
}

type UploadStatus struct {
	Md5            string        `json:"md5"`
	UploadedChunks []int         `json:"uploaded_chunks"`
	UploadedCover  bool          `json:"uploaded_cover"`
	Status         EUploadStatus `json:"status"`
}

type ListAudiosResponse struct {
	Audios     []db.Audio `json:"audios"`
	NextOffset string     `json:"next_offset"`
}

type AudioRequirement struct {
	Offset      string `json:"offset"`
	Length      int    `json:"length"`
	Region      int    `json:"region"`
	Themes      []int  `json:"themes"`
	Moods       []int  `json:"moods"`
	Genres      []int  `json:"genres"`
	MinDuration int    `json:"min_duration"`
	MaxDuration int    `json:"max_duration"`
}

type TiktokAudioDuration struct {
	RangeLower int `json:"range_lower"`
	RangeUpper int `json:"range_upper"`
}
type TiktokPostAudioRequest struct {
	//Duration   TiktokAudioDuration `json:"duration"`
	Genres     []string            `json:"genres"`
	Limit      int                 `json:"limit"`
	Moods      []string            `json:"moods"`
	MusicName  string              `json:"music_name"`
	Page       int                 `json:"page"`
	Placements []string            `json:"placements"`
	Region     string              `json:"region"`
	Scenarios  int                 `json:"scenarios"`
	Singer     string              `json:"singer"`
	Themes     []string            `json:"themes"`
}

type TiktokAudioResponse struct {
	Detail           string   `json:"detail"`
	Duration         int      `json:"duration"`
	Genre            string   `json:"genre"`
	IsOnAd           bool     `json:"is_on_ad"`
	Mood             string   `json:"mood"`
	Theme            string   `json:"theme"`
	MusicId          string   `json:"music_id"`
	Singer           string   `json:"singer"`
	Title            string   `json:"title"`
	PlacementAllowed []string `json:"placement_allowed"`
	PosterUrl        string   `json:"poster_url"`
}
type Pagination struct {
	HasMore    bool `json:"has_more"`
	Limit      int  `json:"limit"`
	Page       int  `json:"page"`
	TotalCount int  `json:"total_count"`
}

type TiktokDataResponse struct {
	List       []TiktokAudioResponse `json:"list"`
	Pagination Pagination            `json:"pagination"`
}

type TiktokRequestResponse struct {
	Data      TiktokDataResponse `json:"data"`
	Code      int                `json:"code"`
	Msg       string             `json:"msg"`
	RequestId string             `json:"request_id"`
}
