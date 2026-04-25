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

type vkClipInfoProvider interface {
	ClipInfo(ownerID, clipID int) (*models.VKClipInfo, error)
}

type Repository struct {
	rapidAPIKey           string
	logger                *slog.Logger
	httpClient            *http.Client
	vkClipInfoProvider    vkClipInfoProvider
	processingInstagramMu sync.Mutex
	processingVkMu        sync.Mutex
	processingTiktokMu    sync.Mutex
}

func NewRepository(
	rapidApiKey string,
	log *slog.Logger,
	vkClipInfoProvider vkClipInfoProvider,
) *Repository {
	return &Repository{
		logger:             log,
		rapidAPIKey:        rapidApiKey,
		vkClipInfoProvider: vkClipInfoProvider,
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
		apiResp, err := r.getRapidRealTimeInstagramScraperUserReels(
			getRapidRealTimeInstagramScraperUserReelsEndpoint(info.Identification, maxID),
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

func (r *Repository) getRapidRealTimeInstagramScraperUserReels(
	endpoint string,
) (*models.GetRapidRealTimeInstagramScraperUserReelsResponse, error) {
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

	var apiResp models.GetRapidRealTimeInstagramScraperUserReelsResponse
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
	r.processingVkMu.Lock()
	defer r.processingVkMu.Unlock()

	clips := make([]*models.VKClipInfo, 0, info.Count)
	var cursor string

	for len(clips) < info.Count {
		apiResp, err := r.getVkClipsInfoForGroup(
			getVkUserClipsEndpoint(info.Identification, cursor),
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

			clipTmp := models.ProcessVkGroupClipResponse(apiClip, info.AccountUrl)

			vkClipInfo, getVkClipInfoErr := r.vkClipInfoProvider.ClipInfo(apiClip.OwnerID, apiClip.ID)
			if getVkClipInfoErr != nil {
				r.logger.Warn("failed to fetch vk clip info",
					slog.Int("clip_id", apiClip.ID),
					slog.Int("owner_id", apiClip.OwnerID),
					slog.String("err", getVkClipInfoErr.Error()),
				)
			}

			if getVkClipInfoErr == nil {
				clipTmp.ErID = vkClipInfo.ErID
				clipTmp.INN = vkClipInfo.INN
				clipTmp.AdvertiserName = vkClipInfo.AdvertiserName
			}

			clips = append(clips, clipTmp)
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

func (r *Repository) getVkClipsInfoForGroup(endpoint string) (*models.RapidVkScraperUserClipsResponse, error) {
	if r.rapidAPIKey == "" {
		return nil, fmt.Errorf("RAPIDAPI_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s%s", constants.RapidVkScraper, endpoint),
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

	var apiResp models.RapidVkScraperUserClipsResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Status != "ok" {
		return nil, fmt.Errorf("API error: %s", apiResp.Message)
	}

	return &apiResp, nil
}

func (r *Repository) GetInstagramReelInfo(reelURL string) (*models.RealTimeScraperMediaInfoResponse, error) {
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
		constants.RapidRealTimeInstagramScraperMediaInfo,
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

	var result models.RealTimeScraperMediaInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v, url: %s", err, reelURL)
	}

	return &result, nil
}

func (r *Repository) GetTiktokVideoInfo(url string) (*models.TikTokVideoApiResponse, error) {
	r.processingTiktokMu.Lock()
	defer r.processingTiktokMu.Unlock()

	if r.rapidAPIKey == "" {
		return nil, fmt.Errorf("RAPIDAPI_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Создаем запрос с Query параметрами
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		constants.RapidTiktokScraper,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Добавляем query параметры
	q := req.URL.Query()
	q.Add("url", url)
	req.URL.RawQuery = q.Encode()

	// Устанавливаем заголовки
	req.Header.Set("x-rapidapi-key", r.rapidAPIKey)
	req.Header.Set("x-rapidapi-host", "tiktok-scraper7.p.rapidapi.com")
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

	var data models.TikTokVideoApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v, url: %s", err, url)
	}

	return &data, nil
}

func (r *Repository) GetTiktokVideoByUserId(info *models.UrlInfo) ([]*models.TikTokVideo, error) {
	r.processingTiktokMu.Lock()
	defer r.processingTiktokMu.Unlock()

	if r.rapidAPIKey == "" {
		return nil, fmt.Errorf("RAPIDAPI_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	videos := make([]*models.TikTokVideo, 0, info.Count)
	var cursor string

	for len(videos) < info.Count {
		// Создаем запрос с Query параметрами
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			constants.RapidTiktokUserPorts,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		// Добавляем query параметры
		q := req.URL.Query()
		q.Add("user_id", info.URL)
		q.Add("count", "30")
		req.URL.RawQuery = q.Encode()

		// Устанавливаем заголовки
		req.Header.Set("x-rapidapi-key", r.rapidAPIKey)
		req.Header.Set("x-rapidapi-host", "tiktok-scraper7.p.rapidapi.com")
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

		var data models.TikTokPostsByUserResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("tiktok failed to parse JSON: %v, url: %s", err, info.URL)
		}

		if len(data.Data.Videos) == 0 {
			break
		}

		for _, video := range data.Data.Videos {
			if len(videos) >= info.Count {
				break
			}

			videos = append(videos, &video)
		}

		// Обновляем курсор для следующего запроса
		cursor = data.Data.Cursor
		if cursor == "" {
			break
		}

		// Задержка между запросами
		time.Sleep(500 * time.Millisecond)
	}

	return videos, nil
}

func (r *Repository) GetTiktokAccountIdByUsername(username string) (string, error) {
	r.processingTiktokMu.Lock()
	defer r.processingTiktokMu.Unlock()

	if r.rapidAPIKey == "" {
		return "", fmt.Errorf("RAPIDAPI_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Создаем запрос с Query параметрами
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		constants.RapidTiktokUserSearch,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Добавляем query параметры
	q := req.URL.Query()
	q.Add("keywords", username)
	q.Add("count", "5")
	req.URL.RawQuery = q.Encode()

	// Устанавливаем заголовки
	req.Header.Set("x-rapidapi-key", r.rapidAPIKey)
	req.Header.Set("x-rapidapi-host", "tiktok-scraper7.p.rapidapi.com")
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to do request: %v", err)
	}
	defer resp.Body.Close()

	// Обрабатываем ответ
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("err http code: %d, body: %s", resp.StatusCode, string(body))
	}

	var data models.TikTokUserSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %v, url: %s", err, username)
	}

	for _, account := range data.Data.UserList {
		if account.User.Nickname == username {
			return account.User.Id, nil
		}
	}

	return "", fmt.Errorf("failed to find account by username: %s", username)
}

func getVkUserClipsEndpoint(identification, cursor string) string {
	endpoint := fmt.Sprintf("/users/clips?owner_id=chplk:%s", identification)
	if cursor != "" {
		endpoint += fmt.Sprintf("&cursor=%s", cursor)
	}

	return endpoint
}

func getRapidRealTimeInstagramScraperUserReelsEndpoint(username, maxID string) string {
	params := url.Values{}
	params.Add("username_or_id", username)

	if maxID != "" {
		params.Add("max_id", maxID)
	}

	return fmt.Sprintf("%s?%s", constants.RapidRealTimeInstagramScraperUserReels, params.Encode())
}
