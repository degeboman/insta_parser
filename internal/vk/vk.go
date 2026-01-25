package vk

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"time"

	"inst_parser/internal/models"

	"github.com/SevereCloud/vksdk/v2/api"
)

type Service struct {
	logger *slog.Logger
	vk     *api.VK
}

func NewVKService(logger *slog.Logger, accessToken string) *Service {
	vk := api.NewVK(accessToken)
	return &Service{
		vk:     vk,
		logger: logger,
	}
}

// GetClipInfoByURL получает информацию о клипе по URL
func (s *Service) GetClipInfoByURL(ctx context.Context, url string) (*models.ClipInfo, error) {
	ownerID, clipID, err := parseClipURL(url)
	if err != nil {
		return nil, err
	}

	return s.getClipInfo(ctx, ownerID, clipID)
}

func (s *Service) OwnerIDsByGroupsUrls(urls []*models.UrlInfo) ([]*models.GroupInfoPair, error) {
	result := make([]*models.GroupInfoPair, len(urls))

	for i, url := range urls {
		groupID, err := parseGroupURL(url.URL)
		if err != nil {
			s.logger.Error("failed to parse group url",
				slog.String("url", url.URL),
				slog.String("err", err.Error()),
			)
			return nil, err
		}

		// Сначала получаем числовой ID группы, если передан screen_name
		var ownerID int
		if _, err := strconv.Atoi(groupID); err != nil {
			// Это screen_name, нужно получить ID
			params := api.Params{
				"group_id": groupID,
			}

			var groupInfo []struct {
				ID int `json:"id"`
			}

			err := s.vk.RequestUnmarshal("groups.getById", &groupInfo, params)
			if err != nil {
				return nil, fmt.Errorf("failed to get group info: group_url = %s, err = %w", url, err)
			}

			if len(groupInfo) == 0 {
				return nil, fmt.Errorf("group not found: group_url = %s", url)
			}

			ownerID = -groupInfo[0].ID // Для групп ID отрицательный
		} else {
			id, _ := strconv.Atoi(groupID)
			ownerID = -id // Для групп ID отрицательный
		}

		result[i] = &models.GroupInfoPair{
			OwnerID:  strconv.Itoa(ownerID),
			GroupUrl: url.URL,
			Count:    url.Count,
		}
	}

	return result, nil
}

// getClipInfo получает информацию о клипе по owner_id и clip_id
func (s *Service) getClipInfo(ctx context.Context, ownerID, clipID int) (*models.ClipInfo, error) {
	// Используем метод video.get для получения информации о клипе
	// Клипы в VK API обрабатываются как видео
	videos := fmt.Sprintf("%d_%d", ownerID, clipID)

	params := api.Params{
		"videos": videos,
	}

	var response struct {
		Count int `json:"count"`
		Items []struct {
			ID          int    `json:"id"`
			OwnerID     int    `json:"owner_id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Duration    int    `json:"duration"`
			Views       int    `json:"views"`
			Comments    int    `json:"comments"`
			Date        int    `json:"date"`
			Image       []struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"image"`
			Likes struct {
				Count int `json:"count"`
			} `json:"likes"`
			Reposts struct {
				Count int `json:"count"`
			} `json:"reposts"`
		} `json:"items"`
	}

	if err := s.vk.RequestUnmarshal("video.get", &response, params); err != nil {
		return nil, fmt.Errorf("failed to get clip info: %w", err)
	}

	if response.Count == 0 {
		return nil, fmt.Errorf("clip not found")
	}

	item := response.Items[0]

	clipInfo := &models.ClipInfo{
		OwnerID:  item.OwnerID,
		ClipID:   item.ID,
		Views:    item.Views,
		Likes:    item.Likes.Count,
		Comments: item.Comments,
		Shares:   item.Reposts.Count,
		Date:     time.Unix(int64(item.Date), 0),
	}

	return clipInfo, nil
}

// parseClipURL извлекает owner_id и clip_id из URL
func parseClipURL(url string) (int, int, error) {
	re := regexp.MustCompile(`clip(-?\d+)_(\d+)`)
	matches := re.FindStringSubmatch(url)

	if len(matches) != 3 {
		return 0, 0, fmt.Errorf("invalid clip URL format")
	}

	ownerID, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, err
	}

	clipID, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, err
	}

	return ownerID, clipID, nil
}

func parseGroupURL(url string) (string, error) {
	// Извлекаем всё после vk.com/
	re := regexp.MustCompile(`vk\.com/([^/?#]+)`)
	matches := re.FindStringSubmatch(url)

	if len(matches) < 2 || matches[1] == "" {
		return "", fmt.Errorf("invalid group URL format")
	}

	groupID := matches[1]

	return groupID, nil

}
