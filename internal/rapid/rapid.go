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

type vkParser interface {
	GetClipInfoByURL(ctx context.Context, url string) (*models.ClipInfo, error)
}
type Service struct {
	rapidAPIKey   string
	logger        *slog.Logger
	httpClient    *http.Client
	sheetsService *google_sheet.Service
	processingMu  sync.Mutex
	vkParser      vkParser
}

func NewService(
	rapidApiKey string,
	log *slog.Logger,
	sheetsService *google_sheet.Service,
	vkParser vkParser,
) *Service {
	return &Service{
		logger:        log,
		rapidAPIKey:   rapidApiKey,
		sheetsService: sheetsService,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		vkParser: vkParser,
	}
}

func (s *Service) ParseUrl(spreadsheetID string, reelUrl []string) []*models.ResultRow {
	s.processingMu.Lock()
	defer s.processingMu.Unlock()

	const batchSize = 50
	results := make([]*models.ResultRow, 0, len(reelUrl))

	tracker, err := google_sheet.NewProgressTracker(s.sheetsService.SheetsService, spreadsheetID)
	if err != nil {
		s.logger.Error("Error creating progress tracker", err)
	}

	var progressRow int
	if tracker != nil {
		progressRow, err = tracker.StartParsing(len(reelUrl))
		if err != nil {
			s.logger.Error("Error starting progress tracking", err)
		}
	}

	processedCount := 0

	defer func() {
		if tracker != nil {
			if err := tracker.FinishParsing(progressRow); err != nil {
				s.logger.Error("Error finishing progress tracking", err)
			}
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

		if tracker != nil {
			if err := tracker.UpdateProgress(progressRow, processedCount); err != nil {
				s.logger.Error("Error updating progress", err)
			}
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
		Shares:      int64(result.Reposts),
		ER:          getER(int64(result.Likes), int64(result.Reposts), int64(result.Comments), int64(result.Views)),
		Virality:    getVirality(int64(result.Reposts), int64(result.Views)),
		PublishDate: publishDate,
		ParsingDate: parsingDate,
	}

	return &resultRow
}

func (s *Service) fetchInstagramDataSafe(reelURL string) (*models.InstagramAPIResponse, error) {
	if s.rapidAPIKey == "" {
		return nil, fmt.Errorf("API ключ не настроен. Установите RAPIDAPI_KEY")
	}

	// Создаем базовый URL
	const baseURL = "https://real-time-instagram-scraper-api1.p.rapidapi.com/v1/media_info"

	// Создаем запрос с Query параметрами
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	// Добавляем query параметры
	q := req.URL.Query()
	q.Add("code_or_id_or_url", reelURL)
	req.URL.RawQuery = q.Encode()

	log.Printf("Запрос к API: %s\n", req.URL.String())

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
