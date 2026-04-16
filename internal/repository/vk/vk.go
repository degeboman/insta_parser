package vk

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"inst_parser/internal/models"

	"github.com/SevereCloud/vksdk/v3/api"
	"github.com/SevereCloud/vksdk/v3/object"
	"golang.org/x/net/html"
)

type Repository struct {
	logger *slog.Logger
	vkApi  *api.VK
}

func NewRepository(logger *slog.Logger, accessToken string) *Repository {
	vk := api.NewVK(accessToken)
	return &Repository{
		logger: logger,
		vkApi:  vk,
	}
}

func (r *Repository) GroupID(groupName string) (string, error) {
	const vkApiMethod = "groups.getById"
	var groupInfo struct {
		Groups []struct {
			ID int `json:"id"`
		} `json:"groups"`
	}

	params := api.Params{
		"group_id": groupName,
	}

	if err := r.vkApi.RequestUnmarshal(vkApiMethod, &groupInfo, params); err != nil {
		return "", fmt.Errorf("failed to get group info: group_id = %s, err = %w", groupName, err)
	}

	if len(groupInfo.Groups) == 0 {
		return "", fmt.Errorf("group not found: group_id = %s", groupName)
	}

	return strconv.Itoa(-groupInfo.Groups[0].ID), nil // Для групп ID отрицательный
}

func (r *Repository) PostInfo(postID string) (*models.VKClipInfo, error) {
	const vkApiMethod = "wall.getById"

	params := api.Params{
		"posts": postID,
	}

	var response models.WallGetByIDResponse

	if err := r.vkApi.RequestUnmarshal(vkApiMethod, &response, params); err != nil {
		fmt.Errorf("failed to get post info: post_id = %s, err = %w", postID, err)
		return nil, err
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("post not found")
	}

	var (
		description     string
		comments, views int
	)

	//todo добавить условие && response.Items[0].Attachments[0].Type == "video"
	if len(response.Items[0].Attachments) > 0 {
		description = response.Items[0].Attachments[0].Video.Description
		views = response.Items[0].Attachments[0].Video.Views
		comments = response.Items[0].Attachments[0].Video.Comments

		if views == 0 {
			views = response.Items[0].Views.Count
		}

		if description == "" {
			description = response.Items[0].Text
		}

		if comments == 0 {
			comments = response.Items[0].Comments.Count
		}

		adsInfo, err := r.getAdvertiserInfo(response.Items[0].AuthorAd.AdvertiserInfoUrl)
		if err != nil {
			r.logger.Warn("failed to get advertiser info",
				slog.String("err", err.Error()),
				slog.String("url", response.Items[0].AuthorAd.AdvertiserInfoUrl),
			)
		}

		return &models.VKClipInfo{
			Description:    description,
			OwnerID:        response.Items[0].OwnerID,
			ClipID:         response.Items[0].ID,
			Views:          views,
			Likes:          response.Items[0].Likes.Count,
			Comments:       comments,
			Shares:         response.Items[0].Reposts.Count,
			Date:           time.Unix(int64(response.Items[0].Date), 0),
			OwnerUrl:       channelUrl(response.Items[0].OwnerID, response.Items[0].ID),
			ErID:           response.Items[0].AuthorAd.AdMarker,
			INN:            adsInfo.INN,
			AdvertiserName: adsInfo.Name,
		}, nil
	}

	adsInfo, err := r.getAdvertiserInfo(response.Items[0].AuthorAd.AdvertiserInfoUrl)
	if err != nil {
		r.logger.Warn("failed to get advertiser info",
			slog.String("err", err.Error()),
			slog.String("url", response.Items[0].AuthorAd.AdvertiserInfoUrl),
		)
	}

	postInfo := &models.VKClipInfo{
		Description:    response.Items[0].Text,
		OwnerID:        response.Items[0].OwnerID,
		ClipID:         response.Items[0].ID,
		Views:          response.Items[0].Views.Count,
		Likes:          response.Items[0].Likes.Count,
		Comments:       response.Items[0].Comments.Count,
		Shares:         response.Items[0].Reposts.Count,
		Date:           time.Unix(int64(response.Items[0].Date), 0),
		OwnerUrl:       channelUrl(response.Items[0].OwnerID, response.Items[0].ID),
		ErID:           response.Items[0].AuthorAd.AdMarker,
		INN:            adsInfo.INN,
		AdvertiserName: adsInfo.Name,
	}

	return postInfo, nil
}

