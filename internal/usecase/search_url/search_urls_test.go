package search_url

import (
	"inst_parser/internal/models"
	"log/slog"
	"reflect"
	"testing"

	"inst_parser/internal/config"
	"inst_parser/internal/logger"

	"google.golang.org/api/sheets/v4"
)

func TestUrlsService_SheetIDByName(t *testing.T) {
	type fields struct {
		sheetsService *sheets.Service
		spreadsheetID string
	}
	type args struct {
		name string
	}

	cfg := config.MustLoadForTest()
	srv := NewService(cfg.GoogleDriveCredentials)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "case 1",
			fields: fields{
				sheetsService: srv.SheetsService,
				spreadsheetID: "1ogSt0VDKj-0Ajuz8U7J0gxs33BoozIWvizffl1z16-E",
			},
			args: args{
				name: "данные",
			},
			want:    1743851130,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UrlsService{
				sheetsService: tt.fields.sheetsService,
			}
			got, err := s.SheetIDByName(tt.fields.spreadsheetID, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("SheetIDByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SheetIDByName() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUrlsService_FindColumns(t *testing.T) {
	type fields struct {
		spreadsheetID string
		sheetsService *sheets.Service
		urlSearchWord string
	}
	type args struct {
		sheetName string
	}
	cfg := config.MustLoadForTest()
	srv := NewService(cfg.GoogleDriveCredentials)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *models.ColumnPositions
		wantErr bool
	}{
		{
			name: "case 1",
			fields: fields{
				spreadsheetID: "1ogSt0VDKj-0Ajuz8U7J0gxs33BoozIWvizffl1z16-E",
				sheetsService: srv.SheetsService,
				urlSearchWord: "видео",
			},
			args: args{
				sheetName: "данные",
			},
			want: &models.ColumnPositions{
				URLColumnIndex:      1,
				CheckboxColumnIndex: 2,
			},
			wantErr: false,
		},
		{
			name: "case 2",
			fields: fields{
				spreadsheetID: "1ogSt0VDKj-0Ajuz8U7J0gxs33BoozIWvizffl1z16-E",
				sheetsService: srv.SheetsService,
				urlSearchWord: "видео",
			},
			args: args{
				sheetName: "🟢 Сет 2 // декабрь",
			},
			want: &models.ColumnPositions{
				URLColumnIndex:      14,
				CheckboxColumnIndex: -1,
			},
			wantErr: false,
		},
		{
			name: "case 3",
			fields: fields{
				spreadsheetID: "1J-_Ka6O8EGWjwbsHxOxdve-H2CFPUXTIeV7phAOlK-8",
				sheetsService: srv.SheetsService,
				urlSearchWord: "аккаунт",
			},
			args: args{
				sheetName: "🟡 Парсинг аккаунтов",
			},
			want: &models.ColumnPositions{
				URLColumnIndex:      1,
				CheckboxColumnIndex: 2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UrlsService{
				log:           logger.NewLogger(),
				sheetsService: tt.fields.sheetsService,
			}
			got, err := s.findColumns(tt.fields.spreadsheetID, tt.args.sheetName, tt.fields.urlSearchWord)
			if (err != nil) != tt.wantErr {
				t.Errorf("findColumns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findColumns() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUrlsService_GetUrls(t *testing.T) {
	type fields struct {
		log           *slog.Logger
		spreadsheetID string
		sheetsService *sheets.Service
	}
	type args struct {
		sheetName    string
		positions    *models.ColumnPositions
		parsingTypes []models.ParsingType
	}

	cfg := config.MustLoadForTest()
	l := logger.NewLogger()
	srv := NewService(cfg.GoogleDriveCredentials)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "case 1",
			fields: fields{
				log:           l,
				spreadsheetID: "1ogSt0VDKj-0Ajuz8U7J0gxs33BoozIWvizffl1z16-E",
				sheetsService: srv.SheetsService,
			},
			args: args{
				sheetName: "данные",
				positions: &models.ColumnPositions{
					URLColumnIndex:      1,
					CheckboxColumnIndex: 2,
				},
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "case 2",
			fields: fields{
				log:           l,
				spreadsheetID: "1J-_Ka6O8EGWjwbsHxOxdve-H2CFPUXTIeV7phAOlK-8",
				sheetsService: srv.SheetsService,
			},
			args: args{
				sheetName: "🟡 Парсинг аккаунтов",
				positions: &models.ColumnPositions{
					URLColumnIndex:      1,
					CheckboxColumnIndex: 2,
				},
				parsingTypes: []models.ParsingType{models.VKParsingType},
			},
			want:    []string{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UrlsService{
				log:           tt.fields.log,
				sheetsService: tt.fields.sheetsService,
			}
			got, err := s.GetUrls(tt.fields.spreadsheetID, tt.args.sheetName, tt.args.positions, tt.args.parsingTypes)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUrls() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUrls() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUrlsService_FindUrls(t *testing.T) {
	type fields struct {
		log           *slog.Logger
		spreadsheetID string
		sheetsService *sheets.Service
	}
	type args struct {
		isSelected   bool
		parsingTypes []models.ParsingType
		sheetName    string
	}
	l := logger.NewLogger()
	cfg := config.MustLoadForTest()
	srv := NewService(cfg.GoogleDriveCredentials)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "case 1",
			fields: fields{
				log:           l,
				spreadsheetID: "1ogSt0VDKj-0Ajuz8U7J0gxs33BoozIWvizffl1z16-E",
				sheetsService: srv.SheetsService,
			},
			args: args{
				isSelected:   true,
				parsingTypes: []models.ParsingType{models.InstagramParsingType},
				sheetName:    "данные",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &UrlsService{
				log:           tt.fields.log,
				sheetsService: tt.fields.sheetsService,
			}
			got, err := s.FindUrls(tt.args.isSelected, tt.args.parsingTypes, tt.fields.spreadsheetID, tt.args.sheetName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindUrls() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindUrls() got = %v, want %v", got, tt.want)
			}
		})
	}
}
