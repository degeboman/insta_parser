package search_url

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"inst_parser/internal/models"

	"google.golang.org/api/sheets/v4"
)

type UrlsService struct {
	log           *slog.Logger
	sheetsService *sheets.Service
}

func NewUrlsService(log *slog.Logger, sheetsService *sheets.Service) *UrlsService {
	return &UrlsService{log: log, sheetsService: sheetsService}
}

func (s *UrlsService) SheetIDByName(spreadsheetID, name string) (int64, error) {
	spreadsheet, err := s.sheetsService.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return 0, fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == name {
			return sheet.Properties.SheetId, nil
		}
	}

	return 0, fmt.Errorf("could not find sheet with name %s", name)
}

func (s *UrlsService) FindUrls(isSelected bool, parsingTypes []models.ParsingType, sheetName, spreadsheetID string) ([]*models.UrlInfo, error) {
	const urlSearchWord = "видео"

	columnsPositions, err := s.findColumns(spreadsheetID, sheetName, urlSearchWord)
	if err != nil {
		return nil, err
	}

	if !isSelected {
		columnsPositions.CheckboxColumnIndex = -1
	}

	return s.GetUrls(spreadsheetID, sheetName, columnsPositions, parsingTypes)
}

func (s *UrlsService) AccountUrls(
	isSelected bool,
	sheetName, spreadsheetID string,
) ([]*models.UrlInfo, error) {
	const urlSearchWord = "аккаунт"

	columnsPositions, err := s.findColumns(spreadsheetID, sheetName, urlSearchWord)
	if err != nil {
		return nil, err
	}

	if !isSelected {
		columnsPositions.CheckboxColumnIndex = -1
	}

	return s.GetUrls(spreadsheetID, sheetName, columnsPositions, []models.ParsingType{models.VKParsingType, models.InstagramParsingType})
}

func (s *UrlsService) findColumns(spreadsheetID, sheetName, urlWord string) (*models.ColumnPositions, error) {
	// Получаем вторую строку (строка 2 в Sheets соответствует индексу 1)
	readRange := fmt.Sprintf("%s!2:2", sheetName)
	resp, err := s.sheetsService.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get values from list: %w", err)
	}

	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("вторая строка листа пуста")
	}

	headerRow := resp.Values[0]
	positions := &models.ColumnPositions{
		URLColumnIndex:      -1,
		CheckboxColumnIndex: -1,
		CountColumnIndex:    -1,
	}

	// Ищем колонки
	for i, cell := range headerRow {
		if positions.URLColumnIndex != -1 &&
			positions.CheckboxColumnIndex != -1 &&
			positions.CountColumnIndex != -1 {
			break
		}

		cellValue, ok := cell.(string)
		if !ok {
			continue
		}

		lowerValue := strings.ToLower(strings.TrimSpace(cellValue))

		// Поиск колонки "Ссылка на видео"
		if strings.Contains(lowerValue, "ссылка") && strings.Contains(lowerValue, urlWord) {
			positions.URLColumnIndex = i + 1 // Преобразуем в 1-based индекс
			s.log.Info(fmt.Sprintf("Найдена колонка \"Ссылка на %s\" в позиции: %d", urlWord, positions.URLColumnIndex))
		}

		// Поиск колонки "Парсинг" или "Select"
		if strings.Contains(lowerValue, "парсинг") {
			positions.CheckboxColumnIndex = i + 1 // Преобразуем в 1-based индекс
			s.log.Info(fmt.Sprintf("Найдена колонка \"%s\" в позиции: %d", cellValue, positions.CheckboxColumnIndex))
		}

		// Поиск колонки "Парсинг" или "Select"
		if strings.Contains(lowerValue, "глубина") {
			positions.CountColumnIndex = i + 1 // Преобразуем в 1-based индекс
			s.log.Info(fmt.Sprintf("Найдена колонка \"%s\" в позиции: %d", cellValue, positions.CountColumnIndex))
		}
	}

	// Проверяем, найдена ли обязательная колонка
	if positions.URLColumnIndex == -1 {
		return nil, errors.New("failed to find url column")
	}

	return positions, nil
}

