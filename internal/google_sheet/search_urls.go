package google_sheet

import (
	"fmt"
	"log/slog"
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

func (s *UrlsService) FindUrls(isSelected bool, parsingTypes []models.ParsingType, sheetName, spreadsheetID string) ([]string, error) {
	columnsPositions, err := s.FindColumns(spreadsheetID, sheetName)
	if err != nil {
		return nil, err
	}

	if !isSelected {
		columnsPositions.CheckboxColumnIndex = -1
	}

	return s.GetUrls(spreadsheetID, sheetName, columnsPositions, parsingTypes)
}

func (s *UrlsService) FindColumns(spreadsheetID, sheetName string) (*models.ColumnPositions, error) {
	// Получаем вторую строку (строка 2 в Sheets соответствует индексу 1)
	readRange := fmt.Sprintf("%s!2:2", sheetName)
	resp, err := s.sheetsService.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных листа: %w", err)
	}

	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("вторая строка листа пуста")
	}

	headerRow := resp.Values[0]
	positions := &models.ColumnPositions{
		URLColumnIndex:      -1,
		CheckboxColumnIndex: -1,
	}

	// Ищем колонки
	for i, cell := range headerRow {
		cellValue, ok := cell.(string)
		if !ok {
			continue
		}

		lowerValue := strings.ToLower(strings.TrimSpace(cellValue))

		// Поиск колонки "Ссылка на видео"
		if strings.Contains(lowerValue, "ссылка") && strings.Contains(lowerValue, "видео") {
			positions.URLColumnIndex = i + 1 // Преобразуем в 1-based индекс
			s.log.Info(fmt.Sprintf("Найдена колонка \"Ссылка на видео\" в позиции: %d", positions.URLColumnIndex))
		}

		// Поиск колонки "Парсинг" или "Select"
		if lowerValue == "парсинг" {
			positions.CheckboxColumnIndex = i + 1 // Преобразуем в 1-based индекс
			s.log.Info(fmt.Sprintf("Найдена колонка \"%s\" в позиции: %d", cellValue, positions.CheckboxColumnIndex))
		}
	}

	// Проверяем, найдена ли обязательная колонка
	if positions.URLColumnIndex == -1 {
		return nil, fmt.Errorf(
			"не найдена колонка \"Ссылка на видео\" во второй строке. " +
				"Убедитесь, что:\n1. Вы находитесь на нужном листе" +
				"2. Во второй строке есть заголовок \"Ссылка на видео\"",
		)
	}

	return positions, nil
}

func (s *UrlsService) GetUrls(spreadsheetID, sheetName string, positions *models.ColumnPositions, parsingTypes []models.ParsingType) ([]string, error) {
	if positions == nil {
		return nil, fmt.Errorf("positions не может быть nil")
	}

	if positions.URLColumnIndex <= 0 {
		return nil, fmt.Errorf("некорректный индекс колонки URL: %d", positions.URLColumnIndex)
	}

	var readRange string
	if positions.CheckboxColumnIndex > 0 {
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
		return nil, fmt.Errorf("ошибка чтения данных: %w", err)
	}

	if len(resp.Values) == 0 {
		return []string{}, nil
	}

	var urls []string

	// Конвертируем 1-based индексы в 0-based для работы с массивом
	urlColIndex := positions.URLColumnIndex - 1
	checkboxColIndex := -1
	if positions.CheckboxColumnIndex > 0 {
		checkboxColIndex = positions.CheckboxColumnIndex - 1
	}

	// Обрабатываем каждую строку
	for rowIndex, row := range resp.Values {
		// Проверяем, что в строке достаточно данных
		if urlColIndex >= len(row) {
			s.log.Info("Предупреждение: строка %d не содержит колонку URL", slog.Int("row", rowIndex+3))
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

		url = strings.TrimSpace(url)

		if isAvailableByParsingType(url, parsingTypes) {
			// Добавляем URL в результат
			urls = append(urls, url)
		}
	}

	s.log.Info(fmt.Sprintf("Найдено %d URL", len(urls)))
	return urls, nil
}

func isAvailableByParsingType(url string, parsingTypes []models.ParsingType) bool {
	for _, parsingType := range parsingTypes {
		if !strings.Contains(url, string(parsingType)) {
			return false
		}
	}

	return true
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
