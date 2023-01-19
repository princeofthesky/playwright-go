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
	Audios     []db.Audio `json:"videos"`
	NextOffset int64      `json:"next_offset"`
}