func (s *UrlsService) GetUrls(
	spreadsheetID, sheetName string,
	positions *models.ColumnPositions,
	parsingTypes []models.ParsingType,
) ([]*models.UrlInfo, error) {
	if positions == nil {
		return nil, fmt.Errorf("positions cannot be nil")
	}

	if positions.URLColumnIndex <= 0 {
		return nil, fmt.Errorf("invalid url column index URL: %d", positions.URLColumnIndex)
	}

	var readRange string
	if positions.CountColumnIndex > 0 {
		// Если есть колонка чекбокса, читаем диапазон от минимальной до максимальной колонки
		startCol := min(positions.URLColumnIndex, positions.CheckboxColumnIndex)
		endCol := max(positions.CountColumnIndex, positions.CountColumnIndex)

		startLetter := getColumnLetter(startCol)
		endLetter := getColumnLetter(endCol)
		readRange = fmt.Sprintf("%s!%s3:%s", sheetName, startLetter, endLetter)
	} else if positions.CheckboxColumnIndex > 0 {
		// Если есть колонка чекбокса, читаем диапазон от минимальной до максимальной колонки
		startCol := min(positions.URLColumnIndex, positions.CheckboxColumnIndex)
		endCol := max(positions.URLColumnIndex, positions.CheckboxColumnIndex)

		startLetter := getColumnLetter(startCol)
		endLetter := getColumnLetter(endCol)
		readRange = fmt.Sprintf("%s!%s3:%s", sheetName, startLetter, endLetter)
	} else {
		// Получаем только колонку с URL
		colLetter := getColumnLetter(positions.URLColumnIndex)
		readRange = fmt.Sprintf("%s!%s3:%s", sheetName, colLetter, colLetter)
	}

	resp, err := s.sheetsService.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	if len(resp.Values) == 0 {
		return nil, nil
	}

	var urls []*models.UrlInfo

	// Конвертируем 1-based индексы в 0-based для работы с массивом
	urlColIndex := 0
	checkboxColIndex := -1
	countColIndex := -1
	if positions.CheckboxColumnIndex > 0 {
		checkboxColIndex = positions.CheckboxColumnIndex - positions.URLColumnIndex
	}
	if positions.CountColumnIndex > 0 {
		countColIndex = positions.CountColumnIndex - positions.URLColumnIndex
	}

	// Обрабатываем каждую строку
	for rowIndex, row := range resp.Values {
		if len(row) == 0 {
			continue
		}
		// Получаем URL
		urlCell := row[urlColIndex]
		url, ok := urlCell.(string)
		if !ok || strings.TrimSpace(url) == "" {
			// Пропускаем пустые или нестроковые значения
			continue
		}

		// Если есть колонка чекбокса, проверяем её значение
		if checkboxColIndex >= 0 {
			// Проверяем, что колонка чекбокса существует в строке
			if checkboxColIndex >= len(row) {
				s.log.Info("Предупреждение: строка %d не содержит колонку чекбокса", slog.Int("row", rowIndex+3))
				continue
			}

			checkboxCell := row[checkboxColIndex]
			checked, ok := parseCheckboxValue(checkboxCell)
			if !ok {
				s.log.Info("Предупреждение: некорректное значение чекбокса в строке", slog.Int("row", rowIndex+3))
				continue
			}

			// Если чекбокс не отмечен, пропускаем строку
			if !checked {
				continue
			}
		}

		var count string
		if countColIndex >= 0 {
			if countColIndex >= len(row) {
				s.log.Info("Предупреждение: строка %d не содержит колонку глубины", slog.Int("row", rowIndex+3))

			}

			countCell := row[countColIndex]
			count, ok = countCell.(string)
			if !ok {
				// Пропускаем пустые или нестроковые значения
				continue
			}
		}

		url = strings.TrimSpace(url)
		countInt, err := strconv.Atoi(strings.TrimSpace(count))
		if err != nil {
			countInt = 20
		}

		if models.IsAvailableByParsingType(url, parsingTypes) {
			// Добавляем URL в результат
			urls = append(urls, &models.UrlInfo{
				URL:   url,
				Count: countInt,
			})
		}
	}

	s.log.Info("Find urls", slog.Int("count", len(urls)))
	return urls, nil
}

func parseCheckboxValue(cellValue interface{}) (bool, bool) {
	if cellValue == nil {
		return false, true // Пустая ячейка = false
	}

	switch v := cellValue.(type) {
	case bool:
		return v, true
	case string:
		str := strings.ToLower(strings.TrimSpace(v))
		switch str {
		case "true", "истина", "да", "yes", "1", "✓", "✔", "☑":
			return true, true
		case "false", "ложь", "нет", "no", "0", "", "✗", "✘", "☐":
			return false, true
		default:
			return false, false // Некорректное значение
		}
	case float64:
		// Google Sheets возвращает числа как float64
		if v == 1 {
			return true, true
		} else if v == 0 {
			return false, true
		} else {
			return false, false
		}
	default:
		return false, false
	}
}

func getColumnLetter(colNumber int) string {
	if colNumber <= 0 {
		return ""
	}

	letter := ""
	for colNumber > 0 {
		colNumber--
		letter = string(rune('A'+(colNumber%26))) + letter
		colNumber = colNumber / 26
	}
	return letter
}
