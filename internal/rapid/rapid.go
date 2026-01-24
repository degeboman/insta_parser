package rapid

import (
	"context"
	"encoding/json"
	"fmt"
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
		GetClipInfoByURL(ctx context.Context, url string) (*models.ClipInfo, error)
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

func (s *Service) ParseUrl(spreadsheetID string, reelUrl []string) []*models.ResultRow {
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
		s.logger.Error("Error starting progress tracking", errStartParsing)
	}

	processedCount := 0

	defer func() {
		if err := s.trackerService.FinishParsing(spreadsheetID, progressRow); err != nil {
			s.logger.Error("Error finishing progress tracking", err)
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
			s.logger.Error("Error updating progress", err)
		}
	}

	return results
}

func (s *Service) processBatch(urls []string) []*models.ResultRow {
	results := make([]*models.ResultRow, 0, len(urls))

	for _, url := range urls {
		var resultRow *models.ResultRow
		switch models.ParsingTypeByUrl(url) {
		case models.Instagram:
			time.Sleep(550 * time.Millisecond)
			resultRow = s.parseInstagram(url)
		case models.VK:
			time.Sleep(250 * time.Millisecond)
			resultRow = s.parseVK(url)
		default:
			s.logger.Warn("Unsupported URL type", slog.String("url", url))
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
		s.logger.Warn("Error fetching instagram data", err, slog.String("url", url))
		return models.EmptyResultRow(url)
	}

	resultRow, err := processInstagramResponse(data, url)
	if err != nil {
		s.logger.Error("Error processing instagram response", err)
		return nil
	}

	return resultRow
}

func (s *Service) parseVK(url string) *models.ResultRow {
	result, err := s.vkParser.GetClipInfoByURL(context.Background(), url)
	if err != nil {
		s.logger.Warn("Error getting clip info", err)
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
		Views:       int64(result.Views),
		Likes:       int64(result.Likes),
		Comments:    int64(result.Comments),
		Shares:      int64(result.Shares),
		ER:          getER(int64(result.Likes), int64(result.Shares), int64(result.Comments), int64(result.Views)),
		Virality:    getVirality(int64(result.Shares), int64(result.Views)),
		PublishDate: publishDate,
		ParsingDate: parsingDate,
	}

	return &resultRow
}

func (s *Service) GetClipsInfoByOwnerID(groupInfo *models.GroupInfoPair) ([]*models.ClipInfo, error) {
	s.processingInstagramMu.Lock()
	defer s.processingInstagramMu.Unlock()

	clips := make([]*models.ClipInfo, 0, groupInfo.Count)
	cursor := ""

	for len(clips) < groupInfo.Count {
		endpoint := fmt.Sprintf("/users/clips?owner_id=chplk:%s", groupInfo.OwnerID)
		if cursor != "" {
			endpoint += fmt.Sprintf("&cursor=%s", cursor)
		}

		apiResp, err := s.makeAPIRequest(endpoint)
		if err != nil {
			return []*models.ClipInfo{models.EmptyClipInfo(groupInfo.GroupUrl)},
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

			clips = append(clips, processVkGroupClipResponse(apiClip, groupInfo.GroupUrl))
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

func (s *Service) makeAPIRequest(endpoint string) (*models.APIResponse, error) {
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

	var apiResp models.APIResponse
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

func processInstagramResponse(apiResponse *models.InstagramAPIResponse, url string) (*models.ResultRow, error) {
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
	publishDate := ""
	parsingDate := ""
	if item.TakenAt > 0 {
		// Конвертируем Unix timestamp в time.Time
		pubTime := time.Unix(item.TakenAt, 0)

		// Устанавливаем временную зону Москвы
		moscow, err := time.LoadLocation("Europe/Moscow")
		if err != nil {
			log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
			moscow = time.Local
		}

		// Форматируем дату в нужный формат
		pubTimeInMoscow := pubTime.In(moscow)
		publishDate = pubTimeInMoscow.Format("02.01.2006 15:04")
		parsingDate = time.Now().In(moscow).Format("02.01.2006 15:04")
	}

	if shares == nil {
		var nilVal int64 = 0
		shares = &nilVal
	}

	// Создаем строку результата
	result := &models.ResultRow{
		URL:         url,
		Views:       views,
		Likes:       likes,
		Comments:    comments,
		Shares:      *shares,
		ER:          getER(likes, *shares, comments, views),
		Virality:    getVirality(*shares, views),
		ParsingDate: parsingDate,
		PublishDate: publishDate,
	}

	return result, nil
}

func processVkGroupClipResponse(apiResponse models.APIVKClip, url string) *models.ClipInfo {
	// Получаем значения с проверкой на нулевые значения
	likes := apiResponse.Likes.Count
	comments := apiResponse.Comments
	shares := apiResponse.Reposts.Count
	views := apiResponse.Views

	// Форматируем дату публикации
	publishDate := ""
	parsingDate := ""
	if apiResponse.Date > 0 {
		// Конвертируем Unix timestamp в time.Time
		pubTime := time.Unix(int64(apiResponse.Date), 0)

		// Устанавливаем временную зону Москвы
		moscow, err := time.LoadLocation("Europe/Moscow")
		if err != nil {
			log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
			moscow = time.Local
		}

		// Форматируем дату в нужный формат
		pubTimeInMoscow := pubTime.In(moscow)
		publishDate = pubTimeInMoscow.Format("02.01.2006 15:04")
		parsingDate = time.Now().In(moscow).Format("02.01.2006 15:04")
	}

	// Создаем строку результата
	return &models.ClipInfo{
		GroupUrl:    url,
		Description: apiResponse.Description,
		Views:       views,
		Likes:       likes,
		Comments:    comments,
		Shares:      shares,
		ER:          getER(int64(likes), int64(shares), int64(comments), int64(views)),
		Virality:    getVirality(int64(shares), int64(views)),
		ParsingDate: parsingDate,
		PublishDate: publishDate,
		URL:         fmt.Sprintf("https://vk.com/clip%d_%d", apiResponse.OwnerID, apiResponse.ID),
	}
}

func getER(likes, shares, comments, views int64) string {
	if likes+shares+comments <= 0 || views <= 0 {
		return "0"
	}

	return fmt.Sprintf("%.2f%%", float64(likes+shares+comments)/float64(views)*100)
}

func getVirality(shares, views int64) string {
	if shares <= 0 || views <= 0 {
		return "0"
	}
	return fmt.Sprintf("%.2f%%", float64(shares)/float64(views)*100)
}
