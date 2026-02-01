package rapid

import (
	"context"
	"encoding/json"
	"fmt"
	"inst_parser/internal/utils"
	"io"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"inst_parser/internal/google_sheet"
	"inst_parser/internal/models"
)

type (
	vkParser interface {
		GetClipInfoByURL(ctx context.Context, url string) (*models.VKClipInfo, error)
	}

	trackerService interface {
		EnsureProgressSheet(spreadsheetID string) error
		StartParsing(spreadsheetID string, totalURLs int) (int, error)
		UpdateProgress(spreadsheetID string, row, progress int) error
		FinishParsing(spreadsheetID string, row int) error
	}
)

type Service struct {
	rapidAPIKey           string
	logger                *slog.Logger
	httpClient            *http.Client
	sheetsService         *google_sheet.Service
	processingInstagramMu sync.Mutex
	processingVkMu        sync.Mutex
	vkParser              vkParser
	trackerService        trackerService
}

func NewService(
	rapidApiKey string,
	log *slog.Logger,
	sheetsService *google_sheet.Service,
	vkParser vkParser,
	trackerService trackerService,
) *Service {
	return &Service{
		logger:        log,
		rapidAPIKey:   rapidApiKey,
		sheetsService: sheetsService,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		vkParser:       vkParser,
		trackerService: trackerService,
	}
}

func (s *Service) ParseUrl(spreadsheetID string, reelUrl []*models.UrlInfo) []*models.ResultRow {
	s.processingInstagramMu.Lock()
	defer s.processingInstagramMu.Unlock()

	const batchSize = 50
	results := make([]*models.ResultRow, 0, len(reelUrl))

	if err := s.trackerService.EnsureProgressSheet(spreadsheetID); err != nil {
		s.logger.Error("failed to ensure progress sheet",
			slog.String("spreadsheet_id", spreadsheetID),
		)
	}

	progressRow, errStartParsing := s.trackerService.StartParsing(spreadsheetID, len(reelUrl))
	if errStartParsing != nil {
		s.logger.Error("Error starting progress tracking", slog.String("err", errStartParsing.Error()))
	}

	processedCount := 0

	defer func() {
		if err := s.trackerService.FinishParsing(spreadsheetID, progressRow); err != nil {
			s.logger.Error("Error finishing progress tracking", slog.String("err", err.Error()))
		}
	}()

	for i := 0; i < len(reelUrl); i += batchSize {
		end := i + batchSize
		if end > len(reelUrl) {
			end = len(reelUrl)
		}

		batch := reelUrl[i:end]
		batchResults := s.processBatch(batch)
		results = append(results, batchResults...)

		processedCount += len(batch)

		if err := s.trackerService.UpdateProgress(spreadsheetID, progressRow, processedCount); err != nil {
			s.logger.Error("Error updating progress", slog.String("err", err.Error()))
		}
	}

	return results
}

func (s *Service) processBatch(urls []*models.UrlInfo) []*models.ResultRow {
	results := make([]*models.ResultRow, 0, len(urls))

	for _, url := range urls {
		var resultRow *models.ResultRow
		switch models.ParsingTypeByUrl(url.URL) {
		case models.InstagramParsingType:
			time.Sleep(550 * time.Millisecond)
			resultRow = s.parseInstagram(url.URL)
		case models.VKParsingType:
			time.Sleep(250 * time.Millisecond)
			resultRow = s.parseVK(url.URL)
		default:
			s.logger.Warn("Unsupported URL type", slog.String("url", url.URL))
			continue
		}

		if resultRow == nil {
			continue
		}
		results = append(results, resultRow)
	}

	return results
}

func (s *Service) parseInstagram(url string) *models.ResultRow {
	data, err := s.fetchInstagramDataSafe(url)
	if err != nil {
		s.logger.Warn("Error fetching instagram data",
			slog.String("url", url),
			slog.String("err", err.Error()),
		)

		return models.EmptyResultRow(url)
	}

	resultRow, err := processInstagramResponse(data, url)
	if err != nil {
		s.logger.Error("Error processing instagram response", slog.String("err", err.Error()))
		return nil
	}

	return resultRow
}

