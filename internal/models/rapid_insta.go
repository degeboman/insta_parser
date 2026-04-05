package models

////////////////////////////////////////////////////////////////////////////////////////////////////
////
////     MEDIA INFO
////
/////////////////////////////////////////////////////////////////////////////////////////////////////

type RealTimeScraperMediaInfoResponse struct {
	Data struct {
		Items []RealTimeScraperMediaInfoData `json:"items"`
	} `json:"data"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type RealTimeScraperMediaInfoData struct {
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
	VideoVersions []MediaInfoVideoVersion `json:"video_versions,omitempty"`
	HasAudio      bool                    `json:"has_audio"`

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

// MediaInfoVideoVersion - информация о версиях видео
type MediaInfoVideoVersion struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
	Type   int    `json:"type"`
}

////////////////////////////////////////////////////////////////////////////////////////////////////
////
////     GET USER REELS
////
/////////////////////////////////////////////////////////////////////////////////////////////////////

// GetRapidRealTimeInstagramScraperUserReelsResponse - основной ответ API
type GetRapidRealTimeInstagramScraperUserReelsResponse struct {
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
	TakenAt       int64            `json:"taken_at"`
	PK            string           `json:"pk"`
	ID            string           `json:"id"`
	MediaType     int              `json:"media_type"`
	Code          string           `json:"code"`
	Caption       InstagramCaption `json:"caption,omitempty"`
	PlayCount     int              `json:"play_count"`
	LikeCount     int              `json:"like_count"`
	CommentCount  int              `json:"comment_count"`
	ReshareCount  int              `json:"reshare_count"`
	IgPlayCount   int              `json:"ig_play_count"`
	VideoDuration float64          `json:"video_duration"`
	HasAudio      bool             `json:"has_audio"`
}

type InstagramCaption struct {
	Text string `json:"text"`
}
