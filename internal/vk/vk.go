package vk

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"inst_parser/internal/models"

	"github.com/SevereCloud/vksdk/v2/api"
)

type ClipService struct {
	vk *api.VK
}

func NewVKClipService(accessToken string) *ClipService {
	vk := api.NewVK(accessToken)
	return &ClipService{vk: vk}
}

// GetClipInfoByURL получает информацию о клипе по URL
func (s *ClipService) GetClipInfoByURL(ctx context.Context, url string) (*models.ClipInfo, error) {
	ownerID, clipID, err := parseClipURL(url)
	if err != nil {
		return nil, err
	}

	return s.getClipInfo(ctx, ownerID, clipID)
}

// getClipInfo получает информацию о клипе по owner_id и clip_id
func (s *ClipService) getClipInfo(ctx context.Context, ownerID, clipID int) (*models.ClipInfo, error) {
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
		Reposts:  item.Reposts.Count,
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
