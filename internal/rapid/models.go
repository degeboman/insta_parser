package rapid

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
