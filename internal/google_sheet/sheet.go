package google_sheet

import (
	"context"
	"encoding/json"
	"fmt"
	"inst_parser/internal/models"
	"log"
	"os"

	"inst_parser/internal/config"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	credentialsPath = "credentials.json"
)

type Credentials struct {
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
type Service struct {
	SheetsService *sheets.Service
}

func NewService(cfg config.GoogleDriveCredentials) *Service {
	if err := createCredentialsFile(cfg); err != nil {
		log.Fatal(err)
	}

	srv, err := getSheetService()
	if err != nil {
		log.Fatal(err)
	}

	return &Service{
		SheetsService: srv,
	}
}

func (s *Service) InsertData(spreadsheetID, sheetName string, data []*models.ResultRow) error {
	if data == nil {
		return nil
	}

	values := make([][]interface{}, 0, len(data))

	for _, item := range data {
		if item == nil {
			continue
		}
		rowValues := []interface{}{
			item.URL,
			item.Views,
			item.Likes,
			item.Comments,
			item.Shares,
			item.ER,
			item.Virality,
			item.ParsingDate,
			item.PublishDate,
		}
		values = append(values, rowValues)
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	rangeData := fmt.Sprintf("%s!A:I", sheetName)

	_, err := s.SheetsService.Spreadsheets.Values.Append(
		spreadsheetID,
		rangeData,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("ошибка вставки данных: %v", err)
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
	credentials := Credentials{
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
	}

	jsonData, err := json.Marshal(credentials)
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
