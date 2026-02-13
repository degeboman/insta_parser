package models

import (
	"fmt"
	"log"
	"time"

	"inst_parser/internal/utils"

	"github.com/SevereCloud/vksdk/v2/object"
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
	Date        time.Time
}

// VKWallResponse - корневая структура ответа
type VKWallResponse struct {
	Response struct {
		Items []object.WallWallpost `json:"items"`
	} `json:"response"`
}

// WallPost - структура записи на стене
type WallPost struct {
	InnerType     string       `json:"inner_type"`
	Comments      Comments     `json:"comments"`
	Type          string       `json:"type"`
	Attachments   []Attachment `json:"attachments"`
	Date          int64        `json:"date"`
	FromID        int64        `json:"from_id"`
	ID            int64        `json:"id"`
	Likes         Likes        `json:"likes"`
	ReactionSetID string       `json:"reaction_set_id"`
	OwnerID       int64        `json:"owner_id"`
	PostType      string       `json:"post_type"`
	Reposts       Reposts      `json:"reposts"`
	Text          string       `json:"text"`
	Views         Views        `json:"views"`
	TrackCode     string       `json:"track_code"`
}

// Comments - информация о комментариях
type Comments struct {
	Count int `json:"count"`
}

// Attachment - вложение (может быть разных типов)
type Attachment struct {
	Type  string `json:"type"`
	Video *Video `json:"video,omitempty"`
}

// Video - структура видеовложения
type Video struct {
	ResponseType string `json:"response_type"`
	Comments     int    `json:"comments"`
	Date         int64  `json:"date"`
	Description  string `json:"description"`
	Title        string `json:"title"`
}

// Likes - информация о лайках
type Likes struct {
	Count int `json:"count"`
}

// Reposts - информация о репостах
type Reposts struct {
	Count        int `json:"count"`
	UserReposted int `json:"user_reposted"`
}

// Views - информация о просмотрах
type Views struct {
	Count int `json:"count"`
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
	parsingDate := time.Now().In(moscow).Format("02.01.2006 15:04")
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

func ProcessVkGroupClipResponse(apiResponse APIVKClip, url string) *VKClipInfo {
	// Получаем значения с проверкой на нулевые значения
	likes := apiResponse.Likes.Count
	comments := apiResponse.Comments
	shares := apiResponse.Reposts.Count
	views := apiResponse.Views

	// Форматируем дату публикации
	publishDate := ""
	parsingDate := ""
	if apiResponse.Date > 0 {
		// Конвертируем Unix timestamp в time.Time
		pubTime := time.Unix(int64(apiResponse.Date), 0)

		// Устанавливаем временную зону Москвы
		moscow, err := time.LoadLocation("Europe/Moscow")
		if err != nil {
			log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
			moscow = time.Local
		}

		// Форматируем дату в нужный формат
		pubTimeInMoscow := pubTime.In(moscow)
		publishDate = pubTimeInMoscow.Format("02.01.2006 15:04")
		parsingDate = time.Now().In(moscow).Format("02.01.2006 15:04")
	}

	// Создаем строку результата
	return &VKClipInfo{
		GroupUrl:    url,
		Description: apiResponse.Description,
		Views:       views,
		Likes:       likes,
		Comments:    comments,
		Shares:      shares,
		ER:          utils.GetER(int64(likes), int64(shares), int64(comments), int64(views)),
		Virality:    utils.GetVirality(int64(shares), int64(views)),
		ParsingDate: parsingDate,
		PublishDate: publishDate,
		URL:         fmt.Sprintf("https://vk.com/clip%d_%d", apiResponse.OwnerID, apiResponse.ID),
	}
}

func ProcessVKClipInfoToResultRow(url string, result *VKClipInfo) *ResultRow {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
		moscow = time.Local
	}
	pubTimeInMoscow := result.Date.In(moscow)
	publishDate := pubTimeInMoscow.Format("02.01.2006 15:04")
	parsingDate := time.Now().In(moscow).Format("02.01.2006 15:04")

	resultRow := ResultRow{
		URL:         url,
		Description: result.Description,
		Views:       int64(result.Views),
		Likes:       int64(result.Likes),
		Comments:    int64(result.Comments),
		Shares:      int64(result.Shares),
		ER:          utils.GetER(int64(result.Likes), int64(result.Shares), int64(result.Comments), int64(result.Views)),
		Virality:    utils.GetVirality(int64(result.Shares), int64(result.Views)),
		PublishDate: publishDate,
		ParsingDate: parsingDate,
	}

	return &resultRow
}

type GetClipsForGroupAPIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Clips  []APIVKClip `json:"clips"`
		Cursor string      `json:"cursor"`
	} `json:"data"`
}

type APIVKClip struct {
	ID          int    `json:"id"`
	OwnerID     int    `json:"owner_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Duration    int    `json:"duration"`
	Views       int    `json:"views"`
	LocalViews  int    `json:"local_views"`
	Date        int    `json:"date"`
	Reposts     struct {
		Count int `json:"count"`
	} `json:"reposts"`
	Likes struct {
		Count int `json:"count"`
	} `json:"likes"`
	Comments int `json:"comments"`
}

type AccountInfo struct {
	Identification string
	ParsingType    ParsingType
	AccountUrl     string
	Count          int
}
