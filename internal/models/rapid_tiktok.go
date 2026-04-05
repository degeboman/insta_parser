package models

import (
	"time"

	"inst_parser/internal/utils"
)

type TikTokUserSearchResponse struct {
	Data struct {
		UserList []struct {
			User struct {
				Id       string `json:"id"`
				Nickname string `json:"nickname"`
			} `json:"user"`
		} `json:"user_list"`
	} `json:"data"`
}

type TikTokVideo struct {
	Id           string `json:"id"`
	VideoId      string `json:"video_id"`
	Title        string `json:"title"`
	PlayCount    int64  `json:"play_count"`
	CommentCount int64  `json:"comment_count"`
	DiggCount    int64  `json:"digg_count"`
	ShareCount   int64  `json:"share_count"`
	CreateTime   int64  `json:"create_time"`
}
type TikTokVideoApiResponse struct {
	Data TikTokVideo `json:"data"`
}

type TikTokPostsByUserResponse struct {
	Data struct {
		Videos []TikTokVideo `json:"videos"`
		Cursor string        `json:"cursor"`
	} `json:"data"`
}

func (t *TikTokVideo) ToResultRow(url string) (*ResultRowUrl, error) {
	// Получаем значения с проверкой на нулевые значения
	likes := t.DiggCount
	comments := t.CommentCount
	shares := t.ShareCount
	views := t.PlayCount

	// Форматируем дату публикации
	var publishDate string

	if t.CreateTime > 0 {
		// Конвертируем Unix timestamp в time.Time
		pubTime := time.Unix(t.CreateTime, 0)
		publishDate = utils.PublishDate(pubTime)
	}

	// Создаем строку результата
	result := &ResultRowUrl{
		URL:         url,
		Description: t.Title,
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

func TikTokVideoApiResponseToInterface(data []*TikTokVideo, accountUrl string) [][]interface{} {
	values := make([][]interface{}, 0, len(data))

	for i := range data {
		if data == nil {
			continue
		}
		result, _ := data[i].ToResultRow(accountUrl)
		values = append(values, ResultRowToInterface(result))
	}

	return values
}
