package models

import (
	"fmt"
	"inst_parser/internal/utils"
	"time"
)

type RapidGetPinsResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Data      Data   `json:"data"`
}

type Data struct {
	Pins []PinterestVideo `json:"pins"`
}

type PinterestVideo struct {
	NodeID                         string         `json:"node_id"`
	PromotedIsLeadAd               bool           `json:"promoted_is_lead_ad"`
	IsEligibleForPDP               bool           `json:"is_eligible_for_pdp"`
	Description                    string         `json:"description"`
	HasRequiredAttributionProvider bool           `json:"has_required_attribution_provider"`
	IsEligibleForRelatedProducts   bool           `json:"is_eligible_for_related_products"`
	IsNative                       bool           `json:"is_native"`
	ImageSignature                 string         `json:"image_signature"`
	Comments                       []interface{}  `json:"comments"` // может быть массив объектов, уточните при необходимости
	ReactionCounts                 map[string]int `json:"reaction_counts"`
	CommentCount                   int            `json:"comment_count"`
	GridTitle                      string         `json:"grid_title"`
	GridDescription                string         `json:"grid_description"`
	IsStaleProduct                 bool           `json:"is_stale_product"`
	Link                           string         `json:"link"`
	IsEligibleForWebCloseup        bool           `json:"is_eligible_for_web_closeup"`
	IsRepin                        bool           `json:"is_repin"`
	ID                             string         `json:"id"`
	Method                         string         `json:"method"`
	IsOOSProduct                   bool           `json:"is_oos_product"`
	IsWhitelistedForTriedIt        bool           `json:"is_whitelisted_for_tried_it"`
	IsQuickPromotable              bool           `json:"is_quick_promotable"`
	ManualInterestTags             []string       `json:"manual_interest_tags"`
	DominantColor                  string         `json:"dominant_color"`
	ShoppingFlags                  []string       `json:"shopping_flags"`
	IsVideo                        bool           `json:"is_video"`
	PromotedIsRemovable            bool           `json:"promoted_is_removable"`
	SeoURL                         string         `json:"seo_url"`
	ViewTags                       []string       `json:"view_tags"`
	IsPlayable                     bool           `json:"is_playable"`
	Type                           string         `json:"type"`
	DescriptionHTML                string         `json:"description_html"`
	DoneByMe                       bool           `json:"done_by_me"`
	Domain                         string         `json:"domain"`
	IsDownstreamPromotion          bool           `json:"is_downstream_promotion"`
	AdditionalHideReasons          []string       `json:"additional_hide_reasons"`
	RepinCount                     int            `json:"repin_count"`
	IsUploaded                     bool           `json:"is_uploaded"`
	Access                         []string       `json:"access"`
	IsPromoted                     bool           `json:"is_promoted"`
	Title                          string         `json:"title"`
	CreatedAt                      string         `json:"created_at"`
	Privacy                        string         `json:"privacy"`
	ShouldOpenInStream             bool           `json:"should_open_in_stream"`
	AltText                        string         `json:"alt_text"`
	SeoTitle                       string         `json:"seo_title"`
}

func (t *PinterestVideo) ToResultRow(url string) (*ResultRowUrl, error) {
	var likes int
	for _, v := range t.ReactionCounts {
		likes += v
	}
	comments := t.CommentCount
	shares := t.RepinCount

	// Форматируем дату публикации
	var publishDate string

	if t.CreatedAt != "" {
		timeCreatedAt, _ := time.Parse(time.RFC1123, t.CreatedAt)
		publishDate = utils.PublishDate(timeCreatedAt)
	}

	// Создаем строку результата
	result := &ResultRowUrl{
		OwnerUrl:    url,
		URL:         fmt.Sprintf("https://www.pinterest.com/pin/%s/", t.ID),
		Description: t.Title,
		Views:       0,
		Likes:       int64(likes),
		Comments:    int64(comments),
		Shares:      int64(shares),
		ER:          utils.GetER(int64(likes), int64(shares), int64(comments), 0),
		Virality:    utils.GetVirality(int64(shares), 0),
		ParsingDate: utils.ParsingDate(),
		PublishDate: publishDate,
	}

	return result, nil
}

func PinterestVideoApiResponseToInterface(data []*PinterestVideo, accountName string) [][]interface{} {
	values := make([][]interface{}, 0, len(data))

	for i := range data {
		if data == nil {
			continue
		}
		result, _ := data[i].ToResultRow(accountName)
		values = append(values, ResultRowToInterface(result))
	}

	return values
}