func (r *Repository) ClipInfo(ownerID, clipID int) (*models.VKClipInfo, error) {
	const vkApiMethod = "video.get"
	params := api.Params{
		"videos":   fmt.Sprintf("%d_%d", ownerID, clipID),
		"extended": 1,
	}

	var response models.VideoGetResponse
	if err := r.vkApi.RequestUnmarshal(vkApiMethod, &response, params); err != nil {
		return nil, fmt.Errorf("failed to get clip info: %w", err)
	}

	if response.Count == 0 {
		return nil, fmt.Errorf("clip not found")
	}

	item := response.Items[0]

	var advertiser models.AdvertiserInfo
	if len(item.OrdInfo.Advertisers) > 0 {
		advertiser = item.OrdInfo.Advertisers[0]
	}

	adsInfo, err := r.getAdvertiserInfo(advertiser.Url)
	if err != nil {
		r.logger.Warn("failed to get advertiser info",
			slog.String("err", err.Error()),
			slog.String("url", advertiser.Url),
		)
	}

	clipInfo := &models.VKClipInfo{
		Description:    item.Description,
		OwnerID:        item.OwnerID,
		ClipID:         item.ID,
		Views:          item.Views,
		Likes:          item.Likes.Count,
		Comments:       item.Comments,
		Shares:         item.Reposts.Count,
		Date:           time.Unix(int64(item.Date), 0),
		DownloadURL:    findHighQualityLink(item.Files),
		OwnerUrl:       channelUrl(item.OwnerID, item.UserID),
		PostID:         item.PostID,
		ErID:           advertiser.ErID,
		INN:            adsInfo.INN,
		AdvertiserName: adsInfo.Name,
	}

	return clipInfo, nil
}

func (r *Repository) getAdvertiserInfo(eridURL string) (models.AdvertiserInfoFromUrl, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", eridURL, nil)
	if err != nil {
		return models.AdvertiserInfoFromUrl{}, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return models.AdvertiserInfoFromUrl{}, fmt.Errorf("ошибка запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.AdvertiserInfoFromUrl{}, fmt.Errorf("ошибка чтения тела ответа: %w", err)
	}

	pageHTML := string(body)
	info := models.AdvertiserInfoFromUrl{}

	// Ищем имя рекламодателя в тексте
	info.INN = findTableValue(pageHTML, "ИНН")
	info.Name = findTableValue(pageHTML, "Рекламодатель")

	return info, nil
}

// findTableValue ищет значение в таблице по ключу (следующий <td> после совпадения)
func findTableValue(pageHTML, key string) string {
	doc, err := html.Parse(strings.NewReader(pageHTML))
	if err != nil {
		return ""
	}

	var result string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if result != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "td" {
			text := strings.TrimSpace(nodeText(n))
			if text == key {
				// Берём следующий sibling <td>
				for sib := n.NextSibling; sib != nil; sib = sib.NextSibling {
					if sib.Type == html.ElementNode && sib.Data == "td" {
						result = strings.TrimSpace(nodeText(sib))
						return
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return result
}

// nodeText извлекает текст из узла рекурсивно
func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(nodeText(c))
	}
	return sb.String()
}

func channelUrl(ownerID, userID int) string {
	return fmt.Sprintf("https://vk.com/club%d", ownerID*(-1))
}

func findHighQualityLink(item object.VideoVideoFiles) string {
	if item.Mp4_2160 != "" {
		return item.Mp4_2160
	}

	if item.Mp4_1440 != "" {
		return item.Mp4_1440
	}

	if item.Mp4_1080 != "" {
		return item.Mp4_1080
	}

	if item.Mp4_720 != "" {
		return item.Mp4_720
	}

	if item.Mp4_480 != "" {
		return item.Mp4_480
	}

	if item.Mp4_360 != "" {
		return item.Mp4_360
	}

	if item.Mp4_240 != "" {
		return item.Mp4_240
	}

	return ""
}
