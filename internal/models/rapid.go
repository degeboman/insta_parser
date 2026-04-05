package models

import (
	"fmt"
	"time"

	"inst_parser/internal/utils"
)

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
	if apiReel.TakenAt > 0 {
		pubTime := time.Unix(apiReel.TakenAt, 0)
		publishDate = utils.PublishDate(pubTime)
	}

	// Создаем строку результата
	return &InstagramReelInfo{
		//todo create func
		URL:         fmt.Sprintf("https://www.instagram.com/reel/%s/", apiReel.Code),
		AccountURL:  accountURL,
		Description: apiReel.Caption.Text,
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
