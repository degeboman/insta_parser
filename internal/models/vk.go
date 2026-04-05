package models

import (
	"fmt"
	"log"
	"time"

	"inst_parser/internal/constants"
	"inst_parser/internal/utils"
)

type VKClipInfo struct {
	OwnerID     int
	ClipID      int
	Views       int
	Likes       int
	Comments    int
	Shares      int
	GroupUrl    string
	URL         string
	Description string
	ER          string
	Virality    string
	ParsingDate string
	PublishDate string
	DownloadURL string
	Date        time.Time
}

func VKClipsInfoToInterface(clips []*VKClipInfo) [][]interface{} {
	values := make([][]interface{}, 0, len(clips))

	for i := range clips {
		if clips == nil {
			continue
		}
		rowValues := []interface{}{
			clips[i].GroupUrl,
			clips[i].URL,
			clips[i].Views,
			clips[i].Likes,
			clips[i].Comments,
			clips[i].Shares,
			clips[i].ER,
			clips[i].Virality,
			clips[i].ParsingDate,
			clips[i].PublishDate,
			clips[i].Description,
		}
		values = append(values, rowValues)
	}

	return values
}

func EmptyClipInfo(url string) *VKClipInfo {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
		moscow = time.Local
	}
	parsingDate := time.Now().In(moscow).Format(constants.ParsingDateFormat)
	return &VKClipInfo{
		URL:         url,
		Views:       0,
		Likes:       0,
		Comments:    0,
		Shares:      0,
		ER:          "0",
		Virality:    "0",
		ParsingDate: parsingDate,
		PublishDate: "unknown",
	}
}

func ProcessVkGroupClipResponse(apiResponse RapidVkScraperClip, url string) *VKClipInfo {
	// Получаем значения с проверкой на нулевые значения
	likes := apiResponse.Likes.Count
	comments := apiResponse.Comments
	shares := apiResponse.Reposts.Count
	views := apiResponse.Views

	// Форматируем дату публикации
	var publishDate string
	if apiResponse.Date > 0 {
		// Конвертируем Unix timestamp в time.Time
		pubTime := time.Unix(int64(apiResponse.Date), 0)
		publishDate = utils.PublishDate(pubTime)
	}

	// Создаем строку результата
	return &VKClipInfo{
		URL:         fmt.Sprintf("https://vk.com/clip%d_%d", apiResponse.OwnerID, apiResponse.ID),
		GroupUrl:    url,
		Description: apiResponse.Description,
		Views:       views,
		Likes:       likes,
		Comments:    comments,
		Shares:      shares,
		ER:          utils.GetER(int64(likes), int64(shares), int64(comments), int64(views)),
		Virality:    utils.GetVirality(int64(shares), int64(views)),
		ParsingDate: utils.ParsingDate(),
		PublishDate: publishDate,
	}
}

func ProcessVKClipInfoToResultRow(url string, result *VKClipInfo) *ResultRowUrl {
	return &ResultRowUrl{
		URL:         url,
		Description: result.Description,
		Views:       int64(result.Views),
		Likes:       int64(result.Likes),
		Comments:    int64(result.Comments),
		Shares:      int64(result.Shares),
		ER:          utils.GetER(int64(result.Likes), int64(result.Shares), int64(result.Comments), int64(result.Views)),
		Virality:    utils.GetVirality(int64(result.Shares), int64(result.Views)),
		PublishDate: utils.PublishDate(result.Date),
		ParsingDate: utils.ParsingDate(),
	}
}

type AccountInfo struct {
	Identification string
	ParsingType    ParsingType
	AccountUrl     string
	Count          int
}