func (s *Service) parseVK(url string) *models.ResultRow {
	result, err := s.vkParser.GetClipInfoByURL(context.Background(), url)
	if err != nil {
		s.logger.Warn("Error getting clip info", slog.String("err", err.Error()))
		return nil
	}

	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
		moscow = time.Local
	}
	pubTimeInMoscow := result.Date.In(moscow)
	publishDate := pubTimeInMoscow.Format("02.01.2006 15:04")
	parsingDate := time.Now().In(moscow).Format("02.01.2006 15:04")

	resultRow := models.ResultRow{
		URL:         url,
		Description: result.Description,
		Views:       int64(result.Views),
		Likes:       int64(result.Likes),
		Comments:    int64(result.Comments),
		Shares:      int64(result.Shares),
		ER:          utils.GetER(int64(result.Likes), int64(result.Shares), int64(result.Comments), int64(result.Views)),
		Virality:    utils.GetVirality(int64(result.Shares), int64(result.Views)),
		PublishDate: publishDate,
		ParsingDate: parsingDate,
	}

	return &resultRow
}

func (s *Service) GetClipsInfoByOwnerID(groupInfo *models.AccountInfo) ([]*models.VKClipInfo, error) {
	s.processingInstagramMu.Lock()
	defer s.processingInstagramMu.Unlock()

	clips := make([]*models.VKClipInfo, 0, groupInfo.Count)
	cursor := ""

	for len(clips) < groupInfo.Count {
		endpoint := fmt.Sprintf("/users/clips?owner_id=chplk:%s", groupInfo.Identification)
		if cursor != "" {
			endpoint += fmt.Sprintf("&cursor=%s", cursor)
		}

		apiResp, err := s.makeAPIRequest(endpoint)
		if err != nil {
			return []*models.VKClipInfo{models.EmptyClipInfo(groupInfo.AccountUrl)},
				fmt.Errorf("failed to fetch clips: %w", err)
		}

		// Если клипов больше нет, выходим
		if len(apiResp.Data.Clips) == 0 {
			break
		}

		// Конвертируем и добавляем клипы
		for _, apiClip := range apiResp.Data.Clips {
			if len(clips) >= groupInfo.Count {
				break
			}

			clips = append(clips, processVkGroupClipResponse(apiClip, groupInfo.AccountUrl))
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

func (s *Service) makeAPIRequest(endpoint string) (*models.GetClipsForGroupAPIResponse, error) {
	const baseURL = "https://vk-scraper.p.rapidapi.com/api/v1"

	if s.rapidAPIKey == "" {
		return nil, fmt.Errorf("API ключ не настроен. Установите RAPIDAPI_KEY")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s%s", baseURL, endpoint), nil)
	if err != nil {
		return nil, err
	}

	// Устанавливаем заголовки
	req.Header.Set("x-rapidapi-key", s.rapidAPIKey)
	req.Header.Set("x-rapidapi-host", "vk-scraper.p.rapidapi.com")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
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

func (s *Service) fetchInstagramDataSafe(reelURL string) (*models.InstagramAPIResponse, error) {
	// Создаем базовый URL
	const baseURL = "https://real-time-instagram-scraper-api1.p.rapidapi.com/v1/media_info"

	if s.rapidAPIKey == "" {
		return nil, fmt.Errorf("API ключ не настроен. Установите RAPIDAPI_KEY")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Создаем запрос с Query параметрами
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	// Добавляем query параметры
	q := req.URL.Query()
	q.Add("code_or_id_or_url", reelURL)
	req.URL.RawQuery = q.Encode()

	s.logger.Info("Запрос к API",
		slog.String("url", req.URL.String()),
	)

	// Устанавливаем заголовки
	req.Header.Set("x-rapidapi-key", s.rapidAPIKey)
	req.Header.Set("x-rapidapi-host", "real-time-instagram-scraper-api1.p.rapidapi.com")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса к API: %v", err)
	}
	defer resp.Body.Close()

	// Обрабатываем ответ
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API вернул код ошибки: %d, тело: %s", resp.StatusCode, string(body))
	}

	var data models.InstagramAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %v, url: %s", err, reelURL)
	}

	return &data, nil
}
