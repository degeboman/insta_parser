package progress

import (
	"fmt"
	"log"
	"time"

	"inst_parser/internal/constants"

	"google.golang.org/api/sheets/v4"
)

const headerRow = 1

type Tracker struct {
	sheetsService *sheets.Service
}

func NewProgressTracker(sheetsService *sheets.Service) *Tracker {
	return &Tracker{
		sheetsService: sheetsService,
	}
}

func (pt *Tracker) EnsureProgressSheet(spreadsheetID string) error {
	// Получаем информацию о таблице
	spreadsheet, err := pt.sheetsService.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	// Проверяем, существует ли лист прогресса
	sheetExists := false
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == constants.ProgressTable {
			sheetExists = true
			break
		}
	}

	// Если лист не существует, создаем его
	if !sheetExists {
		req := &sheets.Request{
			AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{
					Title: constants.ProgressTable,
				},
			},
		}

		batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{req},
		}

		_, err := pt.sheetsService.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Do()
		if err != nil {
			return fmt.Errorf("failed to create progress sheet: %w", err)
		}

		// Добавляем заголовки
		if err := pt.writeHeaders(spreadsheetID); err != nil {
			return err
		}
	}

	return nil
}

func (pt *Tracker) writeHeaders(spreadsheetID string) error {
	headers := []interface{}{"Начало парсинга", "Всего ссылок", "Обработано", "Конец парсинга"}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{headers},
	}

	rangeStr := fmt.Sprintf("%s!A%d:D%d", constants.ProgressTable, headerRow, headerRow)
	_, err := pt.sheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	return err
}

func (pt *Tracker) StartParsing(spreadsheetID string, totalURLs int) (int, error) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
		moscow = time.Local
	}
	startTime := time.Now().In(moscow).Format(time.DateTime)

	row := []interface{}{startTime, totalURLs, 0, ""}
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{row},
	}

	// Добавляем новую строку после заголовка
	rangeStr := fmt.Sprintf("%s!A2:D2", constants.ProgressTable)
	_, err = pt.sheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	if err != nil {
		return 0, fmt.Errorf("failed to start parsing progress: %w", err)
	}

	return 2, nil // Возвращаем номер строки
}

func (pt *Tracker) UpdateProgress(spreadsheetID string, row, progress int) error {
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{progress}},
	}

	rangeStr := fmt.Sprintf("%s!C%d", constants.ProgressTable, row)
	_, err := pt.sheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	return err
}

func (pt *Tracker) FinishParsing(spreadsheetID string, row int) error {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
		moscow = time.Local
	}
	endTime := time.Now().In(moscow).Format(time.DateTime)

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{endTime}},
	}

	rangeStr := fmt.Sprintf("%s!D%d", constants.ProgressTable, row)
	_, err = pt.sheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	return err
}
