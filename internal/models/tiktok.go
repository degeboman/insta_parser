package models

import (
	"fmt"
	"inst_parser/internal/utils"
	"log"
	"regexp"
	"time"
)

type TikTokVideoApiResponse struct {
	Data struct {
		Id           string `json:"id"`
		Title        string `json:"title"`
		PlayCount    int64  `json:"play_count"`
		CommentCount int64  `json:"comment_count"`
		DiggCount    int64  `json:"digg_count"`
		ShareCount   int64  `json:"share_count"`
		CreateTime   int64  `json:"create_time"`
	} `json:"data"`
}

func (t *TikTokVideoApiResponse) ToResultRow(url string) (*ResultRow, error) {
	// Получаем значения с проверкой на нулевые значения
	likes := t.Data.DiggCount
	comments := t.Data.CommentCount
	shares := t.Data.ShareCount
	views := t.Data.PlayCount

	// Форматируем дату публикации
	var publishDate string

	if t.Data.CreateTime > 0 {
		// Конвертируем Unix timestamp в time.Time
		pubTime := time.Unix(t.Data.CreateTime, 0)

		// Устанавливаем временную зону Москвы
		moscow, err := time.LoadLocation("Europe/Moscow")
		if err != nil {
			log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
			moscow = time.Local
		}

		// Форматируем дату в нужный формат
		pubTimeInMoscow := pubTime.In(moscow)
		publishDate = pubTimeInMoscow.Format("02.01.2006 15:04")
	}

	// Создаем строку результата
	result := &ResultRow{
		URL:         url,
		Description: t.Data.Title,
		Views:       views,
		Likes:       likes,
		Comments:    comments,
		Shares:      shares,
		ER:          utils.GetER(likes, shares, comments, views),
		Virality:    utils.GetVirality(shares, views),
		ParsingDate: utils.ParsingDate(),
		PublishDate: publishDate,
	}

	return result, nil
}

func ExtractTiktokVideoID(url string) (string, error) {
	re := regexp.MustCompile(`(?:https?://)?(?:www\.)?(?:vm\.)?tiktok\.com/(?:@[^/]+/video/|)([a-zA-Z0-9]+)/?`)

	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return "", fmt.Errorf("не удалось найти идентификатор в URL: %s", url)
	}

	identifier := matches[1] // Идентификатор находится в первой захваченной группе
	return identifier, nil
}
