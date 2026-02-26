package models

import (
	"fmt"
	"inst_parser/internal/utils"
	"log"
	"strconv"
)

type ClipMoneyResultRow struct {
	AccountUrl  string `json:"accountUrl"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Views       int64  `json:"views"`
	Likes       int64  `json:"likes"`
	Comments    int64  `json:"comments"`
	Shares      int64  `json:"shares"`
	ER          string `json:"er"`
	Virality    string `json:"virality"`
	ParsingDate string `json:"parsing_date"`
	PublishDate string `json:"publish_date"`
}

func ClipMoneyResultRowFromVkClipInfo(data []*VKClipInfo, accountUrl string) []*ClipMoneyResultRow {
	result := make([]*ClipMoneyResultRow, len(data))

	for i := range data {
		result[i] = &ClipMoneyResultRow{
			AccountUrl:  accountUrl,
			URL:         data[i].URL,
			Description: data[i].Description,
			Views:       int64(data[i].Views),
			Likes:       int64(data[i].Likes),
			Comments:    int64(data[i].Comments),
			Shares:      int64(data[i].Shares),
			ER:          data[i].ER,
			Virality:    data[i].Virality,
			ParsingDate: data[i].ParsingDate,
			PublishDate: data[i].PublishDate,
		}
	}

	return result
}

func ClipMoneyResultRowFromInstagramReelInfo(data []*InstagramReelInfo, accountUrl string) []*ClipMoneyResultRow {
	result := make([]*ClipMoneyResultRow, len(data))

	for i := range data {
		result[i] = &ClipMoneyResultRow{
			AccountUrl:  accountUrl,
			URL:         data[i].URL,
			Description: data[i].Description,
			Views:       int64(data[i].Views),
			Likes:       int64(data[i].Likes),
			Comments:    int64(data[i].Comments),
			Shares:      int64(data[i].Shares),
			ER:          data[i].ER,
			Virality:    data[i].Virality,
			ParsingDate: data[i].ParsingDate,
			PublishDate: data[i].PublishDate,
		}
	}

	return result
}

func ClipMoneyResultRowFromYoutubeShortInfoApiResponse(data []*YoutubeShortInfoApiResponse, accountUrl string) []*ClipMoneyResultRow {
	result := make([]*ClipMoneyResultRow, len(data))

	for i := range data {
		likes := data[i].LikeCount
		comments, err := strconv.Atoi(data[i].CommentCount)
		if err != nil {
			log.Println("Error converting comments count to int")
			comments = 0
		}
		views, err := strconv.Atoi(data[i].ViewCount)
		if err != nil {
			log.Println("Error converting views count to int")
			views = 0
		}

		var publishDate string
		if data[i].PublishedDate == "" {
			publishDate = data[i].PublishDate
		} else if data[i].PublishDate == "" {
			publishDate = data[i].PublishedDate
		}

		result[i] = &ClipMoneyResultRow{
			AccountUrl:  accountUrl,
			URL:         fmt.Sprintf("https://www.youtube.com/shorts/%s", data[i].ID),
			Description: fmt.Sprintf("%s.%s", data[i].Title, data[i].Description),
			Views:       int64(views),
			Likes:       int64(likes),
			Comments:    int64(comments),
			Shares:      0,
			ER:          utils.GetER(int64(likes), 0, int64(comments), int64(views)),
			Virality:    utils.GetVirality(0, int64(views)),
			ParsingDate: utils.ParsingDate(),
			PublishDate: publishDate,
		}
	}

	return result
}
