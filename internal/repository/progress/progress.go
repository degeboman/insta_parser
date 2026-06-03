package progress

import (
	"fmt"
	"log"
	"time"

	"inst_parser/internal/constants"

	"google.golang.org/api/sheets/v4"
)

const (
	headerRow  = 1
	maxLogRows = 10
)

type Tracker struct {
	sheetsService *sheets.Service
}

func NewProgressTracker(sheetsService *sheets.Service) *Tracker {
	return &Tracker{
		sheetsService: sheetsService,
	}
}

func (pt *Tracker) EnsureProgressSheet(spreadsheetID string) error {
	spreadsheet, err := pt.sheetsService.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	sheetExists := false
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == constants.ProgressTable {
			sheetExists = true
			break
		}
	}

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

		if err := pt.writeHeaders(spreadsheetID); err != nil {
			return err
		}
	}

	return nil
}

func (pt *Tracker) writeHeaders(spreadsheetID string) error {
	headers := []interface{}{
		"Имя таблицы", "Начало парсинга", "Всего ссылок", "Обработано", "Конец парсинга", "Статус", "Ошибка",
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{headers},
	}

	rangeStr := fmt.Sprintf("%s!A%d:G%d", constants.ProgressTable, headerRow, headerRow)
	_, err := pt.sheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	return err
}

func (pt *Tracker) shiftRowsDown(spreadsheetID string, sheetID int64) error {
	rangeStr := fmt.Sprintf("%s!A2:G%d", constants.ProgressTable, headerRow+maxLogRows)
	resp, err := pt.sheetsService.Spreadsheets.Values.Get(spreadsheetID, rangeStr).Do()
	if err != nil {
		return fmt.Errorf("failed to read existing rows: %w", err)
	}

	var requests []*sheets.Request

	if len(resp.Values) >= maxLogRows {
		lastRow := int64(headerRow + maxLogRows)
		requests = append(requests, &sheets.Request{
			DeleteDimension: &sheets.DeleteDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "ROWS",
					StartIndex: lastRow - 1,
					EndIndex:   lastRow,
				},
			},
		})
	}

	requests = append(requests, &sheets.Request{
		InsertDimension: &sheets.InsertDimensionRequest{
			Range: &sheets.DimensionRange{
				SheetId:    sheetID,
				Dimension:  "ROWS",
				StartIndex: 1,
				EndIndex:   2,
			},
			InheritFromBefore: false,
		},
	})

	_, err = pt.sheetsService.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	return err
}

func (pt *Tracker) getSheetID(spreadsheetID string) (int64, error) {
	spreadsheet, err := pt.sheetsService.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return 0, fmt.Errorf("failed to get spreadsheet: %w", err)
	}
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == constants.ProgressTable {
			return sheet.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("sheet %q not found", constants.ProgressTable)
}

func (pt *Tracker) StartParsing(spreadsheetID, tableName string, totalURLs int) (int, error) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
		moscow = time.Local
	}
	startTime := time.Now().In(moscow).Format(time.DateTime)

	sheetID, err := pt.getSheetID(spreadsheetID)
	if err != nil {
		return 0, err
	}

	if err := pt.shiftRowsDown(spreadsheetID, sheetID); err != nil {
		return 0, fmt.Errorf("failed to shift rows: %w", err)
	}

	row := []interface{}{tableName, startTime, totalURLs, 0, "", "🟡", ""}
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{row},
	}

	rangeStr := fmt.Sprintf("%s!A2:G2", constants.ProgressTable)
	_, err = pt.sheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	if err != nil {
		return 0, fmt.Errorf("failed to start parsing progress: %w", err)
	}

	return 2, nil
}

func (pt *Tracker) UpdateProgress(spreadsheetID string, count, row, progress int) (err error) {
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{progress}},
	}

	if count != 0 {
		value := &sheets.ValueRange{
			Values: [][]interface{}{{count}},
		}

		rangeStr := fmt.Sprintf("%s!C%d", constants.ProgressTable, row)
		_, err = pt.sheetsService.Spreadsheets.Values.Update(
			spreadsheetID,
			rangeStr,
			value,
		).ValueInputOption("RAW").Do()
	}

	rangeStr := fmt.Sprintf("%s!D%d", constants.ProgressTable, row)
	_, err = pt.sheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	return err
}

// FinishParsing завершает запись: проставляет время окончания, статус 🟢.
// Если errMsg не пустой — статус 🔴 и сообщение записывается в столбец G.
func (pt *Tracker) FinishParsing(spreadsheetID string, row int, errMsg string) error {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Printf("Warning: could not load Moscow timezone, using local: %v", err)
		moscow = time.Local
	}
	endTime := time.Now().In(moscow).Format(time.DateTime)

	status := "🟢"
	if errMsg != "" {
		status = "🔴"
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{endTime, status, errMsg}},
	}

	// E — конец парсинга, F — статус, G — ошибка
	rangeStr := fmt.Sprintf("%s!E%d:G%d", constants.ProgressTable, row, row)
	_, err = pt.sheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("RAW").Do()

	return err
}
