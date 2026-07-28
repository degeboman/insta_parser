package google_sheet

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"inst_parser/internal/config"

	"golang.org/x/oauth2/google"
	"golang.org/x/time/rate"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Repository struct {
	SheetsService *sheets.Service
	limiter       *rate.Limiter
}

const credentialsPath = "credentials.json"

type credentials struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
	UniverseDomain          string `json:"universe_domain"`
}

func NewRepository(cfg config.GoogleDriveCredentials) *Repository {
	if err := createCredentialsFile(cfg); err != nil {
		log.Fatal(err)
	}

	srv, err := getSheetService()
	if err != nil {
		log.Fatal(err)
	}

	return &Repository{
		SheetsService: srv,
		limiter:       rate.NewLimiter(rate.Every(time.Second), 1),
	}
}

func (r *Repository) InsertData(
	spreadsheetID,
	sheetName,
	rangeData string,
	data [][]interface{},
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	if err := r.limiter.Wait(ctx); err != nil {
		return err
	}

	if data == nil {
		return nil
	}

	valueRange := &sheets.ValueRange{
		Values: data,
	}

	_, err := r.SheetsService.Spreadsheets.Values.Append(
		spreadsheetID,
		fmt.Sprintf("%s!%s", sheetName, rangeData),
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("failed to insert data: %v", err)
	}

	return nil
}

// InsertDataWithClear очищает диапазон и вставляет новые данные.
// Если rangeData пустая строка, очищается весь лист.
func (r *Repository) InsertDataWithClear(
	spreadsheetID,
	sheetName,
	rangeData string,
	data [][]interface{},
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	if err := r.limiter.Wait(ctx); err != nil {
		return err
	}

	if data == nil || len(data) == 0 {
		// Если данных нет, просто очищаем (если нужно)
		return r.clearRange(ctx, spreadsheetID, sheetName, rangeData)
	}

	// 1. Очищаем диапазон
	if err := r.clearRange(ctx, spreadsheetID, sheetName, rangeData); err != nil {
		return fmt.Errorf("failed to clear range: %v", err)
	}

	// 2. Записываем новые данные через Update (перезапись с начала диапазона)
	valueRange := &sheets.ValueRange{
		Values: data,
	}

	// Определяем стартовую ячейку: если rangeData указан, используем его,
	// иначе начинаем с "A1" (или можно взять из sheetName)
	startCell := "A1"
	if rangeData != "" {
		// rangeData может быть вида "A1:B10" – возьмём только начало
		// Для простоты используем как есть, но лучше парсить
		startCell = rangeData
	}
	fullRange := fmt.Sprintf("%s!%s", sheetName, startCell)

	_, err := r.SheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		fullRange,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return fmt.Errorf("failed to update data: %v", err)
	}

	return nil
}

// clearRange очищает указанный диапазон на листе.
// Если rangeData пусто, очищает весь лист.
func (r *Repository) clearRange(ctx context.Context, spreadsheetID, sheetName, rangeData string) error {

	// Очистка всего листа – нужно получить ID листа и использовать batchUpdate
	// или очистить весь диапазон "SheetName!A1:ZZZ". Упростим: используем большой диапазон.
	rangeData = "A3:ZZZ" // Не идеально, но работает для типовых случаев.

	fullRange := fmt.Sprintf("%s!%s", sheetName, rangeData)
	_, err := r.SheetsService.Spreadsheets.Values.Clear(
		spreadsheetID,
		fullRange,
		&sheets.ClearValuesRequest{},
	).Do()
	return err
}

func (r *Repository) InsertDataV2(
	spreadsheetID,
	sheetName,
	rangeData string,
	data [][]interface{},
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	if err := r.limiter.Wait(ctx); err != nil {
		return err
	}

	if data == nil {
		return nil
	}

	valueRange := &sheets.ValueRange{
		Values: data,
	}

	_, err := r.SheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		fmt.Sprintf("%s!%s", sheetName, rangeData),
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("failed to insert data: %v", err)
	}

	return nil
}

func getSheetService() (*sheets.Service, error) {
	ctx := context.Background()

	// Чтение файла с credentials
	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл credentials: %v", err)
	}

	// Создание конфигурации
	jwtconfig, err := google.JWTConfigFromJSON(data, sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга credentials: %v", err)
	}

	// Создание клиента
	client := jwtconfig.Client(ctx)

	// Создание сервиса Sheets
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания Sheets сервиса: %v", err)
	}

	return srv, nil
}

func createCredentialsFile(cfg config.GoogleDriveCredentials) (err error) {
	jsonData, err := json.Marshal(credentials{
		Type:                    cfg.Type,
		ProjectID:               cfg.GDProjectID,
		PrivateKeyID:            cfg.PrivateKeyID,
		PrivateKey:              cfg.PrivateKey,
		ClientEmail:             cfg.ClientEmail,
		ClientID:                cfg.ClientID,
		AuthURI:                 cfg.AuthURI,
		TokenURI:                cfg.TokenURI,
		AuthProviderX509CertURL: cfg.AuthProviderX509CertURL,
		ClientX509CertURL:       cfg.ClientX509CertURL,
		UniverseDomain:          cfg.UniverseDomain,
	})
	if err != nil {
		return err
	}

	file, err := os.Create(credentialsPath)
	if err != nil {
		return err
	}

	_, err = file.Write(jsonData)
	if err != nil {
		return err
	}

	return nil
}
