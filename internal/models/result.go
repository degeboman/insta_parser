package models

import (
	"fmt"
	"log"
	"time"

	"inst_parser/internal/constants"
	"inst_parser/internal/utils"
)

const defaultReelCount = 12

type UrlInfo struct {
	URL   string
	Count int
}

func DefaultUrlInfo(url string) *UrlInfo {
	return &UrlInfo{
		URL:   url,
		Count: defaultReelCount,
	}
}

func EmptyResultRow(url string) *ResultRowUrl {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
		moscow = time.Local
	}
	parsingDate := time.Now().In(moscow).Format(constants.ParsingDateFormat)

	return &ResultRowUrl{
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

func ProcessInstagramResponse(
	apiResponse *RealTimeScraperMediaInfoResponse,
	url string,
	needFindVideoUrl bool,
) (*ResultRowUrl, error) {
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
	var publishDate string

	if item.TakenAt > 0 {
		// Конвертируем Unix timestamp в time.Time
		pubTime := time.Unix(item.TakenAt, 0)
		publishDate = utils.PublishDate(pubTime)
	}

	if shares == nil {
		var nilVal int64 = 0
		shares = &nilVal
	}

	videoUrls := make([]string, len(item.VideoVersions))
	if needFindVideoUrl {
		videoUrls = findVideoUrls(apiResponse)
	}

	// Создаем строку результата
	result := &ResultRowUrl{
		OwnerUrl:    fmt.Sprintf("https://www.instagram.com/%s/reels/", item.User.Username),
		URL:         url,
		Description: item.Caption.Text,
		Views:       views,
		Likes:       likes,
		Comments:    comments,
		Shares:      *shares,
		ER:          utils.GetER(likes, *shares, comments, views),
		Virality:    utils.GetVirality(*shares, views),
		ParsingDate: utils.ParsingDate(),
		PublishDate: publishDate,
		VideoUrls:   videoUrls,
	}

	return result, nil
}

func findVideoUrls(response *RealTimeScraperMediaInfoResponse) []string {
	if len(response.Data.Items[0].VideoVersions) == 0 {
		return nil
	}

	result := make([]string, len(response.Data.Items[0].VideoVersions))
	for i, item := range response.Data.Items[0].VideoVersions {
		result[i] = item.URL
	}

	return result
}

func ResultRowsToInterface(results []*ResultRowUrl) [][]interface{} {
	values := make([][]interface{}, 0, len(results))

	for i := range results {
		if results == nil {
			continue
		}
		if results[i] == nil {
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
			results[i].OwnerUrl,
			results[i].ErID,
			results[i].INN,
			results[i].AdvertiserName,
		}
		values = append(values, rowValues)
	}

	return values
}

func ResultRowToInterface(result *ResultRowUrl) []interface{} {
	return []interface{}{
		result.URL,
		result.Views,
		result.Likes,
		result.Comments,
		result.Shares,
		result.ER,
		result.Virality,
		result.ParsingDate,
		result.PublishDate,
		result.Description,
	}
}
