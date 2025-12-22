package google_sheet

import (
	logger2 "inst_parser/internal/logger"
	"inst_parser/internal/models"
	"testing"

	"inst_parser/internal/config"
	"inst_parser/internal/rapid"
)

func TestService_InsertData(t *testing.T) {
	type args struct {
		spreadsheetID string
		sheetName     string
		data          []*models.ResultRow
	}

	cfg := config.MustLoad()
	l := logger2.NewLogger()

	srv := NewService(cfg.GoogleDriveCredentials)

	rapidSrv := rapid.NewService(cfg.Rapid.ApiKey, l)

	data := rapidSrv.ParseUrl([]string{"https://www.instagram.com/reel/DR_5rNfjpeb/"})

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				spreadsheetID: "1ogSt0VDKj-0Ajuz8U7J0gxs33BoozIWvizffl1z16-E",
				sheetName:     "Сырые данные",
				data:          data,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := srv
			if err := s.InsertData(tt.args.spreadsheetID, tt.args.sheetName, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("InsertData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
