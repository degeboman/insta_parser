package rapid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"inst_parser/internal/constants"
	"inst_parser/internal/models"
)

type Repository struct {
	rapidAPIKey           string
	logger                *slog.Logger
	httpClient            *http.Client
	processingInstagramMu sync.Mutex
}

func NewRepository(
	rapidApiKey string,
	log *slog.Logger,
) *Repository {
	return &Repository{
		logger:      log,
		rapidAPIKey: rapidApiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (r *Repository) GetInstagramReelsInfoForAccount(info *models.AccountInfo) ([]*models.InstagramReelInfo, error) {
	r.processingInstagramMu.Lock()
	defer r.processingInstagramMu.Unlock()

	reels := make([]*models.InstagramReelInfo, 0, info.Count)
	var maxID string

	for len(reels) < info.Count {
		apiResp, err := r.getReelsForUser(
			getInstagramReelsEndpoint(info.Identification, maxID),
		)
		if err != nil {
			return []*models.InstagramReelInfo{models.EmptyReelInfo(info.AccountUrl)},
				fmt.Errorf("failed to fetch reels: %w", err)
		}

		// Если reels больше нет, выходим
		if len(apiResp.Data.Items) == 0 {
			break
		}

		// Конвертируем и добавляем reels
		for _, apiReel := range apiResp.Data.Items {
			if len(reels) >= info.Count {
				break
			}

			reels = append(reels, models.ProcessInstagramReelResponse(&apiReel.Media, info.AccountUrl))
		}

		// Обновляем max_id для следующего запроса
		maxID = apiResp.Data.PagingInfo.MaxID
		if !apiResp.Data.PagingInfo.MoreAvailable {
			break
		}

		// Задержка между запросами
		time.Sleep(500 * time.Millisecond)
	}

	return reels, nil
}

func (r *Repository) getReelsForUser(endpoint string) (*models.GetInstagramReelsAPIResponse, error) {
	if r.rapidAPIKey == "" {
		return nil, fmt.Errorf("RAPIDAPI_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Устанавливаем заголовки
	req.Header.Set("x-rapidapi-key", r.rapidAPIKey)
	req.Header.Set("x-rapidapi-host", "real-time-instagram-scraper-api1.p.rapidapi.com")
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp models.GetInstagramReelsAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Проверяем статус ответа (адаптируйте под вашу структуру)
	if apiResp.Status != "ok" {
		return nil, fmt.Errorf("API error: %s", apiResp.Message)
	}

	return &apiResp, nil
}

func (r *Repository) GetVKClipsInfoForGroup(info *models.AccountInfo) ([]*models.VKClipInfo, error) {
	r.processingInstagramMu.Lock()
	defer r.processingInstagramMu.Unlock()

	clips := make([]*models.VKClipInfo, 0, info.Count)
	var cursor string

	for len(clips) < info.Count {
		apiResp, err := r.getClipsInfoForGroup(
			getUserClipsEndpoint(info.Identification, cursor),
		)
		if err != nil {
			return []*models.VKClipInfo{models.EmptyClipInfo(info.AccountUrl)},
				fmt.Errorf("failed to fetch clips: %w", err)
		}

		// Если клипов больше нет, выходим
		if len(apiResp.Data.Clips) == 0 {
			break
		}

		// Конвертируем и добавляем клипы
		for _, apiClip := range apiResp.Data.Clips {
			if len(clips) >= info.Count {
				break
			}

			clips = append(clips, models.ProcessVkGroupClipResponse(apiClip, info.AccountUrl))
		}

		// Обновляем курсор для следующего запроса
		cursor = apiResp.Data.Cursor
		if cursor == "" {
			break
		}

		// Задержка между запросами
		time.Sleep(500 * time.Millisecond)
	}

	return clips, nil
}

func (r *Repository) getClipsInfoForGroup(endpoint string) (*models.GetClipsForGroupAPIResponse, error) {
	if r.rapidAPIKey == "" {
		return nil, fmt.Errorf("RAPIDAPI_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s%s", constants.RapidVkGetClipsInfoForGroup, endpoint),
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Устанавливаем заголовки
	req.Header.Set("x-rapidapi-key", r.rapidAPIKey)
	req.Header.Set("x-rapidapi-host", "vk-scraper.p.rapidapi.com")
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp models.GetClipsForGroupAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Status != "ok" {
		return nil, fmt.Errorf("API error: %s", apiResp.Message)
	}

	return &apiResp, nil
}

func (r *Repository) GetInstagramReelInfo(reelURL string) (*models.InstagramAPIResponse, error) {
	r.processingInstagramMu.Lock()
	defer r.processingInstagramMu.Unlock()

	if r.rapidAPIKey == "" {
		return nil, fmt.Errorf("RAPIDAPI_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Создаем запрос с Query параметрами
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		constants.RapidInstagramGetReelInfo,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Добавляем query параметры
	q := req.URL.Query()
	q.Add("code_or_id_or_url", reelURL)
	req.URL.RawQuery = q.Encode()

	// Устанавливаем заголовки
	req.Header.Set("x-rapidapi-key", r.rapidAPIKey)
	req.Header.Set("x-rapidapi-host", "real-time-instagram-scraper-api1.p.rapidapi.com")
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %v", err)
	}
	defer resp.Body.Close()

	// Обрабатываем ответ
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("err http code: %d, body: %s", resp.StatusCode, string(body))
	}

	var data models.InstagramAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v, url: %s", err, reelURL)
	}

	return &data, nil
}

func (r *Repository) YoutubeShortInfo(shortID string) (*models.YoutubeShortInfoApiResponse, error) {
	if r.rapidAPIKey == "" {
		return nil, fmt.Errorf("RAPIDAPI_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Создаем запрос с Query параметрами
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		constants.RapidYoutubeShortInfo,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Добавляем query параметры
	q := req.URL.Query()
	q.Add("id", shortID)
	req.URL.RawQuery = q.Encode()

	// Устанавливаем заголовки
	req.Header.Set("x-rapidapi-key", r.rapidAPIKey)
	req.Header.Set("x-rapidapi-host", "yt-api.p.rapidapi.com")
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %v", err)
	}
	defer resp.Body.Close()

	// Обрабатываем ответ
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("err http code: %d, body: %s", resp.StatusCode, string(body))
	}

	var data models.YoutubeShortInfoApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return &data, nil
}

func (r *Repository) GetShortsInfoByAccountName(accountInfo *models.AccountInfo) ([][]interface{}, error) {
	var (
		shortsID          []string
		continuationToken string
		err               error
	)
	shortsInfo := make([][]interface{}, 0, accountInfo.Count)

	for len(shortsInfo) < accountInfo.Count {
		shortsID, continuationToken, err = r.getShortsGroupByAccountName(accountInfo.Identification, continuationToken)
		if err != nil {
			return nil,
				fmt.Errorf("failed to fetch shorts: %w", err)
		}

		// Если клипов больше нет, выходим
		if len(shortsID) == 0 {
			break
		}

		// Конвертируем и добавляем клипы
		for _, shortID := range shortsID {
			if len(shortsInfo) >= accountInfo.Count {
				break
			}

			resp, err := r.YoutubeShortInfo(shortID)
			if err != nil {
				r.logger.Error("failed to fetch shorts: %w, shortID - %s", err, shortID)
				shortsInfo = append(
					shortsInfo,
					models.ResultRowToInterface([]*models.ResultRow{
						models.EmptyResultRow(fmt.Sprintf("https://www.youtube.com/shorts/%s", shortID)),
					})...,
				)
				continue
			}

			shortsInfo = append(shortsInfo, resp.ToInterface(accountInfo.AccountUrl))
		}

		if continuationToken == "" {
			break
		}

		// Задержка между запросами
		time.Sleep(500 * time.Millisecond)
	}

	return shortsInfo, nil
}

func (r *Repository) getShortsGroupByAccountName(accountName, continuationToken string) ([]string, string, error) {
	if r.rapidAPIKey == "" {
		return nil, "", fmt.Errorf("RAPIDAPI_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Создаем запрос с Query параметрами
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		constants.RapidYoutubeChannelShorts,
		nil,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %v", err)
	}

	// Добавляем query параметры
	q := req.URL.Query()
	q.Add("forUsername", accountName)

	if continuationToken != "" {
		q.Add("token", continuationToken)
	}

	req.URL.RawQuery = q.Encode()

	// Устанавливаем заголовки
	req.Header.Set("x-rapidapi-key", r.rapidAPIKey)
	req.Header.Set("x-rapidapi-host", "yt-api.p.rapidapi.com")
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to do request: %v", err)
	}
	defer resp.Body.Close()

	// Обрабатываем ответ
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("err http code: %d, body: %s", resp.StatusCode, string(body))
	}

	var data models.YoutubeChannelShortsApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, "", fmt.Errorf("failed to parse JSON: %v", err)
	}

	var shortsID []string

	for _, item := range data.Data {
		if item.Type == "shorts" {
			shortsID = append(shortsID, item.VideoId)
		}
	}

	return shortsID, data.Continuation, nil
}

func getUserClipsEndpoint(identification, cursor string) string {
	endpoint := fmt.Sprintf("/users/clips?owner_id=chplk:%s", identification)
	if cursor != "" {
		endpoint += fmt.Sprintf("&cursor=%s", cursor)
	}

	return endpoint
}

func getInstagramReelsEndpoint(username string, maxID string) string {
	params := url.Values{}
	params.Add("username_or_id", username)

	if maxID != "" {
		params.Add("max_id", maxID)
	}

	return fmt.Sprintf("%s?%s", constants.RapidInstagramGetReelsInfoForAccount, params.Encode())
}
