package models

import (
	"fmt"
	"inst_parser/internal/utils"
	"log"
	"time"
)

type ResultRow struct {
	URL         string
	Description string
	Views       int64
	Likes       int64
	Comments    int64
	Shares      int64
	ER          string
	Virality    string
	ParsingDate string
	PublishDate string
}

type UrlInfo struct {
	URL   string
	Count int
}

func EmptyResultRow(url string) *ResultRow {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
		moscow = time.Local
	}
	parsingDate := time.Now().In(moscow).Format("02.01.2006 15:04")
	return &ResultRow{
		URL:         url,
		Description: "",
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

func ProcessInstagramResponse(apiResponse *InstagramAPIResponse, url string) (*ResultRow, error) {
	//Проверяем наличие items
	if len(apiResponse.Data.Items) == 0 {
		return nil, fmt.Errorf("no items found in API response")
	}

	item := apiResponse.Data.Items[0]

	// Получаем значения с проверкой на нулевые значения
	likes := item.LikeCount
	comments := item.CommentCount
	shares := item.ReshareCount
	views := item.IgPlayCount

	// Форматируем дату публикации
	publishDate := ""
	parsingDate := ""
	if item.TakenAt > 0 {
		// Конвертируем Unix timestamp в time.Time
		pubTime := time.Unix(item.TakenAt, 0)

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

	if shares == nil {
		var nilVal int64 = 0
		shares = &nilVal
	}

	// Создаем строку результата
	result := &ResultRow{
		URL:         url,
		Description: item.Caption.Text,
		Views:       views,
		Likes:       likes,
		Comments:    comments,
		Shares:      *shares,
		ER:          utils.GetER(likes, *shares, comments, views),
		Virality:    utils.GetVirality(*shares, views),
		ParsingDate: parsingDate,
		PublishDate: publishDate,
	}

	return result, nil
}

func ResultRowToInterface(results []*ResultRow) [][]interface{} {
	values := make([][]interface{}, 0, len(results))

	for i := range results {
		if results == nil {
			continue
		}
		rowValues := []interface{}{
			results[i].URL,
			results[i].Views,
			results[i].Likes,
			results[i].Comments,
			results[i].Shares,
			results[i].ER,
			results[i].Virality,
			results[i].ParsingDate,
			results[i].PublishDate,
			results[i].Description,
		}
		values = append(values, rowValues)
	}

	return values
}
