package models

import (
	"fmt"
	"log"
	"time"

	"inst_parser/internal/utils"
)

type (
	InstagramItem struct {
		Code      string `json:"code"`
		ID        string `json:"id"`
		TakenAt   int64  `json:"taken_at"`
		MediaType int    `json:"media_type"`

		// Статистика
		LikeCount    int64 `json:"like_count"`
		CommentCount int64 `json:"comment_count"`
		// ReshareCount может отсутствовать, используем указатель
		ReshareCount *int64 `json:"reshare_count,omitempty"`
		IgPlayCount  int64  `json:"ig_play_count,omitempty"` // Просмотры для видео

		// Информация о видео
		VideoVersions []VideoVersion `json:"video_versions,omitempty"`
		HasAudio      bool           `json:"has_audio"`

		// Информация о картинке
		ImageVersions2 struct {
			Candidates []ImageCandidate `json:"candidates"`
		} `json:"image_versions2,omitempty"`

		// Информация о пользователе
		User struct {
			Username   string `json:"username"`
			FullName   string `json:"full_name"`
			IsVerified bool   `json:"is_verified"`
		} `json:"user"`

		// Подпись
		Caption struct {
			Text string `json:"text"`
			Pk   string `json:"pk"`
		} `json:"caption,omitempty"`

		// Метаданные клипов
		ClipsMetadata struct {
			OriginalSoundInfo struct {
				OriginalAudioTitle string `json:"original_audio_title"`
			} `json:"original_sound_info"`
		} `json:"clips_metadata"`

		// Дополнительные поля
		ViewCount      *int64 `json:"view_count,omitempty"`
		FbLikeCount    int64  `json:"fb_like_count,omitempty"`
		ProductType    string `json:"product_type"`
		OriginalHeight int    `json:"original_height"`
		OriginalWidth  int    `json:"original_width"`
		DisplayURI     string `json:"display_uri"`

		// Флаги
		LikeAndViewCountsDisabled bool `json:"like_and_view_counts_disabled"`
		CanViewerReshare          bool `json:"can_viewer_reshare"`
	}

	// VideoVersion - информация о версиях видео
	VideoVersion struct {
		Width  int    `json:"width"`
		Height int    `json:"height"`
		URL    string `json:"url"`
		Type   int    `json:"type"`
	}

	// ImageCandidate - кандидаты изображений
	ImageCandidate struct {
		URL    string `json:"url"`
		Height int    `json:"height"`
		Width  int    `json:"width"`
	}
)

type InstagramAPIResponse struct {
	Data struct {
		Items []InstagramItem `json:"items"`
	} `json:"data"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type InstagramAPIErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// GetInstagramReelsAPIResponse - основной ответ API
type GetInstagramReelsAPIResponse struct {
	Status  string             `json:"status"`
	Message string             `json:"message,omitempty"`
	Data    InstagramReelsData `json:"data"`
}

// InstagramReelsData - данные с reels
type InstagramReelsData struct {
	Items      []InstagramReelItem `json:"items"`
	PagingInfo InstagramPagingInfo `json:"paging_info"`
}

// InstagramPagingInfo - информация о пагинации
type InstagramPagingInfo struct {
	MaxID         string `json:"max_id"`
	MoreAvailable bool   `json:"more_available"`
}

// InstagramReelItem - информация об одном reel
type InstagramReelItem struct {
	Media InstagramMedia `json:"media"`
}

type InstagramMedia struct {
	TakenAt       int64             `json:"taken_at"`
	PK            string            `json:"pk"`
	ID            string            `json:"id"`
	MediaType     int               `json:"media_type"`
	Code          string            `json:"code"`
	Caption       *InstagramCaption `json:"caption"`
	PlayCount     int               `json:"play_count"`
	LikeCount     int               `json:"like_count"`
	CommentCount  int               `json:"comment_count"`
	ReshareCount  int               `json:"reshare_count"`
	IgPlayCount   int               `json:"ig_play_count"`
	VideoDuration float64           `json:"video_duration"`
	HasAudio      bool              `json:"has_audio"`
}

type InstagramCaption struct {
	Text string `json:"text"`
}

// InstagramReelInfo - обработанная информация о reel
type InstagramReelInfo struct {
	AccountURL  string
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

// ProcessInstagramReelResponse - конвертирует API response в вашу модель
func ProcessInstagramReelResponse(apiReel *InstagramMedia, accountURL string) *InstagramReelInfo {
	// Получаем значения с проверкой на нулевые значения
	likes := apiReel.LikeCount
	comments := apiReel.CommentCount
	shares := apiReel.ReshareCount
	views := apiReel.IgPlayCount

	// Форматируем дату публикации
	publishDate := ""
	parsingDate := ""
	if apiReel.TakenAt > 0 {
		// Конвертируем Unix timestamp в time.Time
		pubTime := time.Unix(int64(apiReel.TakenAt), 0)

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
	return &InstagramReelInfo{
		URL:         fmt.Sprintf("https://www.instagram.com/reel/%s/", apiReel.Code),
		AccountURL:  accountURL,
		Description: apiReel.Caption.Text,
		Views:       int(views),
		Likes:       likes,
		Comments:    comments,
		Shares:      shares,
		ER:          utils.GetER(int64(likes), int64(shares), int64(comments), int64(views)),
		Virality:    utils.GetVirality(int64(shares), int64(views)),
		ParsingDate: parsingDate,
		PublishDate: publishDate,
	}
}

// EmptyReelInfo - возвращает пустую информацию при ошибке
func EmptyReelInfo(accountURL string) *InstagramReelInfo {
	return &InstagramReelInfo{
		AccountURL: accountURL,
	}
}

func InstagramReelInfoToInterface(data []*InstagramReelInfo) [][]interface{} {
	values := make([][]interface{}, 0, len(data))

	for i := range data {
		if data == nil {
			continue
		}
		rowValues := []interface{}{
			data[i].AccountURL,
			data[i].URL,
			data[i].Views,
			data[i].Likes,
			data[i].Comments,
			data[i].Shares,
			data[i].ER,
			data[i].Virality,
			data[i].ParsingDate,
			data[i].PublishDate,
			data[i].Description,
		}
		values = append(values, rowValues)
	}

	return values
}
