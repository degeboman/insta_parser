package google_sheet

import (
	"fmt"
	"log"
	"time"

	"inst_parser/internal/constants"

	"google.golang.org/api/sheets/v4"
)

const (
	ProgressHeaderRow = 1
)

type ProgressTracker struct {
	sheetsService *sheets.Service
	spreadsheetID string
	sheetID       int64
}

func NewProgressTracker(sheetsService *sheets.Service, spreadsheetID string) (*ProgressTracker, error) {
	pt := &ProgressTracker{
		sheetsService: sheetsService,
		spreadsheetID: spreadsheetID,
	}

	if err := pt.ensureProgressSheet(); err != nil {
		return nil, err
	}

	return pt, nil
}

func (pt *ProgressTracker) ensureProgressSheet() error {
	// Получаем информацию о таблице
	spreadsheet, err := pt.sheetsService.Spreadsheets.Get(pt.spreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	// Проверяем, существует ли лист прогресса
	sheetExists := false
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == constants.ProgressTable {
			pt.sheetID = sheet.Properties.SheetId
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

		resp, err := pt.sheetsService.Spreadsheets.BatchUpdate(pt.spreadsheetID, batchUpdateRequest).Do()
		if err != nil {
			return fmt.Errorf("failed to create progress sheet: %w", err)
		}

		pt.sheetID = resp.Replies[0].AddSheet.Properties.SheetId

		// Добавляем заголовки
		if err := pt.writeHeaders(); err != nil {
			return err
		}
	}

	return nil
}

func (pt *ProgressTracker) writeHeaders() error {
	headers := []interface{}{"Начало парсинга", "Всего ссылок", "Обработано", "Конец парсинга"}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{headers},
	}

	rangeStr := fmt.Sprintf("%s!A%d:D%d", constants.ProgressTable, ProgressHeaderRow, ProgressHeaderRow)
	_, err := pt.sheetsService.Spreadsheets.Values.Update(
		pt.spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	return err
}

func (pt *ProgressTracker) StartParsing(totalURLs int) (int, error) {
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
		pt.spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	if err != nil {
		return 0, fmt.Errorf("failed to start parsing progress: %w", err)
	}

	return 2, nil // Возвращаем номер строки
}

func (pt *ProgressTracker) UpdateProgress(row, progress int) error {
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{progress}},
	}

	rangeStr := fmt.Sprintf("%s!C%d", constants.ProgressTable, row)
	_, err := pt.sheetsService.Spreadsheets.Values.Update(
		pt.spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	return err
}

func (pt *ProgressTracker) FinishParsing(row int) error {
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
		pt.spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	return err
}
