package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"inst_parser/internal/models"
)

// VideoInfo содержит полную информацию о видео
type VideoInfo struct {
	ID            string `json:"videoId"`
	Title         string `json:"title"`
	LikeCount     int    `json:"likeCount"`
	ViewCount     string `json:"viewCount"`
	Description   string `json:"description"`
	PublishedDate string `json:"publishedDate"`
	CommentCount  string `json:"commentCount"`
}

// Snippet YouTube API response structures
type Snippet struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PublishedAt string `json:"publishedAt"`
}

type Statistics struct {
	ViewCount    string `json:"viewCount"`
	LikeCount    string `json:"likeCount"`
	CommentCount string `json:"commentCount"`
}

type VideoItem struct {
	ID         string     `json:"id"`
	Snippet    Snippet    `json:"snippet"`
	Statistics Statistics `json:"statistics"`
}

type APIResponse struct {
	Items []VideoItem `json:"items"`
}

// Client клиент для работы с YouTube API
type Client struct {
	logger *slog.Logger
	apiKey string
}

func NewYouTubeClient(logger *slog.Logger, apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		logger: logger,
	}
}

// GetYouTubeVideoStats получает статистику для YouTube видео или Shorts
func (c Client) YoutubeShortInfo(videoID string) (*models.YoutubeShortInfoApiResponse, error) {
	// Формируем URL запроса
	const baseURL = "https://www.googleapis.com/youtube/v3/videos"
	params := url.Values{}
	params.Add("part", "snippet,statistics")
	params.Add("id", videoID)
	params.Add("key", c.apiKey)

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Создаём HTTP запрос с контекстом
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Выполняем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API вернул статус %d: %s", resp.StatusCode, string(body))
	}

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Парсим JSON
	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	// Проверяем, что есть результаты
	if len(apiResponse.Items) == 0 {
		return nil, fmt.Errorf("видео с ID %s не найдено", videoID)
	}

	item := apiResponse.Items[0]
	likes, _ := strconv.Atoi(apiResponse.Items[0].Statistics.LikeCount)

	return &models.YoutubeShortInfoApiResponse{
		ID:            item.ID,
		Title:         item.Snippet.Title,
		LikeCount:     likes,
		ViewCount:     item.Statistics.ViewCount,
		Description:   item.Snippet.Description,
		PublishedDate: item.Snippet.PublishedAt,
		CommentCount:  item.Statistics.CommentCount,
	}, nil
}

// Структуры для каналов
type YouTubeChannel struct {
	ID             string `json:"id"`
	ContentDetails struct {
		RelatedPlaylists struct {
			Uploads string `json:"uploads"`
		} `json:"relatedPlaylists"`
	} `json:"contentDetails"`
}

type ChannelResponse struct {
	Items []YouTubeChannel `json:"items"`
}

// Структуры для плейлистов
type PlaylistItem struct {
	Snippet struct {
		ResourceID struct {
			VideoID string `json:"videoId"`
		} `json:"resourceId"`
		Title       string `json:"title"`
		PublishedAt string `json:"publishedAt"`
	} `json:"snippet"`
}

type PlaylistResponse struct {
	Items         []PlaylistItem `json:"items"`
	NextPageToken string         `json:"nextPageToken"`
}

func (c Client) GetChannelIDByUsername(username string) (string, error) {
	const baseURL = "https://www.googleapis.com/youtube/v3/channels"
	params := url.Values{}
	params.Add("part", "contentDetails")
	params.Add("forHandle", username) // Для новых @handles
	params.Add("key", c.apiKey)

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API вернул статус %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var channelResp ChannelResponse
	if err := json.Unmarshal(body, &channelResp); err != nil {
		return "", err
	}

	if len(channelResp.Items) == 0 {
		return "", fmt.Errorf("канал не найден")
	}

	return channelResp.Items[0].ContentDetails.RelatedPlaylists.Uploads, nil
}

// GetShortsInfoByAccountName получает все видео из плейлиста
func (c *Client) GetShortsInfoByAccountName(accountInfo *models.AccountInfo) ([][]interface{}, error) {
	const baseURL = "https://www.googleapis.com/youtube/v3/playlistItems"
	pageToken := ""
	shortsInfo := make([][]interface{}, 0, accountInfo.Count)

	for len(shortsInfo) < accountInfo.Count {
		params := url.Values{}
		params.Add("part", "snippet")
		params.Add("playlistId", accountInfo.Identification)
		params.Add("maxResults", "50")
		params.Add("key", c.apiKey)

		if pageToken != "" {
			params.Add("pageToken", pageToken)
		}

		requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API вернул статус %d: %s", resp.StatusCode, string(body))
		}

		var playlistResp PlaylistResponse
		if err := json.Unmarshal(body, &playlistResp); err != nil {
			return nil, err
		}

		// Если клипов больше нет, выходим
		if len(playlistResp.Items) == 0 {
			break
		}

		for _, item := range playlistResp.Items {
			if len(shortsInfo) >= accountInfo.Count {
				break
			}

			result, err := c.YoutubeShortInfo(item.Snippet.ResourceID.VideoID)
			if err != nil {
				c.logger.Error("failed to fetch shorts: %w, shortID - %s", err, item.Snippet.ResourceID.VideoID)
				shortsInfo = append(
					shortsInfo,
					models.ResultRowToInterface([]*models.ResultRow{
						models.EmptyResultRow(fmt.Sprintf("https://www.youtube.com/shorts/%s", item.Snippet.ResourceID.VideoID)),
					})...,
				)
				continue
			}

			shortsInfo = append(shortsInfo, result.ToInterface(accountInfo.AccountUrl))
		}

		pageToken = playlistResp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return shortsInfo, nil
}
